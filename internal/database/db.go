// Package database initializes database connections and migrations.
package database

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/makimaki04/go-data-keeper.git/internal/migrations"
	"go.uber.org/zap"
)

// InitDB runs migrations, opens a database connection, and verifies connectivity.
// InitDB returns an error if migrations fail, the connection can't be opened, or the database can't be reached.
func InitDB(dsn string, logger *zap.SugaredLogger) (*sql.DB, error) {
	logger = logger.With("component", "database", "op", "db.init_db")

	if err := migrations.RunMigration(dsn, logger); err != nil {
		logger.Errorw("run migration error", "error", err)

		return nil, fmt.Errorf("run migration error: %w", err)
	}	

	logger.Info("migrations successfully started")

	db, err := sql.Open("pgx", dsn)
	if err != nil {
		logger.Errorw("open db error", "error", err)

		return nil, fmt.Errorf("db open error: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	err = db.PingContext(ctx)
	if err != nil {
		logger.Errorw("db connection error", "error", err)
		return nil, fmt.Errorf("db connection error: %w", err)
	}

	logger.Info("database successfully init")

	return db, nil
}
