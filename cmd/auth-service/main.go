package main

import (
	"log"
	"net/http"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jeremzhg/go-auth/internal/configs"
	"github.com/jeremzhg/go-auth/internal/handlers"
	"time"
	mw "github.com/jeremzhg/go-auth/internal/middleware"
	"github.com/joho/godotenv"
	"github.com/casbin/casbin/v2"
	"fmt"
	"github.com/jmoiron/sqlx"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
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
	var db *sqlx.DB
	for i := 1; i <= 10; i++ {
			db, err = sqlx.Open("pgx", cfg.DSN)
			if err == nil {
					err = db.Ping()
			}
			if err == nil {
					log.Println("successfully connected to db")
					break
			}
			log.Printf("attempt %d: failed to connect to db, error: %v", i, err)
			time.Sleep(2 * time.Second)
	}

	if err != nil {
			log.Fatalf("Could not connect to database after multiple retries: %v", err)
	}
	defer db.Close()

	log.Println("Running database migrations...")
	m, err := migrate.New("file://migrations", cfg.DSN)
	if err != nil {
			log.Fatalf("could not create migrator: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
			log.Fatalf("could not run migrations: %v", err)
	}
	log.Println("migrations completed successfully.")

	enforcer, err := setupEnforcer(db)
	if err != nil {
			log.Fatalf("failed to create casbin enforcer: %v", err)
	}

	policyHandler := &handlers.PolicyHandler{Enforcer: enforcer}
	r := chi.NewRouter()
	r.Use(middleware.Logger)

	r.Post("/check", policyHandler.Check)
	r.Group(func(r chi.Router) {
    r.Use(mw.APIKeyAuth(cfg.APIKey)) 
    r.Post("/policies", policyHandler.CreatePolicyHandler)
	})

	log.Printf("starting server on %s", cfg.Port)
	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
