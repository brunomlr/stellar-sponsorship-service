//go:build integration

package store

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/stellar/go-stellar-sdk/keypair"

	"github.com/stellar-sponsorship-service/internal/model"
)

func TestPostgresStoreAPIKeyLifecycleIntegration(t *testing.T) {
	ctx := context.Background()
	pg := setupIntegrationStore(t)

	sponsor := randomAddress(t)
	apiKey := &model.APIKey{
		Name:                  "integration-key",
		KeyHash:               fmt.Sprintf("hash-%s", uuid.NewString()),
		KeyPrefix:             "sk_test_abc...",
		SponsorAccount:        sponsor,
		XLMBudget:             50_000_000,
		AllowedOperations:     []string{"MANAGE_DATA", "SET_OPTIONS"},
		AllowedSourceAccounts: []string{randomAddress(t)},
		RateLimitMax:          120,
		RateLimitWindow:       300,
		Status:                model.StatusPendingFunding,
		FundingTxXDR:          "AAAA-test-xdr",
		ExpiresAt:             time.Now().UTC().Add(24 * time.Hour),
	}

	if err := pg.CreateAPIKey(ctx, apiKey); err != nil {
		t.Fatalf("create api key: %v", err)
	}
	if apiKey.ID == uuid.Nil {
		t.Fatal("expected generated API key ID")
	}

	byHash, err := pg.GetAPIKeyByHash(ctx, apiKey.KeyHash)
	if err != nil {
		t.Fatalf("get by hash: %v", err)
	}
	if byHash.ID != apiKey.ID {
		t.Fatalf("unexpected id from hash lookup: got %s want %s", byHash.ID, apiKey.ID)
	}

	byID, err := pg.GetAPIKeyByID(ctx, apiKey.ID)
	if err != nil {
		t.Fatalf("get by id: %v", err)
	}
	if byID.Name != apiKey.Name {
		t.Fatalf("unexpected name from id lookup: got %q want %q", byID.Name, apiKey.Name)
	}

	newName := "integration-key-updated"
	newRateLimitMax := 999
	newRateLimitWindow := 600
	if err := pg.UpdateAPIKey(ctx, apiKey.ID, APIKeyUpdates{
		Name:            &newName,
		RateLimitMax:    &newRateLimitMax,
		RateLimitWindow: &newRateLimitWindow,
	}); err != nil {
		t.Fatalf("update api key: %v", err)
	}

	updated, err := pg.GetAPIKeyByID(ctx, apiKey.ID)
	if err != nil {
		t.Fatalf("get updated key: %v", err)
	}
	if updated.Name != newName {
		t.Fatalf("unexpected updated name: got %q want %q", updated.Name, newName)
	}
	if updated.RateLimitMax != newRateLimitMax || updated.RateLimitWindow != newRateLimitWindow {
		t.Fatalf("unexpected updated rate limit: max=%d window=%d", updated.RateLimitMax, updated.RateLimitWindow)
	}

	if err := pg.UpdateAPIKeyStatus(ctx, apiKey.ID, model.StatusRevoked); err != nil {
		t.Fatalf("update status: %v", err)
	}

	revoked, err := pg.GetAPIKeyByID(ctx, apiKey.ID)
	if err != nil {
		t.Fatalf("get revoked key: %v", err)
	}
	if revoked.Status != model.StatusRevoked {
		t.Fatalf("unexpected status: got %q want %q", revoked.Status, model.StatusRevoked)
	}

	keys, total, err := pg.ListAPIKeys(ctx, 1, 20)
	if err != nil {
		t.Fatalf("list api keys: %v", err)
	}
	if total != 1 {
		t.Fatalf("unexpected total: got %d want 1", total)
	}
	if len(keys) != 1 || keys[0].ID != apiKey.ID {
		t.Fatalf("unexpected listed keys: %#v", keys)
	}
}

func TestPostgresStoreTransactionQueriesIntegration(t *testing.T) {
	ctx := context.Background()
	pg := setupIntegrationStore(t)

	apiKey := &model.APIKey{
		Name:              "tx-key",
		KeyHash:           fmt.Sprintf("hash-%s", uuid.NewString()),
		KeyPrefix:         "sk_test_xyz...",
		SponsorAccount:    randomAddress(t),
		XLMBudget:         10_000_000,
		AllowedOperations: []string{"MANAGE_DATA"},
		RateLimitMax:      50,
		RateLimitWindow:   60,
		Status:            model.StatusActive,
		ExpiresAt:         time.Now().UTC().Add(24 * time.Hour),
	}
	if err := pg.CreateAPIKey(ctx, apiKey); err != nil {
		t.Fatalf("create api key: %v", err)
	}

	signed := &model.TransactionLog{
		APIKeyID:        apiKey.ID,
		TransactionHash: "deadbeef",
		TransactionXDR:  "AAAA-signed",
		Operations:      []string{"MANAGE_DATA"},
		SourceAccount:   randomAddress(t),
		Status:          model.TxStatusSigned,
	}
	if err := pg.CreateTransactionLog(ctx, signed); err != nil {
		t.Fatalf("create signed tx log: %v", err)
	}

	rejected := &model.TransactionLog{
		APIKeyID:        apiKey.ID,
		TransactionXDR:  "AAAA-rejected",
		Operations:      []string{"SET_OPTIONS"},
		SourceAccount:   randomAddress(t),
		Status:          model.TxStatusRejected,
		RejectionReason: "not allowed",
	}
	if err := pg.CreateTransactionLog(ctx, rejected); err != nil {
		t.Fatalf("create rejected tx log: %v", err)
	}

	count, err := pg.CountTransactionsByAPIKey(ctx, apiKey.ID)
	if err != nil {
		t.Fatalf("count signed transactions: %v", err)
	}
	if count != 1 {
		t.Fatalf("unexpected signed count: got %d want 1", count)
	}

	status := model.TxStatusSigned
	from := time.Now().UTC().Add(-time.Hour)
	to := time.Now().UTC().Add(time.Hour)
	logs, total, err := pg.ListTransactionLogs(ctx, TransactionFilters{
		APIKeyID: &apiKey.ID,
		Status:   &status,
		From:     &from,
		To:       &to,
		Page:     1,
		PerPage:  20,
	})
	if err != nil {
		t.Fatalf("list filtered logs: %v", err)
	}
	if total != 1 || len(logs) != 1 {
		t.Fatalf("unexpected filtered logs result: total=%d len=%d", total, len(logs))
	}
	if logs[0].Status != model.TxStatusSigned {
		t.Fatalf("unexpected log status: got %q", logs[0].Status)
	}
}

func setupIntegrationStore(t *testing.T) *Postgres {
	t.Helper()

	databaseURL := os.Getenv("DATABASE_URL")
	if databaseURL == "" {
		t.Skip("DATABASE_URL not set; skipping integration test")
	}

	migrationsDir := repoMigrationsDir(t)
	m, err := migrate.New("file://"+migrationsDir, databaseURL)
	if err != nil {
		t.Fatalf("init migrate: %v", err)
	}
	if err := m.Up(); err != nil && err != migrate.ErrNoChange {
		t.Fatalf("apply migrations: %v", err)
	}
	if srcErr, dbErr := m.Close(); srcErr != nil || dbErr != nil {
		t.Fatalf("close migrator: source=%v database=%v", srcErr, dbErr)
	}

	pool, err := pgxpool.New(context.Background(), databaseURL)
	if err != nil {
		t.Fatalf("connect pg: %v", err)
	}
	t.Cleanup(pool.Close)

	if err := pool.Ping(context.Background()); err != nil {
		t.Fatalf("ping pg: %v", err)
	}

	if _, err := pool.Exec(context.Background(), `TRUNCATE TABLE transaction_logs, api_keys RESTART IDENTITY CASCADE`); err != nil {
		t.Fatalf("truncate tables: %v", err)
	}

	return NewPostgres(pool)
}

func repoMigrationsDir(t *testing.T) string {
	t.Helper()

	_, filename, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("failed to resolve test file path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(filename), "..", ".."))
	return filepath.Join(root, "migrations")
}

func randomAddress(t *testing.T) string {
	t.Helper()
	kp, err := keypair.Random()
	if err != nil {
		t.Fatalf("random keypair: %v", err)
	}
	return kp.Address()
}
