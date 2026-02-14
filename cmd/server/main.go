package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"
	chiMiddleware "github.com/go-chi/chi/v5/middleware"
	_ "github.com/makimaki04/go-data-keeper.git/docs"
	"github.com/makimaki04/go-data-keeper.git/internal/config"
	"github.com/makimaki04/go-data-keeper.git/internal/database"
	"github.com/makimaki04/go-data-keeper.git/internal/handler"
	"github.com/makimaki04/go-data-keeper.git/internal/logger"
	"github.com/makimaki04/go-data-keeper.git/internal/middleware"
	"github.com/makimaki04/go-data-keeper.git/internal/repository"
	"github.com/makimaki04/go-data-keeper.git/internal/service"
	httpSwagger "github.com/swaggo/http-swagger/v2"
	"go.uber.org/zap"
)

var loggerCfgPath = "configs/logger.json"

func main() {
	logger, err := logger.NewLogger(loggerCfgPath)
	if err != nil {
		log.Fatalf("failed to init logger: %v", err)
	}

	cfg, err := config.InitConfig(logger)
	if err != nil {
		logger.Fatalw("config init error", "layer", "main", "err", err)
	}

	db, err := database.InitDB(cfg.DatabaseURI, logger)
	if err != nil {
		logger.Fatalw("db init error", "layer", "main", "err", err)
	}

	repo := repository.NewRepository(db, logger)
	service := service.NewService(repo, cfg.JWTSecret, logger)
	handler := handler.NewHandler(service, logger)

	r := chi.NewRouter()
	r.Use(chiMiddleware.RequestID)
	r.Use(middleware.WithLogging(logger))

	r.Route("/", func(r chi.Router) {
		r.Route("/api/user", func(r chi.Router) {
			r.Post("/register", handler.RegisterUser)
			r.Post("/login", handler.LoginUser)
			r.Group(func(r chi.Router) {
				r.Use(middleware.WithAuth(cfg.JWTSecret, logger))
				r.Route("/items", func(r chi.Router) {
					r.Get("/", handler.GetAllItems)
					r.Get("/{id}", handler.GetItem)
					r.Put("/{id}", handler.SetItem)
					r.Delete("/{id}", handler.DeleteItem)
				})
				r.Route("/sync", func(r chi.Router) {
					r.Get("/changes", handler.GetChangesSince)
				})
			})
		})
		r.Route("/swagger", func(r chi.Router) {
			r.Get("/*", httpSwagger.WrapHandler)
		})
	})

	APIServer := &http.Server{
		Addr:    cfg.Address,
		Handler: r,
	}

	go func() {
		if err := runServer(APIServer); err != nil {
			logger.Fatalw("http server startup error", "address", cfg.Address, "err", err)
		}
	}()

	gracefulStop(APIServer, logger, 5*time.Second)
}

func runServer(srv *http.Server) error {
	err := srv.ListenAndServe()

	if err != nil && !errors.Is(err, http.ErrServerClosed) {
		return err
	}
	return nil
}

func gracefulStop(srv *http.Server, logger *zap.SugaredLogger, timeout time.Duration) {
	ctx, stop := signal.NotifyContext(context.Background(), syscall.SIGINT, syscall.SIGTERM)
	defer stop()

	<-ctx.Done()

	logger.Infow("received shutdown signal", "signal", ctx.Err())

	shutdownCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		logger.Errorw("http server gracefull shutdown error", "err", err)
		return
	}

	logger.Info("http server stopped gracefully")
}
