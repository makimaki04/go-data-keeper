package service

import (
	"context"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"go.uber.org/zap"
)

type Authorization interface {
	RegisterUser(ctx context.Context, login string, password string) (id uuid.UUID, accessToken JWTToken, err error)
	LoginUser(ctx context.Context, login string, password string) (accessToken JWTToken, err error)
	GenerateToken(id uuid.UUID) (accessToken JWTToken, err error)
}

type Items interface {
	SetItem(ctx context.Context, item models.Item) (id uuid.UUID, updatedRev int64, err error)
	DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (id uuid.UUID, updatedRev int64, err error)
}

type Service struct {
	Authorization
	Items
}

func NewService(repo *repository.Repository, secret string, logger *zap.SugaredLogger) *Service {
	return &Service{
		Authorization: NewAuthService(repo.Authorization, secret, logger),
		Items: NewItemService(repo.Items, logger),
	}
}