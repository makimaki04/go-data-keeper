package service

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v4"
	"github.com/google/uuid"
	"github.com/makimaki04/go-data-keeper.git/internal/models"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"go.uber.org/zap"
	"golang.org/x/crypto/bcrypt"
)

type AuthService struct {
	repo      repository.Authorization
	jwtSecret string
	logger    *zap.SugaredLogger
}

func NewAuthService(repo repository.Authorization, secret string, logger *zap.SugaredLogger) *AuthService {
	logger = logger.With("component", "auth", "layer", "service")

	return &AuthService{
		repo:      repo,
		jwtSecret: secret,
		logger:    logger,
	}
}

type AuthData struct {
	ID        uuid.UUID
	JWT       JWTToken
	KDFSalt   []byte
	KDFParams models.Params
}

func (s *AuthService) RegisterUser(ctx context.Context, login string, password string) (AuthData, error) {
	passHash, err := generatePasswordHash(password)
	if err != nil {
		s.logger.Errorw("couldn't generate password hash",
			"op", "auth.user_register",
			"error", err,
		)

		return AuthData{}, err
	}

	salt, params, err := s.generateKDF()
	if err != nil {
		s.logger.Warnw("registration failed",
			"op", "auth.user_register",
			"login", login,
			"error", repository.ErrUserExists,
		)

		return AuthData{}, err
	}

	user := models.User{
		Login:        login,
		PasswordHash: passHash,
		KdfSalt:      salt,
		KdfParams:    params,
	}

	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	id, err := s.repo.RegisterUser(ctx, user)
	if err != nil {
		if err == repository.ErrUserExists {
			s.logger.Warnw("registration failed",
				"op", "auth.user_register",
				"login", login,
				"error", repository.ErrUserExists,
			)

			return AuthData{}, repository.ErrUserExists
		}

		s.logger.Errorw("registration failed",
			"op", "auth.user_register",
			"login", login,
			"error", err,
		)

		return AuthData{}, fmt.Errorf("user register error: %w", err)
	}

	token, err := s.GenerateToken(id)
	if err != nil {
		s.logger.Errorw("token generated error",
			"op", "auth.user_register",
			"login", login,
			"error", err,
		)
		return AuthData{}, fmt.Errorf("token generated error: %w", err)
	}

	data := AuthData{
		ID:        id,
		JWT:       token,
		KDFSalt:   salt,
		KDFParams: params,
	}

	s.logger.Infow("user registered",
		"op", "auth.user_register",
		"login", login,
		"user_id", id,
	)

	return data, nil
}

func generatePasswordHash(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", err
	}

	return string(hash), nil
}

var ErrInvalidCredentials = errors.New("invalid login or password")

func (s *AuthService) LoginUser(ctx context.Context, login string, password string) (AuthData, error) {
	ctx, cancel := context.WithTimeout(ctx, time.Second*3)
	defer cancel()

	userDB, err := s.repo.LoginUser(ctx, login)
	if err != nil {
		if err == repository.ErrUserNotFound {
			s.logger.Warnw("invalid credentials",
				"op", "auth.user_login",
				"login", login,
				"error", err,
			)

			return AuthData{}, repository.ErrUserNotFound
		}

		s.logger.Errorw("couldn't get user from db",
			"op", "auth.user_login",
			"login", login,
			"error", err,
		)

		return AuthData{}, fmt.Errorf("user login error: %w", err)
	}

	if err := bcrypt.CompareHashAndPassword([]byte(userDB.PasswordHash), []byte(password)); err != nil {
		s.logger.Warnw("invalid credentials",
			"op", "auth.user_login",
			"login", userDB.Login,
			"error", err,
		)

		return AuthData{}, ErrInvalidCredentials
	}

	token, err := s.GenerateToken(userDB.ID)
	if err != nil {
		s.logger.Errorw("generate token error",
			"op", "auth.user_login",
			"login", login,
			"error", err,
		)
		return AuthData{}, fmt.Errorf("generate token error: %w", err)
	}

	data := AuthData{
		ID:        userDB.ID,
		JWT:       token,
		KDFSalt:   userDB.KdfSalt,
		KDFParams: userDB.KdfParams,
	}

	s.logger.Infow("user logged in",
		"op", "auth.user_login",
		"user_id", userDB.ID,
		"login", userDB.Login,
	)

	return data, nil
}

type JWTToken struct {
	AccessToken string
	ExpiresAt   time.Time
}

type Claims struct {
	UserID uuid.UUID `json:"id"`
	jwt.RegisteredClaims
}

func (s *AuthService) GenerateToken(id uuid.UUID) (accessToken JWTToken, err error) {
	expTime := time.Now().Add(15 * time.Minute)
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, Claims{
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			Issuer:    "go-data-keeper",
		},
		UserID: id,
	})

	tokenStr, err := token.SignedString([]byte(s.jwtSecret))
	if err != nil {
		s.logger.Errorw("generate token error",
			"op", "auth.generate_token",
			"error", err,
		)

		return JWTToken{}, fmt.Errorf("generate token error: %w", err)
	}

	return JWTToken{
		AccessToken: tokenStr,
		ExpiresAt:   expTime,
	}, nil
}

var kdf_params = models.Params{
	Algorithm:   "argon2id",
	KeyLen:      32,
	SaltLen:     16,
	Time:        3,
	Memory:      64 * 1024,
	Parallelism: 2,
}

func (s *AuthService) generateKDF() (kdfSalt []byte, kdfParams models.Params, err error) {
	kdfSalt = make([]byte, kdf_params.SaltLen)
	_, err = rand.Read(kdfSalt)
	if err != nil {
		s.logger.Warnw("generate salt error", "op", "generate_kdf")
		return nil, models.Params{}, err
	}

	kdfParams = kdf_params

	return kdfSalt, kdfParams, nil
}
