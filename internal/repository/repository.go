// Package repository provides persistence interfaces and database-backed repositories.
package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"go.uber.org/zap"
)

// Authorization defines persistence operations for user accounts.
type Authorization interface {
	RegisterUser(ctx context.Context, user models.User) (uuid.UUID, error)
	LoginUser(ctx context.Context, login string) (models.User, error)
}

// Items defines persistence operations for encrypted items.
type Items interface {
	SetItem(ctx context.Context, item models.Item) (models.Item, error)
	DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error)
	GetItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error)
	GetAllItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error)
	GetChangesSince(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error)
}

// Repository aggregates persistence interfaces used by the service layer.
type Repository struct {
	Authorization
	Items
}

// NewRepository creates a Repository backed by the provided database connection.
func NewRepository(db *sql.DB, logger *zap.SugaredLogger) *Repository {
	return &Repository{
		Authorization: NewAuthRepository(db, logger),
		Items:         NewItemRepository(db, logger),
	}
}
