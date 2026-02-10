package dto

import (
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
)

type SetItemRequest struct {
	Type       string `json:"type"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	AAD        []byte `json:"aad,omitempty"`
}

type SetItemResponse struct {
	ID         uuid.UUID `json:"id"`
	UpdatedRev int64     `json:"updated_rev"`
}

type DeleteItemResponse struct {
	ID         uuid.UUID `json:"id"`
	UpdatedRev int64     `json:"updated_rev"`
}

type ItemDTO struct {
	ID         uuid.UUID  `json:"id"`
	Type       string     `json:"type"`
	Ciphertext []byte     `json:"ciphertext,omitempty"`
	Nonce      []byte     `json:"nonce"`
	AAD        []byte     `json:"aad,omitempty"`
	Deleted    bool       `json:"deleted"`
	UpdatedRev int64      `json:"updated_rev"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  *time.Time `json:"updated_at,omitempty"`
}

type GetItemResponse struct {
	Item ItemDTO `json:"item"`
}

type GetAllItemsResponse struct {
	Items []ItemDTO `json:"items"`
}

type SyncItemsResponse struct {
	LatestRev int64     `json:"latest_rev"`
	Items     []ItemDTO `json:"items"`
}

func ConvertToItemsDTO(items []models.Item) []ItemDTO {
	resp := make([]ItemDTO, 0, len(items))

	for _, item := range items {
		var updatedAt *time.Time
		if item.UpdatedAt.Valid {
			t := item.UpdatedAt.Time
			updatedAt = &t
		}

		resp = append(resp, ItemDTO{
			ID:         item.ID,
			Type:       item.Type,
			Ciphertext: item.Ciphertext,
			Nonce:      item.Nonce,
			AAD:        item.AAD,
			Deleted:    item.Deleted,
			UpdatedRev: item.UpdatedRev,
			CreatedAt:  item.CreatedAt,
			UpdatedAt:  updatedAt,
		})
	}

	return resp
}
