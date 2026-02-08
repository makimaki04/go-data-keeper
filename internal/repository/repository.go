package repository

import (
	"context"
	"database/sql"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"go.uber.org/zap"
)

type Authorization interface {
	RegisterUser(ctx context.Context, user models.User) (uuid.UUID, error)
	LoginUser(ctx context.Context, login string) (models.User, error)
}

type Items interface {
	SetItem(ctx context.Context, item models.Item) (id uuid.UUID, updatedRev int64, err error)
	DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (id uuid.UUID, updatedRev int64, err error)
}

type Repository struct {
	Authorization
	Items
}

func NewRepository(db *sql.DB, logger *zap.SugaredLogger) *Repository {
	return &Repository{
		Authorization: NewAuthRepository(db, logger),
		Items: NewItemRepository(db, logger),
	}
}
