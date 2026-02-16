package contract

import (
	"time"

	"github.com/google/uuid"
)

// SetItemRequest is the payload for creating or updating an item.
type SetItemRequest struct {
	Type       string `json:"type"`
	Ciphertext []byte `json:"ciphertext"`
	Nonce      []byte `json:"nonce"`
	AAD        []byte `json:"aad,omitempty"`
}

// SetItemResponse is the response returned after creating or updating an item.
type SetItemResponse struct {
	Item ItemDTO `json:"item"`
}

// DeleteItemResponse is the response returned after deleting an item.
type DeleteItemResponse struct {
	Item ItemDTO `json:"item"`
}

// GetItemResponse is the response returned when fetching a single item.
type GetItemResponse struct {
	Item ItemDTO `json:"item"`
}

// GetAllItemsResponse is the response returned when listing items.
type GetAllItemsResponse struct {
	Items []ItemDTO `json:"items"`
}

// SyncItemsResponse is the response returned when fetching item changes since a revision.
type SyncItemsResponse struct {
	LatestRev int64     `json:"latest_rev"`
	Items     []ItemDTO `json:"items"`
}

// ItemDTO is an item representation used in API requests and responses.
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
