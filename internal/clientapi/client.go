package clientapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"github.com/go-resty/resty/v2"
	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
	"go.uber.org/zap"
)

const (
	registerURL   = "/api/user/register"
	loginURL      = "/api/user/login"
	setItemUrl    = "/api/user/items/"
	deleteItemUrl = "/api/user/items/"
)

type HTTPClient struct {
	baseURL string
	client  *resty.Client
	token   string
	logger  *zap.SugaredLogger
}

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

func (c *HTTPClient) Register(ctx context.Context, login string, password string) (contract.RegisterResponse, error) {
	if len(password) < 8 {
		c.logger.Infow("password length < 8",
			"op", "client.register",
			"len", len(password),
		)
		return contract.RegisterResponse{}, fmt.Errorf("password should be at least 8 characters")
	}

	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetContext(ctx).
		SetBody(contract.RegisterRequest{
			Login:    login,
			Password: password,
		})

	resp, err := r.Post(c.baseURL + registerURL)
	if err != nil {
		c.logger.Infow("register request return error",
			"op", "client.register",
			"url", c.baseURL+registerURL,
			"err", err,
		)
		return contract.RegisterResponse{}, fmt.Errorf("failed to register user %s: %v", login, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return contract.RegisterResponse{}, fmt.Errorf("something went wrong. bad status: %s", resp.Status())
	}

	var result contract.RegisterResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		c.logger.Errorw("umurshal resp body error",
			"op", "client.register",
		)
		return contract.RegisterResponse{}, err
	}

	c.token = result.JWTToken

	return result, nil
}

func (c *HTTPClient) Login(ctx context.Context, login string, password string) (contract.LoginResponse, error) {
	r := c.client.R()

	r.SetHeader("Content-Type", "application/json").
		SetContext(ctx).
		SetBody(contract.LoginRequest{
			Login:    login,
			Password: password,
		})

	resp, err := r.Post(c.baseURL + loginURL)
	if err != nil {
		c.logger.Infow("login request return error",
			"op", "client.login",
			"url", c.baseURL+loginURL,
			"err", err,
		)
		return contract.LoginResponse{}, fmt.Errorf("failed to login user %s: %v", login, err)
	}

	if resp.StatusCode() != http.StatusOK {
		return contract.LoginResponse{}, fmt.Errorf("something went wrong. bad status: %s", resp.Status())
	}

	var result contract.LoginResponse
	if err := json.Unmarshal(resp.Body(), &result); err != nil {
		c.logger.Errorw("umurshal resp body error",
			"op", "client.login",
		)
		return contract.LoginResponse{}, err
	}

	c.token = result.JWTToken

	return result, nil
}

func (c *HTTPClient) SetItem(ctx context.Context, itemID string, item contract.SetItemRequest) (contract.SetItemResponse, error) {
	if ok := c.CheckToken(); !ok {
		return contract.SetItemResponse{}, fmt.Errorf("missing token, please login")
	}

	url := c.baseURL + setItemUrl + itemID
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

func (c *HTTPClient) DeleteItem(ctx context.Context, itemID string) (contract.DeleteItemResponse, error) {
	if ok := c.CheckToken(); !ok {
		return contract.DeleteItemResponse{}, fmt.Errorf("missing token, please login")
	}

	url := c.baseURL + deleteItemUrl + itemID
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

func (c *HTTPClient) GetURL() string {
	return c.baseURL
}

func (c *HTTPClient) SetToken(token string) {
	c.token = token
}

func (c *HTTPClient) CheckToken() bool {
	return c.token != ""
}
