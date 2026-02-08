package models

import (
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID           uuid.UUID
	Login        string
	PasswordHash string
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
