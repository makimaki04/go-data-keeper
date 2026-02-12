package contract

import (
	"time"

	"github.com/google/uuid"
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