package models

import (
	"database/sql"

	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
	KdfSalt      []byte
	KdfParams    contract.Params
	CreatedAt    time.Time
}

type Item struct {
	ID         uuid.UUID
	UserID     uuid.UUID
	Type       string
	Ciphertext []byte
	Nonce      []byte
	AAD        []byte
	Deleted    bool
	UpdatedRev int64
	CreatedAt  time.Time
	UpdatedAt  sql.NullTime
}

func ConvertToItemsDTO(items []Item) []contract.ItemDTO {
	resp := make([]contract.ItemDTO, 0, len(items))

	for _, item := range items {
		var updatedAt *time.Time
		if item.UpdatedAt.Valid {
			t := item.UpdatedAt.Time
			updatedAt = &t
		}

		resp = append(resp, contract.ItemDTO{
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
