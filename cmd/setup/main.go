package main

import (
	"context"
	"log"

	"github.com/shellhaki/spark-whatsapp-module/internal/config"
	"github.com/shellhaki/spark-whatsapp-module/internal/database"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	ctx := context.Background()

	sqliteDB, err := database.OpenSQLite(ctx, cfg.SQLitePath, database.PoolOptions{
		MaxOpenConns:    1,
		MaxIdleConns:    1,
		ConnMaxIdleTime: cfg.DBConnMaxIdle,
		ConnMaxLifetime: cfg.DBConnMaxLife,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer sqliteDB.Close()

	if err := database.SetupSQLite(ctx, sqliteDB); err != nil {
		log.Fatal(err)
	}

	if cfg.PostgresURI != "" {
		postgresDB, err := database.OpenPostgres(ctx, cfg.PostgresURI, database.PoolOptions{
			MaxOpenConns:    cfg.DBMaxOpenConns,
			MaxIdleConns:    cfg.DBMaxIdleConns,
			ConnMaxIdleTime: cfg.DBConnMaxIdle,
			ConnMaxLifetime: cfg.DBConnMaxLife,
		})
		if err != nil {
			log.Fatal(err)
		}
		defer postgresDB.Close()

		if err := database.SetupPostgres(ctx, postgresDB); err != nil {
			log.Fatal(err)
		}
	}

	log.Println("database setup completed for sqlite and postgres backup")
}
