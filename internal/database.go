package internal

import (
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/golang-migrate/migrate/v4"
	migratepg "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func ConnectDB(url string) (*gorm.DB, error) {
	db, err := gorm.Open(postgres.Open(url), &gorm.Config{
		Logger: logger.New(log.New(os.Stdout, "", log.LstdFlags), logger.Config{
			SlowThreshold: 200 * time.Millisecond,
			LogLevel:      logger.Warn,
			// "Not found" is a normal answer (e.g. unknown email), not a problem.
			IgnoreRecordNotFoundError: true,
		}),
		// Turns unique violations into gorm.ErrDuplicatedKey.
		TranslateError: true,
	})
	if err != nil {
		return nil, fmt.Errorf("connect: %w", err)
	}
	return db, nil
}

// Migrate applies all pending *.up.sql files from dir.
func Migrate(db *gorm.DB, dir string) error {
	sqlDB, err := db.DB()
	if err != nil {
		return err
	}

	driver, err := migratepg.WithInstance(sqlDB, &migratepg.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}

	abs, err := filepath.Abs(dir)
	if err != nil {
		return err
	}

	m, err := migrate.NewWithDatabaseInstance("file://"+abs, "postgres", driver)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}

	if err := m.Up(); err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return fmt.Errorf("migrate up: %w", err)
	}
	return nil
}
