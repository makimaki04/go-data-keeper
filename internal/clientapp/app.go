package clientapp

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
	"github.com/makimaki04/go-data-keeper.git/internal/clientcrypto"
	"github.com/makimaki04/go-data-keeper.git/internal/clientstate"
	"go.uber.org/zap"
)

type App struct {
	Client IClient
	Store  clientstate.Store
	ctx    context.Context
	logger *zap.SugaredLogger
}

type IClient interface {
	Register(ctx context.Context, login string, password string) (contract.RegisterResponse, error)
	Login(ctx context.Context, login string, password string) (contract.LoginResponse, error)
	SetItem(ctx context.Context, itemId string, item contract.SetItemRequest) (contract.SetItemResponse, error)
	DeleteItem(ctx context.Context, itemID string) (contract.DeleteItemResponse, error)
	// SyncChanges(ctx context.Context) (contract.SyncItemsResponse, error)
	GetURL() string
}

func NewApp(client IClient, store clientstate.Store, logger *zap.SugaredLogger) *App {
	logger = logger.With("component", "app")

	return &App{
		Client: client,
		Store:  store,
		ctx:    context.Background(),
		logger: logger,
	}
}

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
		"user_id", resp.ID,
	)

	data := clientstate.State{
		ServerURL:     a.Client.GetURL(),
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
		"user_id", resp.ID,
	)

	return nil
}

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

	return nil
}

type ItemEnvelope struct {
	V        int               `json:"v"`
	Type     string            `json:"type"`
	Meta     map[string]string `json:"meta,omitempty"`
	MetaText string            `json:"meta_text,omitempty"`
	Data     []byte            `json:"data,omitempty"`
}

type SetOptions struct {
	Meta     map[string]string
	MetaText string
}

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

func (a *App) GetList(all bool, itemType string, deleted bool) ([]contract.ItemDTO, error) {
	var items []contract.ItemDTO

	vault, err := a.Store.LoadVault()
	if err != nil {
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

		return items, nil
	}

	if itemType != "" {
		for _, item := range vault.Store {
			if item.Type == itemType && item.Deleted != true {
				items = append(items, item)
			}
		}

		return items, nil
	}

	for _, item := range vault.Store {
		if item.Deleted != true {
			items = append(items, item)
		}
	}

	return items, nil
}

func (a *App) GetItem(masterPassword string, itemID uuid.UUID) (ItemEnvelope, error) {
	if masterPassword == "" {
		return ItemEnvelope{}, fmt.Errorf("master password is required")
	}

	id := itemID.String()

	vault, err := a.Store.LoadVault()
	if err != nil {
		return ItemEnvelope{}, err
	}

	item, ok := vault.Store[id]
	if !ok {
		return ItemEnvelope{}, fmt.Errorf("item not found: %s", id)
	}

	if item.Deleted == true {
		return ItemEnvelope{}, fmt.Errorf("item deleted")
	}

	if len(item.Ciphertext) == 0 || len(item.Nonce) == 0 {
		return ItemEnvelope{}, fmt.Errorf("no encrypted payload")
	}

	userState, err := a.Store.LoadState()
	if err != nil {
		return ItemEnvelope{}, err
	}

	key, err := clientcrypto.DeriveKey(masterPassword, userState.KDFSalt, userState.KDFParams)
	if err != nil {
		return ItemEnvelope{}, err
	}

	aad := []byte(item.ID.String() + ":" + item.Type)

	plaintext, err := clientcrypto.Decrypt(key, item.Ciphertext, item.Nonce, aad)
	if err != nil {
		return ItemEnvelope{}, err
	}

	var env ItemEnvelope
	if err := json.Unmarshal(plaintext, &env); err != nil {
		return ItemEnvelope{}, err
	}

	if env.Type != item.Type {
		return ItemEnvelope{}, fmt.Errorf("data integrity error: envelope type mismatch")
	}

	return env, nil
}
