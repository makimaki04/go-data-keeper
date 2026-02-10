package dto

import "github.com/makimaki04/go-data-keeper.git/internal/models"

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID        string        `json:"id"`
	JWTToken  string        `json:"jwt_token"`
	ExpiresAt string        `json:"expires_at"`
	KDFSalt   []byte        `json:"kdf_salt"`
	KDFParams models.Params `json:"kdf_params"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	JWTToken  string        `json:"jwt_token"`
	ExpiresAt string        `json:"expires_at"`
	KDFSalt   []byte        `json:"kdf_salt"`
	KDFParams models.Params `json:"kdf_params"`
}
