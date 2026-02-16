package clientstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

func newTestStore(t *testing.T) *Store {
	t.Helper()

	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	vaultPath := filepath.Join(dir, "vault.json")

	s, err := NewStore(statePath, vaultPath, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("NewStore() err=%v", err)
	}
	return s
}

func TestStore_LoadVault_WhenMissing_ReturnsEmptyMap(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	v, err := s.LoadVault()
	if err != nil {
		t.Fatalf("LoadVault() err=%v", err)
	}
	if v.Store == nil {
		t.Fatalf("expected non-nil map")
	}
	if len(v.Store) != 0 {
		t.Fatalf("expected empty vault, got len=%d", len(v.Store))
	}
}

func TestStore_SaveLoadVault_RoundTrip(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	id := uuid.New()
	now := time.Now().UTC()

	vIn := Vault{
		Store: map[string]contract.ItemDTO{
			id.String(): {
				ID:         id,
				Type:       "text",
				Ciphertext: []byte("ct"),
				Nonce:      []byte("nonce"),
				AAD:        []byte("aad"),
				Deleted:    false,
				UpdatedRev: 7,
				CreatedAt:  now,
				UpdatedAt:  &now,
			},
		},
	}

	if err := s.SaveVault(vIn); err != nil {
		t.Fatalf("SaveVault() err=%v", err)
	}

	vOut, err := s.LoadVault()
	if err != nil {
		t.Fatalf("LoadVault() err=%v", err)
	}
	if vOut.Store == nil || len(vOut.Store) != 1 {
		t.Fatalf("unexpected store after roundtrip: %#v", vOut.Store)
	}
	got := vOut.Store[id.String()]
	if got.ID != id || got.Type != "text" || got.UpdatedRev != 7 || got.Deleted {
		t.Fatalf("unexpected item after roundtrip: %#v", got)
	}
}

func TestStore_WipeVault_ClearsVault(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	// Put something first.
	v := Vault{Store: map[string]contract.ItemDTO{uuid.New().String(): {Type: "text"}}}
	if err := s.SaveVault(v); err != nil {
		t.Fatalf("SaveVault() err=%v", err)
	}

	if err := s.WipeVault(); err != nil {
		t.Fatalf("WipeVault() err=%v", err)
	}

	out, err := s.LoadVault()
	if err != nil {
		t.Fatalf("LoadVault() err=%v", err)
	}
	if out.Store == nil || len(out.Store) != 0 {
		t.Fatalf("expected empty store after wipe, got %#v", out.Store)
	}
}

func TestStore_SaveLoadState_RoundTrip(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)

	in := State{
		ServerURL:     "http://127.0.0.1:8080",
		UserID:        uuid.New().String(),
		JWTToken:      "tkn",
		ExpiresAt:     "exp",
		KDFSalt:       []byte("1234567890abcdef"),
		KDFParams:     contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
		LastSyncedRev: 42,
	}
	if err := s.SaveState(in); err != nil {
		t.Fatalf("SaveState() err=%v", err)
	}

	out, err := s.LoadState()
	if err != nil {
		t.Fatalf("LoadState() err=%v", err)
	}
	if out.UserID != in.UserID || out.JWTToken != in.JWTToken || out.LastSyncedRev != in.LastSyncedRev {
		t.Fatalf("state mismatch: got=%#v want=%#v", out, in)
	}
}

func TestStore_LoadVault_CorruptJSON_ReturnsError(t *testing.T) {
	t.Parallel()

	s := newTestStore(t)
	if err := os.WriteFile(s.vaultFile.path, []byte("{not-json"), 0600); err != nil {
		t.Fatalf("write corrupt vault: %v", err)
	}
	_, err := s.LoadVault()
	if err == nil {
		t.Fatalf("expected error for corrupt json")
	}
}
