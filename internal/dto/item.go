package dto

import "github.com/google/uuid"

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
