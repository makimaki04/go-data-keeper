package handler

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/makimaki04/go-data-keeper.git/internal/dto"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"github.com/makimaki04/go-data-keeper.git/internal/service"
	"go.uber.org/zap"
)

type Handler struct {
	service *service.Service
	logger  *zap.SugaredLogger
}

func NewHandler(service *service.Service, logger *zap.SugaredLogger) *Handler {
	logger = logger.With("component", "handler")

	return &Handler{
		service: service,
		logger:  logger,
	}
}

const (
	MB = 1 << 20

	MaxBodyAuth = 1 * MB
	MaxBodyJSON = 3 * MB
)

func parseJSONBody[T any](w http.ResponseWriter, r *http.Request, MaxBytesRead int64, logger *zap.SugaredLogger) (T, error) {
	var req T

	r.Body = http.MaxBytesReader(w, r.Body, MaxBytesRead)
	defer r.Body.Close()

	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()

	if err := dec.Decode(&req); err != nil {
		if errors.Is(err, io.EOF) {
			logger.Errorw("empty request body",
				"op", "parse_json_body",
				"error", err,
			)

			return req, fmt.Errorf("emty request body: %w", err)
		}

		logger.Errorw("decode json body err",
			"op", "parse_json_body",
			"error", err,
		)

		return req, fmt.Errorf("decode json body error: %w", err)
	}

	return req, nil
}

func (h *Handler) RegisterUser(w http.ResponseWriter, r *http.Request) {
	req, err := parseJSONBody[dto.RegisterRequest](w, r, MaxBodyAuth, h.logger)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		return
	}

	login, password, err := checkAuthData(req.Login, req.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		return
	}

	ctx := r.Context()
	id, token, err := h.service.RegisterUser(ctx, login, password)
	if err != nil {
		if errors.Is(err, repository.ErrUserExists) {
			respondWithError(w, http.StatusConflict, repository.ErrUserExists.Error(), h.logger)
			return
		}

		respondWithError(w, http.StatusInternalServerError, "internal server error", h.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, dto.RegisterResponse{
		ID:        id.String(),
		JWTToken:  token.AccessToken,
		ExpiresAt: token.ExpiresAt.Format(time.UnixDate),
	}, h.logger)
}

func (h *Handler) LoginUser(w http.ResponseWriter, r *http.Request) {
	req, err := parseJSONBody[dto.LoginRequest](w, r, MaxBodyAuth, h.logger)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		return
	}

	login, password, err := checkAuthData(req.Login, req.Password)
	if err != nil {
		respondWithError(w, http.StatusBadRequest, err.Error(), h.logger)
		return
	}

	ctx := r.Context()
	token, err := h.service.LoginUser(ctx, login, password)
	if err != nil {
		if errors.Is(err, repository.ErrUserNotFound) || errors.Is(err, service.ErrInvalidCredentials) {
			respondWithError(w, http.StatusUnauthorized, "wrong login or password", h.logger)
			return
		}
		respondWithError(w, http.StatusInternalServerError, "internal server error", h.logger)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Authorization", fmt.Sprintf("Bearer %s", token.AccessToken))
	w.WriteHeader(http.StatusOK)
	encodeResponse(w, dto.LoginResponse{
		JWTToken:  token.AccessToken,
		ExpiresAt: token.ExpiresAt.Format(time.UnixDate),
	}, h.logger)
}

func checkAuthData(login string, password string) (string, string, error) {
	login = strings.TrimSpace(login)
	password = strings.TrimSpace(password)
	if login == "" || password == "" {
		return "", "", fmt.Errorf("login and password are required")
	}

	if len(password) < 8 {
		return "", "", fmt.Errorf("password should be at leaast 8 characters")
	}

	return login, password, nil
}

func respondWithError(w http.ResponseWriter, code int, message string, logger *zap.SugaredLogger) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)

	resp := map[string]string{
		"error": message,
	}

	encodeResponse(w, resp, logger)
}

func encodeResponse[T any](w http.ResponseWriter, resp T, logger *zap.SugaredLogger) {
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		logger.Errorw("encode response failed",
			"op", "encode_response",
			"err", err,
		)
	}
}
