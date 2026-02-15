package repository

import (
	"context"
	"crypto/rand"
	"database/sql"
	"errors"
	"fmt"

	"github.com/google/uuid"
	"github.com/jackc/pgerrcode"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"go.uber.org/zap"
)

const (
	insertItemQuery = `
		WITH next_rev AS (
			UPDATE users
			SET current_rev = current_rev + 1
			WHERE id = $2
			RETURNING current_rev
		)
		INSERT INTO items (id, user_id, type, ciphertext, nonce, aad, deleted, updated_rev, updated_at)
		VALUES(
			$1, 
			$2, 
			$3, 
			$4, 
			$5, 
			$6,
			false,
			(SELECT current_rev FROM next_rev),
			now()
		)
		ON CONFLICT (user_id, id) DO UPDATE 
		SET 
		type = EXCLUDED.type,
		ciphertext = EXCLUDED.ciphertext,
		nonce = EXCLUDED.nonce,
		aad = EXCLUDED.aad,
		deleted = false,
		updated_rev = (SELECT current_rev FROM next_rev),
		updated_at = now()
		RETURNING id, user_id, type, ciphertext, nonce, aad, deleted, updated_rev, created_at, updated_at;
	`
	deleteItemQuery = `
		UPDATE items
		SET
		ciphertext = NULL,
		nonce = $3,
		aad=NULL,
		deleted=true
		WHERE id = $1 AND user_id = $2
		RETURNING id;
	`
	updateUserRevQuery = `
		UPDATE users
		SET
		current_rev = current_rev + 1
		WHERE id = $1
		RETURNING current_rev;
	`
	updateItemRevQuery = `
		UPDATE items
		SET
		updated_rev = $3,
		updated_at = Now()
		WHERE id = $1 AND user_id = $2
		RETURNING updated_rev;
	`
	getItemQuery = `
		SELECT id, user_id, type, ciphertext, nonce, aad, deleted, updated_rev,  created_at, updated_at
		FROM items
		WHERE user_id = $1 AND id = $2 
	`
	getAllItemsQuery = `
		SELECT id, user_id, type, ciphertext, nonce, aad, deleted, updated_rev,  created_at, updated_at
		FROM items
		WHERE user_id = $1
		ORDER BY updated_rev ASC
	`
	getUserLatestRevQuery = `
		SELECT current_rev
		from users
		WHERE id = $1
	`
	getItemsSinceRevQuery = `
		SELECT id, user_id, type, ciphertext, nonce, aad, deleted, updated_rev,  created_at, updated_at
		FROM items
		WHERE user_id = $1 AND updated_rev > $2
		ORDER BY updated_rev ASC
	`
)

// ItemRepository implements item persistence backed by a SQL database.
type ItemRepository struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

// NewItemRepository creates an ItemRepository backed by db.
func NewItemRepository(db *sql.DB, logger *zap.SugaredLogger) *ItemRepository {
	logger = logger.With("component", "items", "layer", "repo")

	return &ItemRepository{
		db:     db,
		logger: logger,
	}
}

var (
	// ErrUserMissing is returned when the referenced user does not exist.
	ErrUserMissing = errors.New("user missing")
	// ErrBadItemType is returned when the item type is invalid.
	ErrBadItemType = errors.New("bad item type")
	// ErrRetryableDB is returned for retryable database errors.
	ErrRetryableDB = errors.New("retryable db error")
	// ErrSchemaMismatch is returned when the database schema is incompatible with the query.
	ErrSchemaMismatch = errors.New("schema mismatch")
	// ErrNotFound is returned when an item can't be found.
	ErrNotFound = errors.New("item not found")
	// ErrDB is returned for non-specific database errors.
	ErrDB = errors.New("db error")
)

// SetItem creates or updates an item and returns the stored record.
// The context controls cancellation and deadlines.
func (r *ItemRepository) SetItem(ctx context.Context, item models.Item) (models.Item, error) {
	var out models.Item
	err := r.db.QueryRowContext(ctx, insertItemQuery,
		item.ID, item.UserID, item.Type,
		item.Ciphertext, item.Nonce, item.AAD,
	).Scan(
		&out.ID,
		&out.UserID,
		&out.Type,
		&out.Ciphertext,
		&out.Nonce,
		&out.AAD,
		&out.Deleted,
		&out.UpdatedRev,
		&out.CreatedAt,
		&out.UpdatedAt,
	)

	if err != nil {
		err = checkErr(err, r.logger, "set_item")
		return models.Item{}, err
	}

	return out, nil
}

// DeleteItem deletes an item and returns the resulting record.
// The context controls cancellation and deadlines.
func (r *ItemRepository) DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (out models.Item, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		r.logger.Errorw("begin delete_item transaction error",
			"op", "delete_item",
			"err", err,
		)

		return models.Item{}, fmt.Errorf("failed to start delete item transaction: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			r.logger.Warnw("transaction rolled back", "op", "delete_item")
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()

			r.logger.Warnw("transaction rolled back",
				"op", "delete_item",
				"error", err,
			)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				r.logger.Errorw("failed to commit transaction",
					"op", "delete_item",
					"error", commitErr,
				)

				err = fmt.Errorf("transaction commit error: %w", commitErr)
			}
		}
	}()

	dummyNonce := make([]byte, 12)
	_, err = rand.Read(dummyNonce)
	if err != nil {
		r.logger.Errorw("rand read dummyNonce error", "op", "delete_item", "err", err)
		dummyNonce = make([]byte, 12)
	}

	var id uuid.UUID
	err = tx.QueryRowContext(ctx, deleteItemQuery, itemID, userID, dummyNonce).Scan(&id)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item.delete_item_query")
		return models.Item{}, err
	}

	var currentRev int64
	err = tx.QueryRowContext(ctx, updateUserRevQuery, userID).Scan(&currentRev)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item")
		return models.Item{}, err
	}

	var updatedRev int64
	err = tx.QueryRowContext(ctx, updateItemRevQuery, id, userID, currentRev).Scan(&updatedRev)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item.update_item_query")
		return models.Item{}, err
	}

	err = tx.QueryRowContext(ctx, getItemQuery, userID, id).Scan(
		&out.ID,
		&out.UserID,
		&out.Type,
		&out.Ciphertext,
		&out.Nonce,
		&out.AAD,
		&out.Deleted,
		&out.UpdatedRev,
		&out.CreatedAt,
		&out.UpdatedAt,
	)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item.get_item_query")
		return models.Item{}, err
	}

	return out, nil
}

// GetItem returns a single item for the given user.
// The context controls cancellation and deadlines.
func (r *ItemRepository) GetItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error) {
	var item models.Item

	err := r.db.QueryRowContext(ctx, getItemQuery, userID, itemID).Scan(
		&item.ID,
		&item.UserID,
		&item.Type,
		&item.Ciphertext,
		&item.Nonce,
		&item.AAD,
		&item.Deleted,
		&item.UpdatedRev,
		&item.CreatedAt,
		&item.UpdatedAt,
	)
	if err != nil {
		err = checkErr(err, r.logger, "get_item")
		return models.Item{}, err
	}

	return item, nil
}

// GetAllItems returns all items for the given user.
// The context controls cancellation and deadlines.
func (r *ItemRepository) GetAllItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	rows, err := r.db.QueryContext(ctx, getAllItemsQuery, userID)
	if err != nil {
		err = checkErr(err, r.logger, "get_all_items.query")
		return nil, err
	}
	defer rows.Close()

	items := make([]models.Item, 0)
	for rows.Next() {
		var item models.Item
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Type,
			&item.Ciphertext,
			&item.Nonce,
			&item.AAD,
			&item.Deleted,
			&item.UpdatedRev,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			err = checkErr(err, r.logger, "get_all_items.scan")
			return nil, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		err = checkErr(err, r.logger, "get_all_items.rows")
		return nil, err
	}

	return items, nil
}

// GetChangesSince returns items updated after the given revision and the latest user revision.
// The context controls cancellation and deadlines.
func (r *ItemRepository) GetChangesSince(ctx context.Context, userID uuid.UUID, since int64) (items []models.Item, latestRev int64, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{
		ReadOnly: true,
	})
	if err != nil {
		r.logger.Errorw("begin get changes transaction error",
			"op", "get_changes_since",
			"err", err,
		)

		return []models.Item{}, 0, fmt.Errorf("failed to start get_changes transaction: %v", err)
	}

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			r.logger.Warnw("transaction rolled back", "op", "get_changes_since")
			panic(p)
		} else if err != nil {
			_ = tx.Rollback()

			r.logger.Warnw("transaction rolled back",
				"op", "get_changes_since",
				"error", err,
			)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				r.logger.Errorw("failed to commit transaction",
					"op", "get_changes_since",
					"error", commitErr,
				)

				err = fmt.Errorf("transaction commit error: %w", commitErr)
			}
		}
	}()

	err = tx.QueryRowContext(ctx, getUserLatestRevQuery, userID).Scan(&latestRev)
	if err != nil {
		err = checkErr(err, r.logger, "get_changes_since")
		return []models.Item{}, 0, err
	}

	if latestRev == 0 {
		r.logger.Infow("user has not revs", "user", userID)
		return []models.Item{}, latestRev, nil
	}

	if since >= latestRev {
		r.logger.Infow("user has not new revs", "user", userID)
		return []models.Item{}, latestRev, nil
	}

	rows, err := tx.QueryContext(ctx, getItemsSinceRevQuery, userID, since)
	if err != nil {
		err = checkErr(err, r.logger, "get_changes_since")
		return []models.Item{}, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var item models.Item
		if err := rows.Scan(
			&item.ID,
			&item.UserID,
			&item.Type,
			&item.Ciphertext,
			&item.Nonce,
			&item.AAD,
			&item.Deleted,
			&item.UpdatedRev,
			&item.CreatedAt,
			&item.UpdatedAt,
		); err != nil {
			err = checkErr(err, r.logger, "get_changes_since")
			return []models.Item{}, 0, err
		}

		items = append(items, item)
	}

	if err := rows.Err(); err != nil {
		err = checkErr(err, r.logger, "get_changes_since.rows")
		return []models.Item{}, 0, err
	}

	return items, latestRev, nil
}

func checkErr(err error, logger *zap.SugaredLogger, op string) error {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		logger.Infow("context error", "op", op, "err", err)
		return err
	}

	if errors.Is(err, sql.ErrNoRows) {
		logger.Infow("item not found", "op", op, "err", err)
		return ErrNotFound
	}

	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		logger.Infow("db query failed (non-pg error)", "op", op, "err", err)
		return ErrDB
	}

	logger.Infow("db query failed",
		"code", pgErr.Code,
		"msg", pgErr.Message,
		"detail", pgErr.Detail,
		"table", pgErr.TableName,
		"column", pgErr.ColumnName,
		"constraint", pgErr.ConstraintName,
	)

	switch pgErr.Code {
	case pgerrcode.ForeignKeyViolation:
		return ErrUserMissing
	case pgerrcode.NotNullViolation:
		return ErrDB
	case pgerrcode.InvalidTextRepresentation:
		return ErrBadItemType
	case pgerrcode.SerializationFailure, pgerrcode.DeadlockDetected, pgerrcode.LockNotAvailable:
		return ErrRetryableDB
	case pgerrcode.InvalidColumnReference:
		return ErrSchemaMismatch
	default:
		return ErrDB
	}
}
