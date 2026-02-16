// Package contract defines transport DTOs shared by client and server.
package contract

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

// RegisterRequest is the payload for user registration.
type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// RegisterResponse is the response returned after a successful registration.
type RegisterResponse struct {
	UserID    string `json:"id"`
	JWTToken  string `json:"jwt_token"`
	ExpiresAt string `json:"expires_at"`
	KDFSalt   []byte `json:"kdf_salt"`
	KDFParams Params `json:"kdf_params"`
}

// Params describes parameters for the key-derivation function.
type Params struct {
	Algorithm   string `json:"algorithm"`
	KeyLen      uint32 `json:"key_len"`
	SaltLen     uint32 `json:"salt_len"`
	Time        uint32 `json:"time"`
	Memory      uint32 `json:"memory"`
	Parallelism uint32 `json:"parallelism"`
}

// Value implements driver.Valuer for storing Params in a database.
// Value returns an error if Params can't be encoded as JSON.
func (p Params) Value() (driver.Value, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal Params: %w", err)
	}

	return b, nil
}

// Scan implements sql.Scanner for decoding Params from a database value.
// Scan returns an error if the source type is unsupported or JSON decoding fails.
func (p *Params) Scan(src any) error {
	if src == nil {
		*p = Params{}
		return nil
	}

	var b []byte
	switch v := src.(type) {
	case []byte:
		b = v
	case string:
		b = []byte(v)
	default:
		return fmt.Errorf("scan Params: unsupported type %T", src)
	}

	if len(b) == 0 {
		*p = Params{}
		return nil
	}

	if err := json.Unmarshal(b, p); err != nil {
		return fmt.Errorf("unmarshal Params: %w", err)
	}

	return nil
}

// LoginRequest is the payload for user login.
type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

// LoginResponse is the response returned after a successful login.
type LoginResponse struct {
	UserID    string `json:"user_id"`
	Login     string `json:"login"`
	JWTToken  string `json:"jwt_token"`
	ExpiresAt string `json:"expires_at"`
	KDFSalt   []byte `json:"kdf_salt"`
	KDFParams Params `json:"kdf_params"`
}
