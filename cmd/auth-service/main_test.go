package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"

	"github.com/casbin/casbin/v2"
	"github.com/go-chi/chi/v5"
	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/jmoiron/sqlx"
	mw "github.com/jeremzhg/go-auth/internal/middleware"
	"github.com/jeremzhg/go-auth/internal/handlers"
	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	sqlxadapter "github.com/memwey/casbin-sqlx-adapter"
)

var testDB *sqlx.DB

func TestMain(m *testing.M) {
	godotenv.Load("../../.env")
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		log.Fatal("TEST_DB_DSN not set in .env file")
	}

	var err error
	testDB, err = sqlx.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open test database connection: %v", err)
	}
	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}

	migrator, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		log.Fatalf("could not create migrator: %v", err)
	}
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("could not run migrations up: %v", err)
	}

	code := m.Run()

	migrator.Down()
	testDB.Close()
	os.Exit(code)
}

func newTestApp(t *testing.T) *handlers.PolicyHandler {
	_, err := testDB.Exec("TRUNCATE policies RESTART IDENTITY")
	if err != nil {
		t.Fatalf("could not truncate policies table: %v", err)
	}

	enforcer, err := setEnforcer(testDB)
	if err != nil {
		t.Fatalf("failed to create enforcer for test: %v", err)
	}

	policyHandler := &handlers.PolicyHandler{Enforcer: enforcer}
	return policyHandler
}

func TestCreatePolicyEndpoint(t *testing.T) {
	policyHandler := newTestApp(t)
	router := chi.NewRouter()
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
			t.Fatal("API_KEY is missing");
	}
		
	router.Use(mw.APIKeyAuth(apiKey))

	router.Post("/policies", policyHandler.CreatePolicyHandler)
	payload := `{"subject": "user:test", "action": "read", "object": "resource:123"}`
	req := httptest.NewRequest(http.MethodPost, "/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+apiKey)
	rr := httptest.NewRecorder()

	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d, body=%s", rr.Code, rr.Body.String())
	}

	var count int

	err := testDB.Get(&count, `SELECT COUNT(*) FROM policies WHERE v0=$1 AND v2=$2 AND v1=$3`,
		"user:test", "read", "resource:123")
	if err != nil {
		t.Fatalf("failed to query policies table: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 policy inserted, got %d", count)
	}
}

func TestCheckEndpoint(t *testing.T) {
	policyHandler := newTestApp(t)
	router := chi.NewRouter()
	router.Post("/check", policyHandler.Check)
	
	_, err := policyHandler.Enforcer.AddPolicy("user:test", "resource:1", "read")
	if err != nil {
		t.Fatalf("failed to seed test policy: %v", err)
	}
    
	// --- Test the "Allow" Case ---
	allowPayload := `{"subject": "user:test", "object": "resource:1", "action": "read"}`
	req := httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(allowPayload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	router.ServeHTTP(rr, req)

	if rr.Code != http.StatusOK {
		t.Fatalf("allow case: expected status 200 OK, got %d", rr.Code)
	}

	var respBody struct {
		Allowed bool `json:"allowed"`
	}
	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("allow case: failed to decode response body: %v", err)
	}
	if !respBody.Allowed {
		t.Errorf("allow case: expected allowed=true, got false")
	}

	// --- Test the "Deny" Case ---
	denyPayload := `{"subject": "user:scammer", "object": "resource:1", "action": "read"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(denyPayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder()
	router.ServeHTTP(rr, req)
	
	if rr.Code != http.StatusOK {
		t.Fatalf("deny case: expected status 200 OK, got %d", rr.Code)
	}

	if err := json.NewDecoder(rr.Body).Decode(&respBody); err != nil {
		t.Fatalf("deny case: failed to decode response body: %v", err)
	}
	if respBody.Allowed {
		t.Errorf("deny case: expected allowed=false, got true")
	}
}

// setupEnforcer is a helper from main, duplicated here for testing.
func setEnforcer(db *sqlx.DB) (*casbin.Enforcer, error) {
	opts := &sqlxadapter.AdapterOptions{
		DB:        db,
		TableName: "policies",
	}
	adapter := sqlxadapter.NewAdapterFromOptions(opts)

	enforcer, err := casbin.NewEnforcer("../../model.conf", adapter)
	if err != nil {
		return nil, fmt.Errorf("failed to create casbin enforcer: %w", err)
	}
	// Note: We don't call LoadPolicy here because the table is truncated clean by newTestApp.
	// The enforcer starts empty, and we add policies to it directly in the test.
	return enforcer, nil
}