package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jeremzhg/go-auth/internal/configs"
	"github.com/jeremzhg/go-auth/internal/handlers"
	"github.com/jeremzhg/go-auth/internal/repository"
	"github.com/joho/godotenv"
)

type Application struct{
	DB *sql.DB
}

func (app *Application) getHelloWorldHandler(w http.ResponseWriter, r *http.Request) {
    w.Write([]byte("hello from the app struct"))
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
	db, err := sql.Open("pgx", cfg.DSN)
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
