package main

import (
	"context"
	"database/sql"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/shellhaki/spark-whatsapp-module/internal/app"
	"github.com/shellhaki/spark-whatsapp-module/internal/config"
	"github.com/shellhaki/spark-whatsapp-module/internal/database"
	"github.com/shellhaki/spark-whatsapp-module/internal/httpapi"
	"github.com/shellhaki/spark-whatsapp-module/internal/subscribers"
	"github.com/shellhaki/spark-whatsapp-module/internal/whatsapp"
)

func main() {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

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

	var postgresDB *sql.DB
	if cfg.PostgresURI != "" {
		postgresDB, err = database.OpenPostgres(ctx, cfg.PostgresURI, database.PoolOptions{
			MaxOpenConns:    cfg.DBMaxOpenConns,
			MaxIdleConns:    cfg.DBMaxIdleConns,
			ConnMaxIdleTime: cfg.DBConnMaxIdle,
			ConnMaxLifetime: cfg.DBConnMaxLife,
		})
		if err != nil {
			log.Printf("postgres backup disabled: %v", err)
		} else {
			defer postgresDB.Close()
			if err := database.SetupPostgres(ctx, postgresDB); err != nil {
				log.Printf("postgres backup setup failed: %v", err)
				postgresDB.Close()
				postgresDB = nil
			}
		}
	}

	repo := subscribers.NewRepository(sqliteDB, postgresDB)

	whatsAppService, err := whatsapp.NewService(ctx, cfg, sqliteDB, repo)
	if err != nil {
		log.Fatal(err)
	}
	defer whatsAppService.Close()

	application := app.New(cfg, repo, whatsAppService)

	server := &http.Server{
		Addr:    cfg.HTTPAddress,
		Handler: httpapi.NewRouter(application),
	}

	go func() {
		log.Printf("spark-whatsapp-module listening on %s", cfg.HTTPAddress)
		if err := server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			log.Printf("http server error: %v", err)
			stop()
		}
	}()

	<-ctx.Done()

	shutdownCtx, cancel := context.WithTimeout(context.Background(), cfg.ShutdownTimeout)
	defer cancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("http shutdown error: %v", err)
	}
}
