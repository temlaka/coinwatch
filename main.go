package main

import (
	"context"
	"errors"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/temlaka/coinwatch/internal/client/coingecko"
	"github.com/temlaka/coinwatch/internal/handler"
	"github.com/temlaka/coinwatch/internal/repository/postgres"
	"github.com/temlaka/coinwatch/internal/scheduler"
	"github.com/temlaka/coinwatch/internal/usecase/auth"
	"github.com/temlaka/coinwatch/internal/usecase/crypto"
	"github.com/temlaka/coinwatch/internal/usecase/schedule"
	"github.com/temlaka/coinwatch/migrations"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/jackc/pgx/v5/stdlib"
	"github.com/joho/godotenv"
	"github.com/pressly/goose/v3"
)

func main() {

	err := godotenv.Load()
	if err != nil {
		log.Println(".env not found, using environment variables")
	}
	JWT_SECRET := os.Getenv("JWT_SECRET")
	COINGECKO_API_KEY := os.Getenv("COINGECKO_API_KEY")
	DATABASE_URL := os.Getenv("DATABASE_URL")
	if JWT_SECRET == "" {
		log.Fatal("secret is none")
	}
	if COINGECKO_API_KEY == "" {
		log.Fatal("api key is none")
	}
	if DATABASE_URL == "" {
		log.Fatal("database url is none")
	}
	pool, err := pgxpool.New(context.Background(), DATABASE_URL)
	if err != nil {
		log.Fatalf("parse database URL: %v", err)
	}
	if err := pool.Ping(context.Background()); err != nil {
		log.Fatalf("connect to database: %v", err)
	}
	defer pool.Close()

	db := stdlib.OpenDBFromPool(pool)
	provider, err := goose.NewProvider(goose.DialectPostgres, db, migrations.FS)
	if err != nil {
		log.Fatalf("failed to create goose provider: %v", err)
	}
	migres, err := provider.Up(context.Background())
	if err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}
	for _, m := range migres {
		log.Printf("applied migration: %s", m)
	}
	log.Printf("migrations: %d applied", len(migres))
	cgAPI := coingecko.NewClient("https://api.coingecko.com/api/v3", COINGECKO_API_KEY)

	userRepo := postgres.NewUserRepo(pool)
	cryptoRepo := postgres.NewCryptoRepo(pool, 100)
	scheduleRepo := postgres.NewScheduleRepo(pool)

	authSvc := auth.NewService(userRepo, JWT_SECRET)
	authHandler := handler.NewAuthHandler(authSvc)

	cryptoSvc := crypto.NewService(cryptoRepo, cgAPI)
	cryptoHandler := handler.NewCryptoHandler(cryptoSvc)

	scheduleSvc := schedule.NewService(scheduleRepo, cryptoSvc)
	scheduleHandler := handler.NewScheduleHandler(scheduleSvc)

	mux := handler.Routes(authHandler, cryptoHandler, scheduleHandler, JWT_SECRET)

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	go scheduler.UpdateAllPrice(ctx, scheduleSvc)

	server := &http.Server{
		Addr:              ":8080",
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	go func() {
		log.Println("server started on :8080")
		err := server.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			return
		}
		log.Printf("server error: %v", err)
		stop()
	}()

	<-ctx.Done()
	log.Println("shutting down...")

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("shutdown: %v", err)
	}
	log.Println("server stopped")
}
