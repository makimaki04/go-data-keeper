package handler

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/middleware"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"github.com/makimaki04/go-data-keeper.git/internal/service"
	"github.com/makimaki04/go-data-keeper.git/pkg/contract"
	"go.uber.org/zap"
)

type fakeAuth struct {
	registerFn func(ctx context.Context, login, password string) (service.AuthData, error)
	loginFn    func(ctx context.Context, login, password string) (service.AuthData, error)
}

func (f fakeAuth) RegisterUser(ctx context.Context, login string, password string) (service.AuthData, error) {
	return f.registerFn(ctx, login, password)
}
func (f fakeAuth) LoginUser(ctx context.Context, login string, password string) (service.AuthData, error) {
	return f.loginFn(ctx, login, password)
}
func (f fakeAuth) GenerateToken(id uuid.UUID) (service.JWTToken, error) {
	return service.JWTToken{}, nil
}

type fakeItems struct {
	getChangesFn func(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error)
}

func (f fakeItems) SetItem(ctx context.Context, item models.Item) (models.Item, error) {
	panic("not used")
}
func (f fakeItems) DeleteItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error) {
	panic("not used")
}
func (f fakeItems) GetItem(ctx context.Context, itemID uuid.UUID, userID uuid.UUID) (models.Item, error) {
	panic("not used")
}
func (f fakeItems) GetAllItems(ctx context.Context, userID uuid.UUID) ([]models.Item, error) {
	panic("not used")
}
func (f fakeItems) GetChangesSince(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error) {
	return f.getChangesFn(ctx, userID, since)
}

func TestRegisterUser_OK(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	svc := &service.Service{
		Authorization: fakeAuth{
			registerFn: func(ctx context.Context, login, password string) (service.AuthData, error) {
				return service.AuthData{
					ID: wantID,
					JWT: service.JWTToken{
						AccessToken: "jwt",
						ExpiresAt:   time.Now().Add(time.Minute),
					},
					KDFSalt:   bytes.Repeat([]byte{1}, 16),
					KDFParams: contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
				}, nil
			},
			loginFn: func(ctx context.Context, login, password string) (service.AuthData, error) {
				t.Fatalf("unexpected login")
				return service.AuthData{}, nil
			},
		},
		Items: fakeItems{getChangesFn: func(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error) {
			return nil, 0, nil
		}},
	}

	h := NewHandler(svc, zap.NewNop().Sugar())
	body := []byte(`{"login":"alice","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/register", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.RegisterUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	if gotAuth := rr.Header().Get("Authorization"); gotAuth == "" {
		t.Fatalf("expected Authorization header set")
	}
	var resp contract.RegisterResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if resp.UserID != wantID.String() || resp.JWTToken == "" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
}

func TestLoginUser_OK(t *testing.T) {
	t.Parallel()

	wantID := uuid.New()
	svc := &service.Service{
		Authorization: fakeAuth{
			registerFn: func(ctx context.Context, login, password string) (service.AuthData, error) {
				t.Fatalf("unexpected register")
				return service.AuthData{}, nil
			},
			loginFn: func(ctx context.Context, login, password string) (service.AuthData, error) {
				return service.AuthData{
					ID: wantID,
					JWT: service.JWTToken{
						AccessToken: "jwt",
						ExpiresAt:   time.Now().Add(time.Minute),
					},
					KDFSalt:   bytes.Repeat([]byte{1}, 16),
					KDFParams: contract.Params{Algorithm: "argon2id", KeyLen: 32, SaltLen: 16, Time: 1, Memory: 8, Parallelism: 1},
				}, nil
			},
		},
		Items: fakeItems{getChangesFn: func(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error) {
			return nil, 0, nil
		}},
	}

	h := NewHandler(svc, zap.NewNop().Sugar())
	body := []byte(`{"login":"alice","password":"password123"}`)
	req := httptest.NewRequest(http.MethodPost, "/api/user/login", bytes.NewReader(body))
	rr := httptest.NewRecorder()

	h.LoginUser(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}
	var resp contract.LoginResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal resp: %v", err)
	}
	if resp.UserID != wantID.String() || resp.Login != "alice" || resp.JWTToken == "" {
		t.Fatalf("unexpected resp: %#v", resp)
	}
}

func TestGetChangesSince_Validation(t *testing.T) {
	t.Parallel()

	svc := &service.Service{
		Authorization: fakeAuth{
			registerFn: func(ctx context.Context, login, password string) (service.AuthData, error) { return service.AuthData{}, nil },
			loginFn:    func(ctx context.Context, login, password string) (service.AuthData, error) { return service.AuthData{}, nil },
		},
		Items: fakeItems{getChangesFn: func(ctx context.Context, userID uuid.UUID, since int64) ([]models.Item, int64, error) {
			return nil, 0, nil
		}},
	}

	h := NewHandler(svc, zap.NewNop().Sugar())

	tests := []struct {
		name   string
		url    string
		want   int
		userID uuid.UUID
	}{
		{name: "bad_since", url: "/api/user/sync/changes?since=abc", want: http.StatusBadRequest, userID: uuid.New()},
		{name: "negative_since", url: "/api/user/sync/changes?since=-1", want: http.StatusBadRequest, userID: uuid.New()},
	}

	for _, tt := range tests {
		tt := tt
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()
			req := httptest.NewRequest(http.MethodGet, tt.url, nil)
			rr := httptest.NewRecorder()
			h.GetChangesSince(rr, req)
			if rr.Code != tt.want {
				t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
			}
		})
	}
}

func TestGetChangesSince_OK_WithAuthMiddleware(t *testing.T) {
	t.Parallel()

	secret := "test-secret-which-is-long-enough-to-not-be-trivial"
	userID := uuid.New()

	svc := &service.Service{
		Authorization: fakeAuth{
			registerFn: func(ctx context.Context, login, password string) (service.AuthData, error) { return service.AuthData{}, nil },
			loginFn:    func(ctx context.Context, login, password string) (service.AuthData, error) { return service.AuthData{}, nil },
		},
		Items: fakeItems{getChangesFn: func(ctx context.Context, uid uuid.UUID, since int64) ([]models.Item, int64, error) {
			if uid != userID {
				t.Fatalf("unexpected userID: got=%s want=%s", uid, userID)
			}
			if since != 10 {
				t.Fatalf("unexpected since: got=%d want=%d", since, 10)
			}
			it := models.Item{
				ID:         uuid.New(),
				UserID:     userID,
				Type:       "text",
				Ciphertext: []byte("ct"),
				Nonce:      []byte("nonce"),
				AAD:        []byte("aad"),
				Deleted:    false,
				UpdatedRev: 11,
				CreatedAt:  time.Now(),
			}
			return []models.Item{it}, 11, nil
		}},
	}

	h := NewHandler(svc, zap.NewNop().Sugar())

	claims := service.Claims{UserID: userID, RegisteredClaims: jwt.RegisteredClaims{}}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/user/sync/changes?since=10", nil)
	req.Header.Set("Authorization", "Bearer "+tokenStr)
	rr := httptest.NewRecorder()

	protected := middleware.WithAuth(secret, zap.NewNop().Sugar())(http.HandlerFunc(h.GetChangesSince))
	protected.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("status=%d body=%s", rr.Code, rr.Body.String())
	}

	var out contract.SyncItemsResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &out); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if out.LatestRev != 11 || len(out.Items) != 1 {
		t.Fatalf("unexpected response: %#v", out)
	}
}

