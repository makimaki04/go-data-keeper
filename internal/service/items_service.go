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

func (s *ItemService) SetItem(ctx context.Context, item models.Item) (models.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	out, err := s.repo.SetItem(ctx, item)
	if err != nil {
		s.logger.Errorw("set/update item error",
			"op", "item.set_item",
			"err", err,
			"item", item.ID,
			"user", item.UserID,
		)
		return models.Item{}, err
	}

	s.logger.Infow("set item succeeded",
		"op", "item.set_item",
		"item", item.ID,
		"user", item.UserID,
	)

	return out, nil
}

func (s *ItemService) DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 5*time.Second)
	defer cancel()

	out, err := s.repo.DeleteItem(ctx, itemID, userID)
	if err != nil {
		s.logger.Errorw("delete item error",
			"op", "item.delete_item",
			"err", err,
			"item", itemID,
			"user", userID,
		)
		return models.Item{}, err
	}

	s.logger.Infow("delete item succeeded",
		"op", "item.delete_item",
		"item", itemID,
		"user", userID,
	)

	return out, nil
}

func (s *ItemService) GetItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 4*time.Second)
	defer cancel()

	item, err := s.repo.GetItem(ctx, itemID, userID)
	if err != nil {
		s.logger.Errorw("get item error",
			"op", "item.get_item",
			"err", err,
			"item", itemID,
			"user", userID,
		)
		return models.Item{}, err
	}

	s.logger.Infow("get item succeeded",
		"op", "item.get_item",
		"item", itemID,
		"user", userID,
	)

	return item, nil
}

func (s *ItemService) GetAllItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	items, err := s.repo.GetAllItems(ctx, userID)
	if err != nil {
		s.logger.Errorw("get all items error",
			"op", "item.get_all_items",
			"err", err,
			"user", userID,
		)
		return nil, err
	}

	s.logger.Infow("get all items succeeded",
		"op", "item.get_all_items",
		"user", userID,
		"count", len(items),
	)

	return items, nil
}

func (s *ItemService) GetChangesSince(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error) {
	ctx, cancel := context.WithTimeout(ctx, 10*time.Second)
	defer cancel()

	items, rev, err := s.repo.GetChangesSince(ctx, userID, since)
	if err != nil {
		s.logger.Errorw("get changes since error",
			"op", "item.get_changes_since",
			"err", err,
			"user", userID,
		)

		return []models.Item{}, 0, err
	}

	s.logger.Infow("get chenges since succeeded",
		"op", "item.get_changes_since",
		"user", userID,
		"count", len(items),
	)

	return items, rev, nil
}
