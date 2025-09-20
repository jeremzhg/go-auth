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
	"github.com/jeremzhg/go-auth/internal/handlers"
	"github.com/jeremzhg/go-auth/internal/models"
	"github.com/jeremzhg/go-auth/internal/repository"
	"github.com/joho/godotenv"
	_ "github.com/jackc/pgx/v5/stdlib"
	sqlxadapter "github.com/memwey/casbin-sqlx-adapter"
)

// testDB is a global database connection pool for the test suite.
var testDB *sqlx.DB

// TestMain runs once before all tests in the package. It's used for
// expensive setup and teardown, like creating a DB connection and running migrations.
func TestMain(m *testing.M) {
	// Load .env from the project root.
	godotenv.Load("../../.env")
	dsn := os.Getenv("TEST_DB_DSN")
	if dsn == "" {
		log.Fatal("TEST_DB_DSN not set in .env file")
	}

	// Connect to the test database.
	var err error
	testDB, err = sqlx.Open("pgx", dsn)
	if err != nil {
		log.Fatalf("failed to open test database connection: %v", err)
	}
	if err := testDB.Ping(); err != nil {
		log.Fatalf("failed to ping test database: %v", err)
	}

	// Run migrations once for the entire test suite.
	migrator, err := migrate.New("file://../../migrations", dsn)
	if err != nil {
		log.Fatalf("could not create migrator: %v", err)
	}
	if err := migrator.Up(); err != nil && err != migrate.ErrNoChange {
		log.Fatalf("could not run migrations up: %v", err)
	}

	// Run the actual tests.
	code := m.Run()

	// Perform teardown after all tests have run.
	if err := migrator.Down(); err != nil {
		log.Fatalf("could not run migrations down: %v", err)
	}
	testDB.Close()

	os.Exit(code)
}

// newTestApp is a helper that gives us a clean application state for each test.
func newTestApp(t *testing.T) *handlers.PolicyHandler {
	// Truncate the policies table to ensure a clean slate before each test.
	_, err := testDB.Exec("TRUNCATE policies RESTART IDENTITY")
	if err != nil {
		t.Fatalf("could not truncate policies table: %v", err)
	}

	// Create dependencies using the single testDB connection pool.
	policyRepo := &repository.PostgresPolicyRepo{DB: testDB}
	enforcer, err := setEnforcer(testDB)
	if err != nil {
		t.Fatalf("failed to create enforcer for test: %v", err)
	}

	policyHandler := &handlers.PolicyHandler{Repo: policyRepo, Enforcer: enforcer}
	return policyHandler
}

func TestCreatePolicyEndpoint(t *testing.T) {
	// 1. Arrange
	policyHandler := newTestApp(t)
	router := chi.NewRouter()
	router.Post("/policies", policyHandler.CreatePolicyHandler)

	payload := `{"subject": "user:test", "action": "read", "object": "resource:123"}`
	req := httptest.NewRequest(http.MethodPost, "/policies", strings.NewReader(payload))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()

	// 2. Act
	router.ServeHTTP(rr, req)

	// 3. Assert
	if rr.Code != http.StatusCreated {
		t.Fatalf("expected status 201 Created, got %d, body=%s", rr.Code, rr.Body.String())
	}

	// Verify database state
	var count int
	err := testDB.Get(&count, `SELECT COUNT(*) FROM policies WHERE subject=$1 AND action=$2 AND object=$3`,
		"user:test", "read", "resource:123")
	if err != nil {
		t.Fatalf("failed to query policies table: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 policy inserted, got %d", count)
	}
}

func TestCheckEndpoint(t *testing.T) {
	// 1. Arrange
	policyHandler := newTestApp(t)
	router := chi.NewRouter()
	router.Post("/check", policyHandler.Check)

	// Seed the database with a policy for this test
	testPolicy := models.Policy{Subject: "user:test", Object: "resource:1", Action: "read"}
	_, err := policyHandler.Repo.CreatePolicy(testPolicy)
	if err != nil {
		t.Fatalf("failed to seed test policy: %v", err)
	}
    
    // --- 2. Act & 3. Assert: The "Allow" Case ---
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

    // --- 2. Act & 3. Assert: The "Deny" Case ---
	denyPayload := `{"subject": "user:scammer", "object": "resource:1", "action": "read"}`
	req = httptest.NewRequest(http.MethodPost, "/check", strings.NewReader(denyPayload))
	req.Header.Set("Content-Type", "application/json")
	rr = httptest.NewRecorder() // Use a fresh recorder
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
	if err := enforcer.LoadPolicy(); err != nil {
		return nil, fmt.Errorf("failed to load casbin policy: %w", err)
	}
	return enforcer, nil
}