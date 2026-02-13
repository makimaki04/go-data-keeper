package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"go.uber.org/zap"
)

type Authorization interface {
	RegisterUser(ctx context.Context, login string, password string) (AuthData, error)
	LoginUser(ctx context.Context, login string, password string) (AuthData, error)
	GenerateToken(id uuid.UUID) (accessToken JWTToken, err error)
}

type Items interface {
	SetItem(ctx context.Context, item models.Item) (models.Item, error)
	DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error)
	GetItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error)
	GetAllItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error)
	GetChangesSince(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error)
}

type Service struct {
	Authorization
	Items
}

func NewService(repo *repository.Repository, secret string, logger *zap.SugaredLogger) *Service {
	return &Service{
		Authorization: NewAuthService(repo.Authorization, secret, logger),
		Items:         NewItemService(repo.Items, logger),
	}
}
