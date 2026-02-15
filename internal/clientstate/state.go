package clientstate

import (
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
)

type State struct {
	ServerURL     string          `json:"server_url"`
	UserID        string          `json:"user_id"`
	JWTToken      string          `json:"jwt_token"`
	ExpiresAt     string          `json:"expires_at"`
	KDFSalt       []byte          `json:"kdf_salt"`
	KDFParams     contract.Params `json:"kdf_params"`
	LastSyncedRev int64           `json:"last_synced_rev"`
}

type Vault struct {
	Store map[string]contract.ItemDTO `json:"vault"`
}
