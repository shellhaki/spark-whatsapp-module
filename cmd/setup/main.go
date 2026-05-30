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

	db, err := database.Open(ctx, cfg.PostgresURI, database.PoolOptions{
		MaxOpenConns:    cfg.DBMaxOpenConns,
		MaxIdleConns:    cfg.DBMaxIdleConns,
		ConnMaxIdleTime: cfg.DBConnMaxIdle,
		ConnMaxLifetime: cfg.DBConnMaxLife,
	})
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	if err := database.Setup(ctx, db); err != nil {
		log.Fatal(err)
	}

	log.Println("database setup completed")
}
