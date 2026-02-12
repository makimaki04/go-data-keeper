package contract

import (
	"database/sql/driver"
	"encoding/json"
	"fmt"
)

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID        string `json:"id"`
	JWTToken  string `json:"jwt_token"`
	ExpiresAt string `json:"expires_at"`
	KDFSalt   []byte `json:"kdf_salt"`
	KDFParams Params `json:"kdf_params"`
}

type Params struct {
	Algorithm   string `json:"algorithm"`
	KeyLen      uint32 `json:"key_len"`
	SaltLen     uint32 `json:"salt_len"`
	Time        uint32 `json:"time"`
	Memory      uint32 `json:"memory"`
	Parallelism uint32 `json:"parallelism"`
}

func (p Params) Value() (driver.Value, error) {
	b, err := json.Marshal(p)
	if err != nil {
		return nil, fmt.Errorf("marshal Params: %w", err)
	}

	return b, nil
}

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

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	JWTToken  string `json:"jwt_token"`
	ExpiresAt string `json:"expires_at"`
	KDFSalt   []byte `json:"kdf_salt"`
	KDFParams Params `json:"kdf_params"`
}
