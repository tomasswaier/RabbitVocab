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

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/joho/godotenv"
	"github.com/tomasswaier/RabbitVocab/internal/db/sqlc"
	apikeydomain "github.com/tomasswaier/RabbitVocab/internal/domain/apikey"
	"github.com/tomasswaier/RabbitVocab/internal/domain/language"
	"github.com/tomasswaier/RabbitVocab/internal/domain/oauth"
	"github.com/tomasswaier/RabbitVocab/internal/domain/session"
	"github.com/tomasswaier/RabbitVocab/internal/domain/user"
	"github.com/tomasswaier/RabbitVocab/internal/domain/word"
	"github.com/tomasswaier/RabbitVocab/internal/domain/wordform"
	vocabhttp "github.com/tomasswaier/RabbitVocab/internal/http"
	"github.com/tomasswaier/RabbitVocab/internal/http/handler"
)

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("no .env file found, relying on environment variables")
	}

	if err := run(); err != nil {
		log.Fatalf("fatal: %v", err)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	dbURL := getEnv("DATABASE_URL", "postgres://postgres@/vocab?host=/run/postgresql")
	port := getEnv("PORT", "8080")
	webDir := getEnv("WEB_DIR", "web")

	pool, err := connectDB(ctx, dbURL)
	if err != nil {
		return err
	}
	defer pool.Close()

	log.Println("connected to postgres")

	queries := sqlc.New(pool)

	userRepo := user.NewPostgresRepository(queries)
	languageRepo := language.NewPostgresRepository(queries)
	wordRepo := word.NewPostgresRepository(queries)
	wordService := word.NewService(wordRepo, languageRepo)
	wordFormRepo := wordform.NewPostgresRepository(queries)
	wordFormService := wordform.NewService(wordFormRepo, languageRepo)
	sessionRepo := session.NewPostgresRepository(queries)
	sessionService := session.NewService(sessionRepo)
	apiKeyRepo := apikeydomain.NewPostgresRepository(queries)
	oauthRepo := oauth.NewPostgresRepository(queries)
	publicBaseURL := getEnv("PUBLIC_BASE_URL", "http://localhost:8080")

	handlers := vocabhttp.Handlers{
		Auth:     handler.NewAuthHandler(userRepo, languageRepo, apiKeyRepo, sessionService),
		Language: handler.NewLanguageHandler(languageRepo),
		Word:     handler.NewWordHandler(wordService),
		WordForm: handler.NewWordFormHandler(wordFormService),
		OAuth:    handler.NewOAuthHandler(oauthRepo, apiKeyRepo, userRepo, publicBaseURL),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /healthz", healthzHandler(pool))
	vocabhttp.RegisterRoutes(mux, handlers, apiKeyRepo, sessionService)
	mux.Handle("GET /", http.FileServer(http.Dir(webDir)))
	srv := &http.Server{
		Addr:              ":" + port,
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
	}

	errCh := make(chan error, 1)
	go func() {
		log.Printf("api listening on :%s", port)
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			errCh <- err
		}
		close(errCh)
	}()

	select {
	case err := <-errCh:
		if err != nil {
			return err
		}
	case <-ctx.Done():
		log.Println("shutdown signal received")
	}

	shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		return err
	}

	log.Println("server stopped cleanly")
	return nil
}

// connectDB opens a pgx connection pool and verifies connectivity with a ping,
// retrying briefly since Postgres in a fresh local setup may still be starting.
func connectDB(ctx context.Context, dbURL string) (*pgxpool.Pool, error) {
	cfg, err := pgxpool.ParseConfig(dbURL)
	if err != nil {
		return nil, err
	}

	pool, err := pgxpool.NewWithConfig(ctx, cfg)
	if err != nil {
		return nil, err
	}

	pingCtx, cancel := context.WithTimeout(ctx, 3*time.Second)
	defer cancel()

	var pingErr error
	for i := 0; i < 5; i++ {
		if pingErr = pool.Ping(pingCtx); pingErr == nil {
			return pool, nil
		}
		time.Sleep(time.Second)
	}

	pool.Close()
	return nil, pingErr
}

func healthzHandler(pool *pgxpool.Pool) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()

		if err := pool.Ping(ctx); err != nil {
			http.Error(w, "db unreachable", http.StatusServiceUnavailable)
			return
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok"))
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
