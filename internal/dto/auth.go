package dto

type RegisterRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type RegisterResponse struct {
	ID        string `json:"id"`
	JWTToken  string `json:"jwt-token"`
	ExpiresAt string `json:"expires_at"`
}

type LoginRequest struct {
	Login    string `json:"login"`
	Password string `json:"password"`
}

type LoginResponse struct {
	JWTToken  string `json:"jwt-token"`
	ExpiresAt string `json:"expires_at"`
}

