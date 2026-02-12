package clientapi

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/go-resty/resty/v2"
	"github.com/makimaki04/go-data-keeper.git/cmd/pkg/contract"
	"go.uber.org/zap"
)

const (
	registerURL = `/api/user/register`
	loginURL    = `/api/user/login`
)

type HTTPClient struct {
	baseURL string
	client  *resty.Client
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

	return result, nil
}

func (c *HTTPClient) GetURL() string {
	return c.baseURL
}
