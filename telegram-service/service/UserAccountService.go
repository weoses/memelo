package service

import (
	"context"
	"embed"
	"errors"
	"fmt"
	"log/slog"

	"github.com/golang-migrate/migrate/v4"
	migratepgx "github.com/golang-migrate/migrate/v4/database/pgx/v5"
	"github.com/golang-migrate/migrate/v4/source/iofs"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/weoses/memelo/telegram-service/conf"
)

//go:embed migrations/*.sql
var migrationsFS embed.FS

type UserAccountService interface {
	MapUserToAccount(ctx context.Context, userId int64) (uuid.UUID, error)
}

type UserAccountServiceImpl struct {
	pool *pgxpool.Pool
	log  *slog.Logger
}

// MapUserToAccount implements UserAccountService.
func (u *UserAccountServiceImpl) MapUserToAccount(ctx context.Context, userId int64) (uuid.UUID, error) {
	newId, _ := uuid.NewRandom()
	_, err := u.pool.Exec(ctx,
		`INSERT INTO tg_user_account_bindings (telegram_id, account_id)
		 VALUES ($1, $2) ON CONFLICT (telegram_id) DO NOTHING`,
		userId, newId)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("insert: %w", err)
	}

	var accountIdStr string
	err = u.pool.QueryRow(ctx,
		`SELECT account_id FROM tg_user_account_bindings WHERE telegram_id = $1`,
		userId).Scan(&accountIdStr)
	if err != nil {
		return uuid.UUID{}, fmt.Errorf("select: %w", err)
	}
	return uuid.Parse(accountIdStr)
}

type UserAccountServiceStaticImpl struct {
	staticUuid uuid.UUID
	log        *slog.Logger
}

// MapUserToAccount implements UserAccountService.
func (u *UserAccountServiceStaticImpl) MapUserToAccount(ctx context.Context, userId int64) (uuid.UUID, error) {
	return u.staticUuid, nil
}

func runMigrations(pool *pgxpool.Pool) error {
	src, err := iofs.New(migrationsFS, "migrations")
	if err != nil {
		return fmt.Errorf("iofs source: %w", err)
	}
	db := stdlib.OpenDBFromPool(pool)
	drv, err := migratepgx.WithInstance(db, &migratepgx.Config{})
	if err != nil {
		return fmt.Errorf("migrate driver: %w", err)
	}
	m, err := migrate.NewWithInstance("iofs", src, "postgres", drv)
	if err != nil {
		return fmt.Errorf("migrate init: %w", err)
	}
	err = m.Up()
	if err != nil && !errors.Is(err, migrate.ErrNoChange) {
		return err
	}
	return nil
}

func NewUserAccountService(config *conf.Config) (UserAccountService, error) {
	logger := slog.With("service", "UserAccountService")

	if config.UserAccount.StaticUuid != "" {
		return &UserAccountServiceStaticImpl{
			staticUuid: uuid.MustParse(config.UserAccount.StaticUuid),
			log:        logger,
		}, nil
	}

	pool, err := pgxpool.New(context.Background(), config.Postgres.DSN)
	if err != nil {
		return nil, fmt.Errorf("pgxpool.New: %w", err)
	}

	if err := runMigrations(pool); err != nil {
		return nil, fmt.Errorf("migrations: %w", err)
	}

	return &UserAccountServiceImpl{
		pool: pool,
		log:  logger,
	}, nil
}
