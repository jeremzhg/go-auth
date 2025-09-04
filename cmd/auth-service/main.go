package main

import (
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jeremzhg/go-auth/internal/configs"
	"github.com/jeremzhg/go-auth/internal/handlers"
	"github.com/jeremzhg/go-auth/internal/repository"
	"github.com/joho/godotenv"
	"github.com/casbin/casbin/v2"
	"fmt"
	"github.com/jmoiron/sqlx"
	sqlxadapter "github.com/memwey/casbin-sqlx-adapter"
)


func setupEnforcer(db *sqlx.DB) (*casbin.Enforcer, error) {
    opts := &sqlxadapter.AdapterOptions{
        DB:        db,
        TableName: "policies",
    }

    adapter := sqlxadapter.NewAdapterFromOptions(opts)

    enforcer, err := casbin.NewEnforcer("model.conf", adapter)
    if err != nil {
        return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
    }

    if err := enforcer.LoadPolicy(); err != nil {
        return nil, fmt.Errorf("failed to load casbin policy: %w", err)
    }

    return enforcer, nil
}
func main() {
	// if err := godotenv.Load(); err != nil{
	// 	log.Fatalf("failed to load env: %v", err)
	// }
	godotenv.Load()
	cfg, err := configs.Load()
	if err != nil{
		log.Fatalf("failed to load configs: %v", err)
	}
	db, err := sqlx.Open("pgx", cfg.DSN)
	if err != nil {
		log.Fatalf("failed to open database connection: %v", err)
	}
	if err := db.Ping(); err != nil {
    log.Fatalf("failed to ping database: %v", err)
	}
	log.Println("database connection successful")
	defer db.Close()

	policyRepo := &repository.PostgresPolicyRepo{DB: db}
	policyHandler := handlers.PolicyHandler{Repo: policyRepo}
	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello world")); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})
	r.Post("/policies", policyHandler.CreatePolicyHandler)
	log.Printf("starting server on %s", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
