package clientapp

import (
	"context"
	"time"

	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
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
	// SetItem(ctx context.Context, item contract.SetItemRequest) (contract.SetItemResponse, error)
	// DeleteItem(ctx context.Context) (contract.DeleteItemResponse, error)
	// GetItem(ctx context.Context) (contract.GetItemResponse, error)
	// GetAllItems(ctx context.Context) (contract.GetAllItemsResponse, error)
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
