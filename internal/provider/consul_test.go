package provider

import (
	"context"
	"testing"

	consul "github.com/hashicorp/consul/api"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

// Integration tests against a real Consul instance.
// Requires Consul running at localhost:8500 with the following KV data:
//
//	consul kv put lumos/test/database-url "postgres://localhost:5432/db"
//	consul kv put lumos/test/redis-url    "redis://localhost:6379"
//	consul kv put lumos/test/config       '{"DB_HOST":"localhost","DB_PORT":"5432"}'
const (
	consulAddr   = "localhost:8500"
	consulPrefix = "lumos/test"
)

// skipIfConsulNotAvailable skips the test if Consul isn't reachable.
func skipIfConsulNotAvailable(t *testing.T) {
	t.Helper()
	cfg := consul.DefaultConfig()
	cfg.Address = consulAddr
	c, err := consul.NewClient(cfg)
	if err != nil {
		t.Skipf("skipping: consul client error: %v", err)
	}
	if _, err := c.Agent().Self(); err != nil {
		t.Skipf("skipping: consul not available at %s: %v", consulAddr, err)
	}
}

// ── Raw ──────────────────────────────────────────────────────────────────────

func TestConsulProvider_Fetch_Raw(t *testing.T) {
	skipIfConsulNotAvailable(t)
	p := NewConsul(consulAddr, consulPrefix, "")

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "database-url", Key: "DATABASE_URL", Format: syncv1alpha1.FormatRaw},
		{Source: "redis-url", Key: "REDIS_URL", Format: syncv1alpha1.FormatRaw},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/db",
		"REDIS_URL":    "redis://localhost:6379",
	})
	if result.Version == "" {
		t.Error("expected non-empty Version (ModifyIndex)")
	}
}

// ── Env ──────────────────────────────────────────────────────────────────────

func TestConsulProvider_Fetch_Env(t *testing.T) {
	skipIfConsulNotAvailable(t)
	p := NewConsul(consulAddr, consulPrefix, "")

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "config", Format: syncv1alpha1.FormatEnv},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}

	assertData(t, result.Data, map[string]string{
		"DB_HOST": "localhost",
		"DB_PORT": "5432",
	})
}

// ── Error cases ──────────────────────────────────────────────────────────────

func TestConsulProvider_Fetch_MissingKey(t *testing.T) {
	skipIfConsulNotAvailable(t)
	p := NewConsul(consulAddr, consulPrefix, "")

	_, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "does-not-exist", Key: "NOPE", Format: syncv1alpha1.FormatRaw},
	})
	if err == nil {
		t.Fatal("expected error for missing key, got nil")
	}
}

func TestConsulProvider_Fetch_WrongAddress(t *testing.T) {
	// Does not need a running Consul — expects connection failure.
	p := NewConsul("localhost:19999", consulPrefix, "")

	_, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "database-url", Key: "DATABASE_URL", Format: syncv1alpha1.FormatRaw},
	})
	if err == nil {
		t.Fatal("expected error for unreachable Consul, got nil")
	}
}

func TestConsulProvider_Fetch_WithToken(t *testing.T) {
	skipIfConsulNotAvailable(t)
	// Consul in dev mode ignores ACL tokens; passing one exercises the token code path.
	p := NewConsul(consulAddr, consulPrefix, "fake-dev-token")

	result, err := p.Fetch(context.Background(), []syncv1alpha1.ExternalConfigData{
		{Source: "database-url", Key: "DATABASE_URL", Format: syncv1alpha1.FormatRaw},
	})
	if err != nil {
		t.Fatalf("Fetch failed: %v", err)
	}
	assertData(t, result.Data, map[string]string{
		"DATABASE_URL": "postgres://localhost:5432/db",
	})
}
