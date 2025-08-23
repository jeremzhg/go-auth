package main

import (
	"log"
	"net/http"
	"github.com/jeremzhg/go-auth/internal/configs"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"
)

func main() {
	if err := godotenv.Load(); err != nil{
		log.Fatalf("failed to load env: %v", err)
	}

	cfg, err := configs.Load()
	if err != nil{
		log.Fatalf("failed to load configs: %v", err)
	}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Get("/", func(w http.ResponseWriter, r *http.Request) {
		if _, err := w.Write([]byte("hello world")); err != nil {
			http.Error(w, "failed to write response", http.StatusInternalServerError)
		}
	})

	if err := http.ListenAndServe(cfg.Port, r); err != nil {
		log.Fatal(err)
	}
}
