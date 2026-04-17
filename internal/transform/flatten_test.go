package transform

import (
	"testing"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

func TestApply(t *testing.T) {
	tests := []struct {
		name     string
		format   syncv1alpha1.DataFormat
		key      string
		rawValue string
		want     map[string]string
		wantErr  bool
	}{
		// ── Raw ──────────────────────────────────────────────────────────────
		{
			name:     "raw: stores value under key",
			format:   syncv1alpha1.FormatRaw,
			key:      "app.yaml",
			rawValue: "hello: world\n",
			want:     map[string]string{"app.yaml": "hello: world\n"},
		},

		// ── Env (JSON) ───────────────────────────────────────────────────────
		{
			name:     "env(json): nested map flattens to UPPER_SNAKE_CASE",
			format:   syncv1alpha1.FormatEnv,
			key:      "",
			rawValue: `{"database": {"maxConnections": 20, "host": "localhost"}}`,
			want: map[string]string{
				"DATABASE_MAX_CONNECTIONS": "20",
				"DATABASE_HOST":            "localhost",
			},
		},

		// ── Env (.env) ───────────────────────────────────────────────────────
		{
			name:   "env(.env): basic key=value",
			format: syncv1alpha1.FormatEnv,
			key:    "",
			rawValue: `
DB_HOST=localhost
DB_PORT=5432
`,
			want: map[string]string{
				"DB_HOST": "localhost",
				"DB_PORT": "5432",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := Apply(tt.format, tt.key, tt.rawValue)

			if tt.wantErr {
				if err == nil {
					t.Fatal("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if len(got) != len(tt.want) {
				t.Fatalf("len mismatch: got %d keys %v, want %d keys %v", len(got), got, len(tt.want), tt.want)
			}
			for k, wantVal := range tt.want {
				if got[k] != wantVal {
					t.Errorf("key %q: got %q, want %q", k, got[k], wantVal)
				}
			}
		})
	}
}
