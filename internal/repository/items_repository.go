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
		RETURNING id, updated_rev;
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
)

type ItemRepository struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewItemRepository(db *sql.DB, logger *zap.SugaredLogger) *ItemRepository {
	logger = logger.With("component", "items", "layer", "repo")

	return &ItemRepository{
		db:     db,
		logger: logger,
	}
}

var (
	ErrUserMissing    = errors.New("user missing")
	ErrBadItemType    = errors.New("bad item type")
	ErrRetryableDB    = errors.New("retryable db error")
	ErrSchemaMismatch = errors.New("schema mismatch")
	ErrNotFound       = errors.New("item not found")
	ErrDB             = errors.New("db error")
)

func (r *ItemRepository) SetItem(ctx context.Context, item models.Item) (id uuid.UUID, updatedRev int64, err error) {
	err = r.db.QueryRowContext(ctx, insertItemQuery,
		item.ID, item.UserID, item.Type,
		item.Ciphertext, item.Nonce, item.AAD,
	).Scan(&id, &updatedRev)

	if err != nil {
		err = checkErr(err, r.logger, "set_item")
		return uuid.Nil, 0, err
	}

	return id, updatedRev, nil
}

func (r *ItemRepository) DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (id uuid.UUID, updatedRev int64, err error) {
	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		r.logger.Errorw("begin delete_item transaction error",
			"op", "delete_item",
			"err", err,
		)

		return uuid.Nil, 0, fmt.Errorf("failed to start delete item transaction: %v", err)
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

	err = tx.QueryRowContext(ctx, deleteItemQuery, itemID, userID, dummyNonce).Scan(&id)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item.delete_item_query")
		return uuid.Nil, 0, err
	}

	var currentRev int64
	err = tx.QueryRowContext(ctx, updateUserRevQuery, userID).Scan(&currentRev)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item")
		return uuid.Nil, 0, err
	}

	err = tx.QueryRowContext(ctx, updateItemRevQuery, id, userID, currentRev).Scan(&updatedRev)
	if err != nil {
		err = checkErr(err, r.logger, "delete_item.update_item_query")
		return uuid.Nil, 0, err
	}

	return id, updatedRev, err
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
