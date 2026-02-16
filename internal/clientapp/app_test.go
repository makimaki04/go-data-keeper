package clientapp

import (
	"context"
	"encoding/json"
	"path/filepath"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/clientcrypto"
	"github.com/makimaki04/go-data-keeper.git/internal/clientstate"
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

type fakeClient struct {
	url string

	registerResp contract.RegisterResponse
	registerErr  error

	loginResp contract.LoginResponse
	loginErr  error

	syncResp contract.SyncItemsResponse
	syncErr  error

	lastSyncSince int64
}

func (f *fakeClient) Register(ctx context.Context, login string, password string) (contract.RegisterResponse, error) {
	return f.registerResp, f.registerErr
}
func (f *fakeClient) Login(ctx context.Context, login string, password string) (contract.LoginResponse, error) {
	return f.loginResp, f.loginErr
}
func (f *fakeClient) SetItem(ctx context.Context, itemId string, item contract.SetItemRequest) (contract.SetItemResponse, error) {
	panic("not used in tests")
}
func (f *fakeClient) DeleteItem(ctx context.Context, itemID string) (contract.DeleteItemResponse, error) {
	panic("not used in tests")
}
func (f *fakeClient) SyncChanges(ctx context.Context, lastSyncedRev int64) (contract.SyncItemsResponse, error) {
	f.lastSyncSince = lastSyncedRev
	return f.syncResp, f.syncErr
}
func (f *fakeClient) GetURL() string { return f.url }

func newTestApp(t *testing.T) (*App, *clientstate.Store) {
	t.Helper()

	dir := t.TempDir()
	statePath := filepath.Join(dir, "state.json")
	vaultPath := filepath.Join(dir, "vault.json")
	store, err := clientstate.NewStore(statePath, vaultPath, zap.NewNop().Sugar())
	if err != nil {
		t.Fatalf("NewStore() err=%v", err)
	}

	fc := &fakeClient{url: "http://example"}
	app := NewApp(fc, *store, zap.NewNop().Sugar())
	return app, store
}

func TestLogin_WipesVaultOnUserSwitch(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name        string
		oldUserID   string
		newUserID   string
		wantWiped   bool
		wantLastRev int64
	}{
		{name: "no_old_user_no_wipe", oldUserID: "", newUserID: "u2", wantWiped: false, wantLastRev: 0},
		{name: "same_user_no_wipe", oldUserID: "u1", newUserID: "u1", wantWiped: false, wantLastRev: 0},
		{name: "different_user_wipe", oldUserID: "u1", newUserID: "u2", wantWiped: true, wantLastRev: 0},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			app, store := newTestApp(t)
			fc := app.Client.(*fakeClient)

			// Seed state + vault.
			_ = store.SaveState(clientstate.State{
				ServerURL:     fc.url,
				UserID:        tt.oldUserID,
				JWTToken:      "old",
				ExpiresAt:     "exp",
				KDFSalt:       make([]byte, 16),
				KDFParams:     contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
				LastSyncedRev: 123,
			})
			_ = store.SaveVault(clientstate.Vault{
				Store: map[string]contract.ItemDTO{
					uuid.New().String(): {Type: "text"},
				},
			})

			fc.loginResp = contract.LoginResponse{
				UserID:    tt.newUserID,
				Login:     "alice",
				JWTToken:  "new",
				ExpiresAt: "exp2",
				KDFSalt:   make([]byte, 16),
				KDFParams: contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
			}
			fc.syncResp = contract.SyncItemsResponse{LatestRev: tt.wantLastRev, Items: nil}

			if err := app.Login("alice", "password123"); err != nil {
				t.Fatalf("Login() err=%v", err)
			}

			v, err := store.LoadVault()
			if err != nil {
				t.Fatalf("LoadVault() err=%v", err)
			}
			if tt.wantWiped && len(v.Store) != 0 {
				t.Fatalf("expected wiped vault, got len=%d", len(v.Store))
			}
			if !tt.wantWiped && len(v.Store) == 0 {
				t.Fatalf("expected vault preserved, got empty")
			}

			st, err := store.LoadState()
			if err != nil {
				t.Fatalf("LoadState() err=%v", err)
			}
			if st.UserID != tt.newUserID {
				t.Fatalf("state user mismatch: got=%q want=%q", st.UserID, tt.newUserID)
			}
		})
	}
}

func TestSyncChanges_AppliesAndSanitizesDeleted(t *testing.T) {
	t.Parallel()

	app, store := newTestApp(t)
	fc := app.Client.(*fakeClient)

	_ = store.SaveState(clientstate.State{
		ServerURL:     fc.url,
		UserID:        "u1",
		JWTToken:      "tkn",
		ExpiresAt:     "exp",
		KDFSalt:       make([]byte, 16),
		KDFParams:     contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
		LastSyncedRev: 5,
	})

	existingID := uuid.New()
	_ = store.SaveVault(clientstate.Vault{Store: map[string]contract.ItemDTO{
		existingID.String(): {ID: existingID, Type: "text", Deleted: false, UpdatedRev: 1, Ciphertext: []byte("old"), Nonce: []byte("n")},
	}})

	delID := uuid.New()
	fc.syncResp = contract.SyncItemsResponse{
		LatestRev: 7,
		Items: []contract.ItemDTO{
			{ID: delID, Type: "text", Deleted: true, UpdatedRev: 6, Ciphertext: []byte("ct"), Nonce: []byte("nn"), AAD: []byte("aa"), CreatedAt: time.Now()},
			{ID: existingID, Type: "text", Deleted: false, UpdatedRev: 7, Ciphertext: []byte("new"), Nonce: []byte("n2"), AAD: []byte("a2"), CreatedAt: time.Now()},
		},
	}

	if err := app.SyncChanges(); err != nil {
		t.Fatalf("SyncChanges() err=%v", err)
	}

	v, _ := store.LoadVault()
	gotDel := v.Store[delID.String()]
	if !gotDel.Deleted {
		t.Fatalf("expected deleted item")
	}
	if gotDel.Ciphertext != nil || gotDel.Nonce != nil || gotDel.AAD != nil {
		t.Fatalf("expected deleted item crypto fields nil")
	}
	gotExisting := v.Store[existingID.String()]
	if gotExisting.UpdatedRev != 7 || gotExisting.Deleted {
		t.Fatalf("expected existing updated, got=%#v", gotExisting)
	}
}

func TestGetItem_DecryptsEnvelope(t *testing.T) {
	t.Parallel()

	app, store := newTestApp(t)
	fc := app.Client.(*fakeClient)

	salt := make([]byte, 16)
	params := contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1}
	_ = store.SaveState(clientstate.State{
		ServerURL:     fc.url,
		UserID:        "u1",
		JWTToken:      "tkn",
		ExpiresAt:     "exp",
		KDFSalt:       salt,
		KDFParams:     params,
		LastSyncedRev: 0,
	})

	id := uuid.New()
	env := ItemEnvelope{V: 1, Type: "text", Meta: map[string]string{"k": "v"}, MetaText: "m", Data: []byte("hello")}
	plain, _ := json.Marshal(env)
	key, err := clientcrypto.DeriveKey("master", salt, params)
	if err != nil {
		t.Fatalf("DeriveKey() err=%v", err)
	}
	aad := []byte(id.String() + ":" + "text")
	ct, nonce, err := clientcrypto.Encrypt(key, plain, aad)
	if err != nil {
		t.Fatalf("Encrypt() err=%v", err)
	}

	_ = store.SaveVault(clientstate.Vault{Store: map[string]contract.ItemDTO{
		id.String(): {ID: id, Type: "text", Ciphertext: ct, Nonce: nonce, AAD: aad, Deleted: false},
	}})

	got, err := app.GetItem("master", id)
	if err != nil {
		t.Fatalf("GetItem() err=%v", err)
	}
	if got.Type != "text" || string(got.Data) != "hello" || got.MetaText != "m" || got.Meta["k"] != "v" {
		t.Fatalf("unexpected envelope: %#v", got)
	}
}
