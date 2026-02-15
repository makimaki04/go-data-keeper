// Package migrations provides embedded database migration execution.
package migrations

import (
	"embed"
	"errors"
	"fmt"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"go.uber.org/zap"
)

//go:embed migration_files/*.sql
var migrationsDir embed.FS

// RunMigration applies embedded SQL migrations to the database referenced by dsn.
// RunMigration returns an error if migrations can't be initialized or applied.
func RunMigration(dsn string, logger *zap.SugaredLogger) error {
	d, err := iofs.New(migrationsDir, "migration_files")
	if err != nil {
		logger.Errorw("return FS driver error", "error", err)

		return fmt.Errorf("failed to return FS driver: %w", err)
	}

	m, err := migrate.NewWithSourceInstance("iofs", d, dsn)
	if err != nil {
		logger.Errorw("get migrate instance error", "error", err)

		return fmt.Errorf("failed to get migrate instance: %w", err)
	}
	defer func() {
		srcErr, dbErr := m.Close()
		if srcErr != nil {
			logger.Warnw("migarte source close error", "error", srcErr)
		}
		if dbErr != nil {
			logger.Warnw("migrate db close error", "error", dbErr)
		}
	}()

	if err := m.Up(); err != nil {
		if !errors.Is(err, migrate.ErrNoChange) {
			logger.Errorw("load migration error", "error", err)

			return fmt.Errorf("failed to load migrations: %w", err)
		}
	}

	return nil
}
