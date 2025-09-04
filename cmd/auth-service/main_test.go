package main

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"

	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/joho/godotenv"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
  "github.com/jmoiron/sqlx"
	"github.com/jeremzhg/go-auth/internal/handlers"
	"github.com/jeremzhg/go-auth/internal/repository"
)

func newTestDB(t *testing.T) *sqlx.DB {
    dsn := os.Getenv("TEST_DB_DSN")
    if dsn == "" {
        godotenv.Load("../../.env") 
        dsn = os.Getenv("TEST_DB_DSN")
    }

    db, err := sqlx.Open("pgx", dsn)
    if err != nil {
        t.Fatalf("failed to open database connection: %v", err)
    }

    if err := db.Ping(); err != nil {
        t.Fatalf("failed to ping database: %v", err)
    }

    t.Cleanup(func() {
        db.Close()
    })
    
    m, err := migrate.New("file://../../migrations", dsn)
    if err != nil {
        t.Fatalf("could not create migrator: %v", err)
    }

    if err := m.Up(); err != nil && err != migrate.ErrNoChange {
        t.Fatalf("could not run migrations up: %v", err)
    }

    // t.Cleanup(func() {
    //     if err := m.Down(); err != nil {
    //         t.Fatalf("could not run migrations down: %v", err)
    //     }
    // })

    return db
}
func TestDBConnection(t *testing.T) {
	db := newTestDB(t)
	if err := db.Ping(); err != nil {
		t.Fatalf("failed to ping database: %v", err)
	}
	t.Log("database connection successful")
}

func TestCreatePolicyEndpoint(t *testing.T) {
	db := newTestDB(t)

	policyRepo := &repository.PostgresPolicyRepo{DB: db}
	policyHandler := &handlers.PolicyHandler{Repo: policyRepo}

	r := chi.NewRouter()
	r.Use(middleware.Logger)
	r.Post("/policies", policyHandler.CreatePolicyHandler)
	
	payload := map[string]string{
		"subject": "user:test",
		"action":  "read",
		"object":  "resource:123",
	}
	body, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("failed to marshal payload: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/policies", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")

	rr := httptest.NewRecorder()
	r.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var count int
	err = db.QueryRow(`SELECT COUNT(*) FROM policies WHERE subject=$1 AND action=$2 AND object=$3`,
	payload["subject"], payload["action"], payload["object"]).Scan(&count)
	if err != nil {
		t.Fatalf("failed to query policies table: %v", err)
	}

	if count != 1 {
		t.Fatalf("expected 1 policy inserted, got %d", count)
	}
}