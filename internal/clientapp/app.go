// Package clientapp provides the client application layer and high-level workflows.
package clientapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/clientcrypto"
	"github.com/makimaki04/go-data-keeper.git/internal/clientstate"
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

// App orchestrates client workflows using an API client and local storage.
type App struct {
	// Client is the API client used to talk to the server.
	Client IClient
	// Store is the local persistence layer for client state and vault data.
	Store  clientstate.Store
	ctx    context.Context
	logger *zap.SugaredLogger
}

// IClient describes the server API used by App.
type IClient interface {
	Register(ctx context.Context, login string, password string) (contract.RegisterResponse, error)
	Login(ctx context.Context, login string, password string) (contract.LoginResponse, error)
	SetItem(ctx context.Context, itemId string, item contract.SetItemRequest) (contract.SetItemResponse, error)
	DeleteItem(ctx context.Context, itemID string) (contract.DeleteItemResponse, error)
	SyncChanges(ctx context.Context, lastSyncedRev int64) (contract.SyncItemsResponse, error)
	GetURL() string
}

// NewApp creates an App using the provided API client and local store.
func NewApp(client IClient, store clientstate.Store, logger *zap.SugaredLogger) *App {
	logger = logger.With("component", "app")

	return &App{
		Client: client,
		Store:  store,
		ctx:    context.Background(),
		logger: logger,
	}
}

// Register registers a new user, persists the resulting auth state, and performs an initial sync.
// Register returns an error if the server request fails or local persistence fails.
func (a *App) Register(login string, password string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 7*time.Second)
	defer cancel()

	a.logger.Infow("register started",
		"op", "app.register",
		"login", login,
		"url", a.Client.GetURL(),
	)

	resp, err := a.Client.Register(ctx, login, password)
	if err != nil {
		a.logger.Errorw("register request failed",
			"op", "app.register",
			"login", login,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err
	}

	a.logger.Infow("register request succeeded",
		"op", "app.register",
		"login", login,
		"url", a.Client.GetURL(),
		"user_id", resp.UserID,
	)

	oldState, err := a.Store.LoadState()
	if err != nil {
		a.logger.Errorw("load state failed",
			"op", "app.register",
			"login", login,
			"err", err,
		)
		return err
	}

	if oldState.UserID != "" && oldState.UserID != resp.UserID {
		err := a.Store.WipeVault()
		if err != nil {
			a.logger.Errorw("wipe vault failed",
				"op", "app.register",
				"login", login,
				"err", err,
			)
			return err
		}
		a.logger.Infow("vault successfully wiped",
			"op", "app.register",
			"login", login,
			"user_id", resp.UserID,
		)
	}

	data := clientstate.State{
		ServerURL:     a.Client.GetURL(),
		UserID:        resp.UserID,
		JWTToken:      resp.JWTToken,
		ExpiresAt:     resp.ExpiresAt,
		KDFSalt:       resp.KDFSalt,
		KDFParams:     resp.KDFParams,
		LastSyncedRev: 0,
	}

	if err := a.Store.SaveState(data); err != nil {
		a.logger.Errorw("save state failed",
			"op", "app.register",
			"login", login,
			"err", err,
		)
		return err
	}

	a.logger.Infow("register completed",
		"op", "app.register",
		"login", login,
		"user_id", resp.UserID,
	)

	a.logger.Infow("sync started",
		"op", "app.sync_changes",
		"trigger", "app.register",
		"login", login,
		"user_id", resp.UserID,
	)

	if err := a.SyncChanges(); err != nil {
		a.logger.Errorw("sync changes failed",
			"op", "app.sync_changes",
			"trigger", "app.register",
			"login", login,
			"err", err,
		)
		return err
	}

	a.logger.Infow("sync completed",
		"op", "app.sync_changes",
		"trigger", "app.register",
		"login", login,
		"user_id", resp.UserID,
	)

	return nil
}

// Login authenticates a user, persists the resulting auth state, and performs a sync.
// Login returns an error if the server request fails or local persistence fails.
func (a *App) Login(login string, password string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 7*time.Second)
	defer cancel()

	a.logger.Infow("login started",
		"op", "app.login",
		"login", login,
		"url", a.Client.GetURL(),
	)

	resp, err := a.Client.Login(ctx, login, password)
	if err != nil {
		a.logger.Errorw("login request failed",
			"op", "app.login",
			"login", login,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err
	}

	state, err := a.Store.LoadState()
	if err != nil {
		a.logger.Errorw("load state failed",
			"op", "app.login",
			"login", login,
			"err", err,
		)
		return err
	}

	if state.UserID != "" && state.UserID != resp.UserID {
		if err := a.Store.WipeVault(); err != nil {
			a.logger.Errorw("wipe vault failed",
				"op", "app.login",
				"login", login,
				"err", err,
			)
			return err
		}

		a.logger.Infow("vault successfully wiped",
			"op", "app.login",
			"login", login,
			"user_id", resp.UserID,
		)

		state.LastSyncedRev = 0
	}

	state.UserID = resp.UserID
	state.ServerURL = a.Client.GetURL()
	state.JWTToken = resp.JWTToken
	state.ExpiresAt = resp.ExpiresAt
	state.KDFSalt = resp.KDFSalt
	state.KDFParams = resp.KDFParams

	if err := a.Store.SaveState(state); err != nil {
		a.logger.Errorw("save state failed",
			"op", "app.login",
			"login", login,
			"err", err,
		)
		return err
	}

	a.logger.Infow("login completed",
		"op", "app.login",
		"login", login,
		"url", a.Client.GetURL(),
	)

	a.logger.Infow("sync started",
		"op", "app.sync_changes",
		"trigger", "app.login",
		"login", login,
		"url", a.Client.GetURL(),
	)

	if err := a.SyncChanges(); err != nil {
		a.logger.Errorw("sync changes failed",
			"op", "app.sync_changes",
			"trigger", "app.login",
			"login", login,
			"err", err,
		)
		return err
	}

	a.logger.Infow("sync completed",
		"op", "app.sync_changes",
		"trigger", "app.login",
		"login", login,
		"url", a.Client.GetURL(),
	)

	return nil
}

// ItemEnvelope is a plaintext payload that is encrypted and stored as an item.
type ItemEnvelope struct {
	V        int               `json:"v"`
	Type     string            `json:"type"`
	Meta     map[string]string `json:"meta,omitempty"`
	MetaText string            `json:"meta_text,omitempty"`
	Data     []byte            `json:"data,omitempty"`
}

// SetOptions configures metadata stored in the item envelope.
type SetOptions struct {
	Meta     map[string]string
	MetaText string
}

// SetItem encrypts data and stores it on the server, then updates the local vault.
// SetItem returns an error if encryption, the server request, or local persistence fails.
func (a *App) SetItem(masterPassword string, data []byte, options SetOptions, id uuid.UUID, itemType string) error {
	ctx, cancel := context.WithTimeout(a.ctx, 12*time.Second)
	defer cancel()

	if masterPassword == "" {
		err := fmt.Errorf("masterPassword is required")
		a.logger.Errorw("set item failed: empty master password",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err
	}

	if itemType == "" {
		err := fmt.Errorf("itemType is required")
		a.logger.Errorw("set item failed: empty item type",
			"op", "app.set_item",
			"item_id", id,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err
	}

	a.logger.Infow("set item started",
		"op", "app.set_item",
		"item_id", id,
		"type", itemType,
		"plaintext_len", len(data),
		"url", a.Client.GetURL(),
	)

	user, err := a.Store.LoadState()
	if err != nil {
		a.logger.Errorw("load state failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"err", err,
		)
		return err

	}

	key, err := clientcrypto.DeriveKey(masterPassword, user.KDFSalt, user.KDFParams)
	if err != nil {
		a.logger.Errorw("derive key failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"err", err,
		)
		return err

	}

	aad := []byte(id.String() + ":" + itemType)

	env := ItemEnvelope{
		V:        1,
		Type:     itemType,
		Meta:     options.Meta,
		MetaText: options.MetaText,
		Data:     data,
	}

	plaintext, err := json.Marshal(env)
	if err != nil {
		a.logger.Errorw("envelope marshal failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"err", err,
		)
		return err
	}

	ciphertext, nonce, err := clientcrypto.Encrypt(key, plaintext, aad)
	if err != nil {
		a.logger.Errorw("encrypt failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"err", err,
		)
		return err
	}

	resp, err := a.Client.SetItem(ctx, id.String(), contract.SetItemRequest{
		Type:       itemType,
		Ciphertext: ciphertext,
		Nonce:      nonce,
		AAD:        aad,
	})

	if err != nil {
		a.logger.Errorw("set item request failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err

	}

	vault, err := a.Store.LoadVault()
	if err != nil {
		a.logger.Errorw("load vault failed",
			"op", "app.set_item",
			"item_id", id,
			"type", itemType,
			"err", err,
		)
		return err

	}

	if vault.Store == nil {
		vault.Store = make(map[string]contract.ItemDTO)
	}

	item := vault.Store[resp.Item.ID.String()]
	item.ID = resp.Item.ID
	item.Type = resp.Item.Type
	item.Ciphertext = resp.Item.Ciphertext
	item.Nonce = resp.Item.Nonce
	item.AAD = resp.Item.AAD
	item.UpdatedRev = resp.Item.UpdatedRev
	item.Deleted = resp.Item.Deleted
	item.CreatedAt = resp.Item.CreatedAt
	item.UpdatedAt = resp.Item.UpdatedAt

	vault.Store[resp.Item.ID.String()] = item

	if err := a.Store.SaveVault(vault); err != nil {
		a.logger.Errorw("save vault failed",
			"op", "app.set_item",
			"item_id", resp.Item.ID,
			"type", itemType,
			"updated_rev", resp.Item.UpdatedRev,
			"err", err,
		)
		return err

	}

	a.logger.Infow("set item completed",
		"op", "app.set_item",
		"item_id", resp.Item.ID,
		"type", resp.Item.Type,
		"deleted", resp.Item.Deleted,
		"updated_rev", resp.Item.UpdatedRev,
		"url", a.Client.GetURL(),
	)

	return nil
}

// DeleteItem deletes an item on the server and records the tombstone in the local vault.
// DeleteItem returns an error if the server request or local persistence fails.
func (a *App) DeleteItem(id uuid.UUID) error {
	ctx, cancel := context.WithTimeout(a.ctx, 10*time.Second)
	defer cancel()

	a.logger.Infow("delete item started",
		"op", "app.delete_item",
		"item_id", id,
		"url", a.Client.GetURL(),
	)

	resp, err := a.Client.DeleteItem(ctx, id.String())
	if err != nil {
		a.logger.Errorw("delete item request failed",
			"op", "app.delete_item",
			"item_id", id,
			"url", a.Client.GetURL(),
			"err", err,
		)
		return err

	}

	vault, err := a.Store.LoadVault()
	if err != nil {
		a.logger.Errorw("load vault failed",
			"op", "app.delete_item",
			"item_id", id,
			"err", err,
		)
		return err

	}

	if vault.Store == nil {
		vault.Store = make(map[string]contract.ItemDTO)
	}

	item := vault.Store[resp.Item.ID.String()]
	item.ID = resp.Item.ID
	item.Type = resp.Item.Type
	item.Deleted = resp.Item.Deleted
	item.UpdatedRev = resp.Item.UpdatedRev
	item.Ciphertext = nil
	item.AAD = nil
	item.Nonce = nil
	item.CreatedAt = resp.Item.CreatedAt
	item.UpdatedAt = resp.Item.UpdatedAt

	vault.Store[resp.Item.ID.String()] = item

	if err := a.Store.SaveVault(vault); err != nil {
		a.logger.Errorw("save vault failed",
			"op", "app.delete_item",
			"item_id", resp.Item.ID,
			"updated_rev", resp.Item.UpdatedRev,
			"err", err,
		)
		return err

	}

	a.logger.Infow("delete item completed",
		"op", "app.delete_item",
		"item_id", resp.Item.ID,
		"deleted", resp.Item.Deleted,
		"updated_rev", resp.Item.UpdatedRev,
		"url", a.Client.GetURL(),
	)

	return nil
}

// GetList returns items from the local vault filtered by the provided flags.
// GetList returns an error if the vault can't be loaded.
func (a *App) GetList(all bool, itemType string, deleted bool) ([]contract.ItemDTO, error) {
	var items []contract.ItemDTO

	a.logger.Infow("get list started",
		"op", "app.get_list",
		"all", all,
		"type", itemType,
		"deleted", deleted,
	)

	vault, err := a.Store.LoadVault()
	if err != nil {
		a.logger.Errorw("load vault failed",
			"op", "app.get_list",
			"all", all,
			"type", itemType,
			"deleted", deleted,
			"err", err,
		)
		return []contract.ItemDTO{}, err
	}

	if all {
		if itemType != "" {
			for _, item := range vault.Store {
				if item.Type == itemType {
					items = append(items, item)
				}
			}
		} else {
			for _, item := range vault.Store {
				items = append(items, item)
			}
		}

		a.logger.Infow("get list completed",
			"op", "app.get_list",
			"all", all,
			"type", itemType,
			"deleted", deleted,
			"items_count", len(items),
		)
		return items, nil
	}

	if deleted {
		if itemType != "" {
			for _, item := range vault.Store {
				if item.Deleted == true && item.Type == itemType {
					items = append(items, item)
				}
			}
		} else {
			for _, item := range vault.Store {
				if item.Deleted == true {
					items = append(items, item)
				}
			}
		}

		a.logger.Infow("get list completed",
			"op", "app.get_list",
			"all", all,
			"type", itemType,
			"deleted", deleted,
			"items_count", len(items),
		)
		return items, nil
	}

	if itemType != "" {
		for _, item := range vault.Store {
			if item.Type == itemType && item.Deleted != true {
				items = append(items, item)
			}
		}

		a.logger.Infow("get list completed",
			"op", "app.get_list",
			"all", all,
			"type", itemType,
			"deleted", deleted,
			"items_count", len(items),
		)
		return items, nil
	}

	for _, item := range vault.Store {
		if item.Deleted != true {
			items = append(items, item)
		}
	}

	a.logger.Infow("get list completed",
		"op", "app.get_list",
		"all", all,
		"type", itemType,
		"deleted", deleted,
		"items_count", len(items),
	)
	return items, nil
}

// GetItem decrypts an item from the local vault and returns its envelope.
// GetItem returns an error if the item is missing or deleted, decryption fails, or local state can't be loaded.
func (a *App) GetItem(masterPassword string, itemID uuid.UUID) (ItemEnvelope, error) {
	if masterPassword == "" {
		err := fmt.Errorf("master password is required")
		a.logger.Errorw("get item failed: empty master password",
			"op", "app.get_item",
			"item_id", itemID,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	id := itemID.String()

	a.logger.Infow("get item started",
		"op", "app.get_item",
		"item_id", id,
	)

	vault, err := a.Store.LoadVault()
	if err != nil {
		a.logger.Errorw("load vault failed",
			"op", "app.get_item",
			"item_id", id,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	item, ok := vault.Store[id]
	if !ok {
		err := fmt.Errorf("item not found: %s", id)
		a.logger.Errorw("get item failed: not found",
			"op", "app.get_item",
			"item_id", id,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	if item.Deleted == true {
		err := fmt.Errorf("item deleted")
		a.logger.Errorw("get item failed: item deleted",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	if len(item.Ciphertext) == 0 || len(item.Nonce) == 0 {
		err := fmt.Errorf("no encrypted payload")
		a.logger.Errorw("get item failed: missing encrypted payload",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	userState, err := a.Store.LoadState()
	if err != nil {
		a.logger.Errorw("load state failed",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	key, err := clientcrypto.DeriveKey(masterPassword, userState.KDFSalt, userState.KDFParams)
	if err != nil {
		a.logger.Errorw("derive key failed",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	aad := []byte(item.ID.String() + ":" + item.Type)

	plaintext, err := clientcrypto.Decrypt(key, item.Ciphertext, item.Nonce, aad)
	if err != nil {
		a.logger.Errorw("decrypt failed",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	var env ItemEnvelope
	if err := json.Unmarshal(plaintext, &env); err != nil {
		a.logger.Errorw("envelope unmarshal failed",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	if env.Type != item.Type {
		err := fmt.Errorf("data integrity error: envelope type mismatch")
		a.logger.Errorw("get item failed: envelope type mismatch",
			"op", "app.get_item",
			"item_id", id,
			"type", item.Type,
			"envelope_type", env.Type,
			"err", err,
		)
		return ItemEnvelope{}, err
	}

	a.logger.Infow("get item completed",
		"op", "app.get_item",
		"item_id", id,
		"type", item.Type,
		"has_meta", len(env.Meta) > 0,
		"has_meta_text", env.MetaText != "",
		"data_len", len(env.Data),
	)
	return env, nil
}

// SyncChanges fetches changes from the server since the last sync and applies them to the local vault.
// SyncChanges returns an error if the server request fails or local persistence fails.
func (a *App) SyncChanges() error {
	ctx, cancel := context.WithTimeout(a.ctx, 15*time.Second)
	defer cancel()

	a.logger.Infow("sync changes started",
		"op", "app.sync_changes",
		"url", a.Client.GetURL(),
	)

	state, err := a.Store.LoadState()
	if err != nil {
		a.logger.Errorw("load state failed",
			"op", "app.sync_changes",
			"err", err,
		)
		return err
	}

	lastRev := state.LastSyncedRev
	a.logger.Infow("sync changes fetching",
		"op", "app.sync_changes",
		"url", a.Client.GetURL(),
		"since", lastRev,
	)
	changes, err := a.Client.SyncChanges(ctx, lastRev)
	if err != nil {
		a.logger.Errorw("sync changes request failed",
			"op", "app.sync_changes",
			"url", a.Client.GetURL(),
			"since", lastRev,
			"err", err,
		)
		return err
	}

	vault, err := a.Store.LoadVault()
	if err != nil {
		a.logger.Errorw("load vault failed",
			"op", "app.sync_changes",
			"err", err,
		)
		return err
	}

	itemsStore := vault.Store

	for _, item := range changes.Items {
		if item.Deleted == true {
			item.Ciphertext = nil
			item.Nonce = nil
			item.AAD = nil
		}

		itemsStore[item.ID.String()] = item
	}

	vault.Store = itemsStore
	if err := a.Store.SaveVault(vault); err != nil {
		a.logger.Errorw("save vault failed",
			"op", "app.sync_changes",
			"latest_rev", changes.LatestRev,
			"items_count", len(changes.Items),
			"err", err,
		)
		return err
	}

	state.LastSyncedRev = changes.LatestRev
	if err := a.Store.SaveState(state); err != nil {
		a.logger.Errorw("save state failed",
			"op", "app.sync_changes",
			"latest_rev", changes.LatestRev,
			"err", err,
		)
		return err
	}

	a.logger.Infow("sync changes completed",
		"op", "app.sync_changes",
		"url", a.Client.GetURL(),
		"since", lastRev,
		"latest_rev", changes.LatestRev,
		"items_count", len(changes.Items),
	)

	return nil
}
