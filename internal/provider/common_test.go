package provider

import "testing"

func assertData(t *testing.T, got, want map[string]string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("len mismatch: got %d %v, want %d %v", len(got), got, len(want), want)
	}
	for k, wantVal := range want {
		gotVal, ok := got[k]
		if !ok {
			t.Errorf("missing key %q", k)
			continue
		}
		if gotVal != wantVal {
			t.Errorf("key %q: got %q, want %q", k, gotVal, wantVal)
		}
	}
}
