// Package clientapi provides an HTTP client for the server API.
package clientapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

const (
	registerURL   = "/api/user/register"
	loginURL      = "/api/user/login"
	setItemURL    = "/api/user/items/"
	deleteItemURL = "/api/user/items/"
	syncURL       = "/api/user/sync/changes"
)

// HTTPClient implements client-side access to the server HTTP API.
type HTTPClient struct {
	baseURL string
	client  *resty.Client
	token   string
	logger  *zap.SugaredLogger
}

// NewHTTPClient creates an HTTPClient configured for the given baseURL.
func NewHTTPClient(baseURL string, logger *zap.SugaredLogger) *HTTPClient {
	logger = logger.With("component", "http client")
	client := resty.New()

	return &HTTPClient{
		baseURL: baseURL,
		client:  client,
		logger:  logger,
	}
}

type apiError struct {
	Error string `json:"error"`
}

func parseAPIError(body []byte) string {
	var e apiError
	if err := json.Unmarshal(body, &e); err == nil && strings.TrimSpace(e.Error) != "" {
		return e.Error
	}
	return strings.TrimSpace(string(body))
}

// Register registers a new user using the server API.
// The context controls cancellation and deadlines.
// Register returns an error if the request fails, the response status is not OK, or the response body can't be decoded.
func (c *HTTPClient) Register(ctx context.Context, login string, password string) (contract.RegisterResponse, error) {
	if len(password) < 8 {
		c.logger.Infow("password length < 8",
			"op", "client.register",
			"len", len(password),
		)
		return contract.RegisterResponse{}, fmt.Errorf("password should be at least 8 characters")
	}

	url := c.baseURL + registerURL
	c.logger.Infow("register request started",
		"op", "client.register",
		"url", url,
		"login", login,
	)

	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetContext(ctx).
		SetBody(contract.RegisterRequest{
			Login:    login,
			Password: password,
		})

	resp, err := r.Post(url)
	if err != nil {
		c.logger.Errorw("register request failed",
			"op", "client.register",
			"url", url,
			"login", login,
			"err", err,
		)
		return contract.RegisterResponse{}, fmt.Errorf("failed to register user %s: %w", login, err)
	}

	if resp.StatusCode() != http.StatusOK {
		msg := parseAPIError(resp.Body())
		c.logger.Errorw("register bad status",
			"op", "client.register",
			"url", url,
			"login", login,
			"status", resp.Status(),
			"error_msg", msg,
		)
		if msg == "" {
			msg = resp.Status()
		}
		return contract.RegisterResponse{}, fmt.Errorf("register failed: %s", msg)
	}

	var result contract.RegisterResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		c.logger.Errorw("unmarshal resp body error",
			"op", "client.register",
			"url", url,
			"login", login,
			"err", err,
		)
		return contract.RegisterResponse{}, fmt.Errorf("unmarshal register response: %w", err)
	}

	c.token = result.JWTToken

	c.logger.Infow("register request succeeded",
		"op", "client.register",
		"url", url,
		"login", login,
		"user_id", result.UserID,
	)

	return result, nil
}

// Login authenticates a user using the server API.
// The context controls cancellation and deadlines.
// Login returns an error if the request fails, the response status is not OK, or the response body can't be decoded.
func (c *HTTPClient) Login(ctx context.Context, login string, password string) (contract.LoginResponse, error) {
	url := c.baseURL + loginURL
	c.logger.Infow("login request started",
		"op", "client.login",
		"url", url,
		"login", login,
	)

	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetContext(ctx).
		SetBody(contract.LoginRequest{
			Login:    login,
			Password: password,
		})

	resp, err := r.Post(url)
	if err != nil {
		c.logger.Errorw("login request failed",
			"op", "client.login",
			"url", url,
			"login", login,
			"err", err,
		)
		return contract.LoginResponse{}, fmt.Errorf("failed to login user %s: %w", login, err)
	}

	if resp.StatusCode() != http.StatusOK {
		msg := parseAPIError(resp.Body())
		c.logger.Errorw("login bad status",
			"op", "client.login",
			"url", url,
			"login", login,
			"status", resp.Status(),
			"error_msg", msg,
		)
		if msg == "" {
			msg = resp.Status()
		}
		return contract.LoginResponse{}, fmt.Errorf("login failed: %s", msg)
	}

	var result contract.LoginResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		c.logger.Errorw("unmarshal resp body error",
			"op", "client.login",
			"url", url,
			"login", login,
			"err", err,
		)
		return contract.LoginResponse{}, fmt.Errorf("unmarshal login response: %w", err)
	}

	c.token = result.JWTToken

	c.logger.Infow("login request succeeded",
		"op", "client.login",
		"url", url,
		"login", login,
		"user_id", result.UserID,
	)

	return result, nil
}

// SetItem creates or updates an item with the given ID using the server API.
// The context controls cancellation and deadlines.
// SetItem returns an error if the client has no token, the request fails, the response status is not OK, or the response body can't be decoded.
func (c *HTTPClient) SetItem(ctx context.Context, itemID string, item contract.SetItemRequest) (contract.SetItemResponse, error) {
	if ok := c.CheckToken(); !ok {
		return contract.SetItemResponse{}, fmt.Errorf("missing token, please login")
	}

	url := c.baseURL + setItemURL + itemID
	c.logger.Infow("set item request started",
		"op", "client.set_item",
		"url", url,
		"item_id", itemID,
		"type", item.Type,
	)

	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.token)).
		SetContext(ctx).
		SetBody(item)

	resp, err := r.Put(url)
	if err != nil {
		c.logger.Errorw("set item request failed",
			"op", "client.set_item",
			"url", url,
			"item_id", itemID,
			"err", err,
		)
		return contract.SetItemResponse{}, fmt.Errorf("set item request failed: %w", err)

	}

	if resp.StatusCode() != http.StatusOK {
		msg := parseAPIError(resp.Body())
		c.logger.Errorw("set item bad status",
			"op", "client.set_item",
			"url", url,
			"item_id", itemID,
			"status", resp.Status(),
			"error_msg", msg,
		)
		if msg == "" {
			msg = resp.Status()
		}
		return contract.SetItemResponse{}, fmt.Errorf("set item failed: %s", msg)

	}

	var res contract.SetItemResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		c.logger.Errorw("unmarshal resp body error",
			"op", "client.set_item",
			"url", url,
			"item_id", itemID,
			"err", err,
		)
		return contract.SetItemResponse{}, fmt.Errorf("unmarshal set item response: %w", err)

	}

	c.logger.Infow("set item request succeeded",
		"op", "client.set_item",
		"url", url,
		"item_id", itemID,
		"updated_rev", res.Item.UpdatedRev,
		"deleted", res.Item.Deleted,
	)

	return res, nil
}

// DeleteItem deletes an item with the given ID using the server API.
// The context controls cancellation and deadlines.
// DeleteItem returns an error if the client has no token, the request fails, the response status is not OK, or the response body can't be decoded.
func (c *HTTPClient) DeleteItem(ctx context.Context, itemID string) (contract.DeleteItemResponse, error) {
	if ok := c.CheckToken(); !ok {
		return contract.DeleteItemResponse{}, fmt.Errorf("missing token, please login")
	}

	url := c.baseURL + deleteItemURL + itemID
	c.logger.Infow("delete item request started",
		"op", "client.delete_item",
		"url", url,
		"item_id", itemID,
	)

	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.token)).
		SetContext(ctx)

	resp, err := r.Delete(url)
	if err != nil {
		c.logger.Errorw("delete item request failed",
			"op", "client.delete_item",
			"url", url,
			"item_id", itemID,
			"err", err,
		)
		return contract.DeleteItemResponse{}, fmt.Errorf("delete item request failed: %w", err)

	}

	if resp.StatusCode() != http.StatusOK {
		msg := parseAPIError(resp.Body())
		c.logger.Errorw("delete item bad status",
			"op", "client.delete_item",
			"url", url,
			"item_id", itemID,
			"status", resp.Status(),
			"error_msg", msg,
		)
		if msg == "" {
			msg = resp.Status()
		}
		return contract.DeleteItemResponse{}, fmt.Errorf("delete item failed: %s", msg)

	}

	var res contract.DeleteItemResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		c.logger.Errorw("unmarshal resp body error",
			"op", "client.delete_item",
			"url", url,
			"item_id", itemID,
			"err", err,
		)
		return contract.DeleteItemResponse{}, fmt.Errorf("unmarshal delete item response: %w", err)

	}

	c.logger.Infow("delete item request succeeded",
		"op", "client.delete_item",
		"url", url,
		"item_id", itemID,
		"updated_rev", res.Item.UpdatedRev,
		"deleted", res.Item.Deleted,
	)

	return res, nil
}

// SyncChanges fetches items changed since lastSyncedRev using the server API.
// The context controls cancellation and deadlines.
// SyncChanges returns an error if the client has no token, lastSyncedRev is negative, the request fails, the response status is not OK, or the response body can't be decoded.
func (c *HTTPClient) SyncChanges(ctx context.Context, lastSyncedRev int64) (contract.SyncItemsResponse, error) {
	if ok := c.CheckToken(); !ok {
		return contract.SyncItemsResponse{}, fmt.Errorf("missing token, please login")
	}

	if lastSyncedRev < 0 {
		return contract.SyncItemsResponse{}, fmt.Errorf("lastSyncedRev can't be negative")
	}

	url := c.baseURL + syncURL
	c.logger.Infow("sync changes request started",
		"op", "client.sync_changes",
		"url", url,
		"since", lastSyncedRev,
	)

	r := c.client.R()

	r.SetHeader("Authorization", fmt.Sprintf("Bearer %s", c.token)).
		SetQueryParam("since", strconv.FormatInt(lastSyncedRev, 10)).
		SetContext(ctx)
		
	resp, err := r.Get(url)
	if err != nil {
		c.logger.Errorw("sync changes request failed",
			"op", "client.sync_changes",
			"url", url,
			"since", lastSyncedRev,
			"err", err,
		)
		return contract.SyncItemsResponse{}, fmt.Errorf("sync changes request failed: %w", err)
	}

	if resp.StatusCode() != http.StatusOK {
		msg := parseAPIError(resp.Body())
		c.logger.Errorw("sync changes bad status",
			"op", "client.sync_changes",
			"url", url,
			"since", lastSyncedRev,
			"status", resp.Status(),
			"error_msg", msg,
		)
		if msg == "" {
			msg = resp.Status()
		}
		return contract.SyncItemsResponse{}, fmt.Errorf("sync changes failed: %s", msg)
	}

	var res contract.SyncItemsResponse
	if err := json.Unmarshal(resp.Body(), &res); err != nil {
		c.logger.Errorw("unmarshal resp body error",
			"op", "client.sync_changes",
			"url", url,
			"since", lastSyncedRev,
			"err", err,
		)
		return contract.SyncItemsResponse{}, fmt.Errorf("unmarshal sync changes response: %w", err)
	}

	c.logger.Infow("sync changes request succeeded",
		"op", "client.sync_changes",
		"url", url,
		"since", lastSyncedRev,
		"latest_rev", res.LatestRev,
		"items_count", len(res.Items),
	)

	return res, nil
}

// GetURL returns the configured API base URL.
func (c *HTTPClient) GetURL() string {
	return c.baseURL
}

// SetToken sets the bearer token used for authenticated requests.
func (c *HTTPClient) SetToken(token string) {
	c.token = token
}

// CheckToken reports whether a bearer token is currently configured.
func (c *HTTPClient) CheckToken() bool {
	return c.token != ""
}
