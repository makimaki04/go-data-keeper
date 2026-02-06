package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/service"
	"go.uber.org/zap"
)

func WithAuth(secret string, logger *zap.SugaredLogger) func(next http.Handler) http.Handler {
	logger = logger.With("component", "auth_middleware")

	return func(next http.Handler) http.Handler {
		fn := func(w http.ResponseWriter, r *http.Request) {
			authHeader := r.Header.Get("Authorization")
			if !strings.HasPrefix(authHeader, "Bearer ") {
				logger.Warnf("Got invalid token")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			tokenStr := strings.TrimPrefix(authHeader, "Bearer ")
			if tokenStr == "" {
				logger.Warnf("Got empty token")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			claims := &service.Claims{}
			token, err := jwt.ParseWithClaims(tokenStr, claims, func(t *jwt.Token) (interface{}, error) {
				if t.Method.Alg() != jwt.SigningMethodHS256.Alg() {
					logger.Infow("JWT invalid signature method", "method", t.Method.Alg())
					return nil, jwt.ErrTokenSignatureInvalid
				}

				return []byte(secret), nil
			})
			if err != nil {
				var verr *jwt.ValidationError
				if errors.As(err, &verr) {
					switch {
					case errors.Is(err, jwt.ErrTokenExpired):
						logger.Infow("JWT Expired", "err", err)
						http.Error(w, "token expired", http.StatusUnauthorized)
						return
					case errors.Is(err, jwt.ErrTokenNotValidYet):
						logger.Infow("JWT not valid yet", "err", err)
						http.Error(w, "token not valid yet", http.StatusUnauthorized)
						return
					case errors.Is(err, jwt.ErrTokenMalformed):
						logger.Infow("JWT malformed", "err", err)
						http.Error(w, "invalid token", http.StatusUnauthorized)
						return
					case errors.Is(err, jwt.ErrTokenSignatureInvalid):
						logger.Infow("JWT bad signature", "err", err)
						http.Error(w, "invalid token", http.StatusUnauthorized)
						return
					default:
						logger.Infow("JWT validation error", "err", err)
						http.Error(w, "invalid token", http.StatusUnauthorized)
						return
					}
				}

				logger.Infow("JWT parse error", "err", err)
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if !token.Valid {
				logger.Info("JWT not valid")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			if claims.UserID == uuid.Nil {
				logger.Info("JWT empty user_id")
				http.Error(w, "invalid token", http.StatusUnauthorized)
				return
			}

			ctx := context.WithValue(r.Context(), userIDKey, claims.UserID)
			r = r.WithContext(ctx)
			next.ServeHTTP(w, r)
		}

		return http.HandlerFunc(fn)
	}
}

type contextKey string

const userIDKey contextKey = "userID"

func GetUserID(r *http.Request) (uuid.UUID, bool) {
	id, ok := r.Context().Value(userIDKey).(uuid.UUID)
	return id, ok
}
