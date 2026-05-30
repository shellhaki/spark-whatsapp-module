package main

import (
	"context"
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

	repo := subscribers.NewRepository(db)

	whatsAppService, err := whatsapp.NewService(ctx, cfg, db, repo)
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
