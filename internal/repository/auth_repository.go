package repository

import (
	"context"
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
	insertUserQuery = `
		INSERT INTO users (login, password_hash)
		VALUES ($1, $2)
		RETURNING id
	`
	selectUserQuery = `
		SELECT id, login, password_hash
		FROM users
		WHERE login = $1
	`
)

type AuthRepository struct {
	db     *sql.DB
	logger *zap.SugaredLogger
}

func NewAuthRepository(db *sql.DB, logger *zap.SugaredLogger) *AuthRepository {
	logger = logger.With("component", "auth", "layer", "repo")

	return &AuthRepository{
		db:     db,
		logger: logger,
	}
}

var (
	ErrUserExists   = errors.New("user already exists")
	ErrUserNotFound = errors.New("user not found")
)

func (r *AuthRepository) RegisterUser(ctx context.Context, user models.User) (uuid.UUID, error) {
	var id uuid.UUID

	tx, err := r.db.BeginTx(ctx, &sql.TxOptions{})
	if err != nil {
		r.logger.Errorw("create user transaction error",
			"op", "auth.user_register",
			"error", err,
		)

		return uuid.Nil, fmt.Errorf("failed to start transaction: %w", err)
	}
	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		} else if err != nil {
			tx.Rollback()

			r.logger.Warnw("transaction rolled back",
				"op", "auth.user_register",
				"user", user.Login,
				"error", err,
			)
		} else {
			if commitErr := tx.Commit(); commitErr != nil {
				r.logger.Errorw("failed to commit transaction",
					"op", "auth.user_register",
					"user", user.Login,
					"error", commitErr,
				)

				err = fmt.Errorf("transaction commit error: %w", commitErr)
			}
		}
	}()

	err = tx.QueryRowContext(ctx, insertUserQuery, user.Login, user.PasswordHash).Scan(&id)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == pgerrcode.UniqueViolation {
			r.logger.Errorw("user already exists",
				"op", "auth.user_register",
				"user", user.Login,
				"error", err,
			)

			return uuid.Nil, ErrUserExists
		}

		r.logger.Errorw("register failed",
			"op", "auth.user_register",
			"user", user.Login,
			"error", err,
		)

		return uuid.Nil, fmt.Errorf("failed to register user %q: %w", user.Login, err)
	}

	return id, nil
}

func (r *AuthRepository) LoginUser(ctx context.Context, login string) (models.User, error) {
	var user models.User

	err := r.db.QueryRowContext(ctx, selectUserQuery, login).Scan(&user.ID, &user.Login, &user.PasswordHash)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			r.logger.Errorw("user not found",
				"op", "auth.user_login",
				"login", login,
				"error", err,
			)

			return models.User{}, ErrUserNotFound
		}

		r.logger.Errorw("db get user error",
			"op", "auth.user_login",
			"login", login,
			"error", err,
		)

		return models.User{}, fmt.Errorf("failed to get user: %s: %w", login, err)
	}

	return user, nil
}
