package service

import (
	"context"
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"go.uber.org/zap"
)

type ItemService struct {
	repo   repository.Items
	logger *zap.SugaredLogger
}

func NewItemService(repo repository.Items, logger *zap.SugaredLogger) *ItemService {
	logger = logger.With("component", "items", "layer", "service")
	return &ItemService{
		repo:   repo,
		logger: logger,
	}
}

func (s *ItemService) SetItem(ctx context.Context, item models.Item) (id uuid.UUID, updatedRev int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	id, updatedRev, err = s.repo.SetItem(ctx, item)
	if err != nil {
		s.logger.Errorw("set/update item error",
			"op", "item.set_item",
			"err", err,
			"item", item.ID,
			"user", item.UserID,
		)
		return uuid.Nil, 0, err
	}

	s.logger.Infow("set item succeeded",
		"op", "item.set_item",
		"item", item.ID,
		"user", item.UserID,
	)

	return id, updatedRev, nil
}

func (s *ItemService) DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (id uuid.UUID, updatedRev int64, err error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	id, updatedRev, err = s.repo.DeleteItem(ctx, itemID, userID)
	if err != nil {
		s.logger.Errorw("delete item error",
			"op", "item.delete_item",
			"err", err,
			"item", itemID,
			"user", userID,
		)
		return uuid.Nil, 0, err
	}

	s.logger.Infow("delete item succeeded",
		"op", "item.delete_item",
		"item", itemID,
		"user", userID,
	)

	return id, updatedRev, nil
}
