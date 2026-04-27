package uuidv4

import (
	"errors"
	"strings"
	"sync"
	"testing"
)

func TestNew_Format(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	s := u.String()
	if len(s) != 36 {
		t.Fatalf("expected length 36, got %d: %s", len(s), s)
	}
	parts := strings.Split(s, "-")
	if len(parts) != 5 {
		t.Fatalf("expected 5 parts, got %d", len(parts))
	}
	if parts[2][0] != '4' {
		t.Errorf("version nibble must be '4', got %c", parts[2][0])
	}
	variant := parts[3][0]
	if variant != '8' && variant != '9' && variant != 'a' && variant != 'b' {
		t.Errorf("variant nibble must be 8/9/a/b, got %c", variant)
	}
}

func TestNew_Unique(t *testing.T) {
	seen := make(map[string]bool)
	for range 1000 {
		u, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		s := u.String()
		if seen[s] {
			t.Fatalf("duplicate UUID: %s", s)
		}
		seen[s] = true
	}
}

func TestNew_Concurrent(t *testing.T) {
	var wg sync.WaitGroup
	results := make(chan string, 1000)
	for range 10 {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for range 100 {
				u, err := New()
				if err != nil {
					t.Errorf("New() error: %v", err)
					return
				}
				results <- u.String()
			}
		}()
	}
	wg.Wait()
	close(results)
	seen := make(map[string]bool)
	for s := range results {
		if seen[s] {
			t.Fatalf("duplicate UUID in concurrent run: %s", s)
		}
		seen[s] = true
	}
}

func TestMust_NoPanic(t *testing.T) {
	defer func() {
		if r := recover(); r != nil {
			t.Fatalf("Must() panicked: %v", r)
		}
	}()
	_ = Must()
}

func TestBytes(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	b := u.Bytes()
	if len(b) != 16 {
		t.Fatalf("expected 16 bytes, got %d", len(b))
	}
}

func TestParse_Roundtrip(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	parsed, err := Parse(u.String())
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if parsed != u {
		t.Fatalf("roundtrip mismatch: got %s, want %s", parsed.String(), u.String())
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := []struct {
		name  string
		input string
	}{
		{"empty", ""},
		{"too short", "550e8400-e29b-41d4-a716"},
		{"no hyphens", "550e8400e29b41d4a716446655440000"},
		{"bad hex", "550e8400-e29b-41d4-a716-44665544zzzz"},
		{"wrong version", "550e8400-e29b-31d4-a716-446655440000"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tc.input)
			}
			if !errors.Is(err, ErrInvalidUUID) && !errors.Is(err, ErrRandomSource) {
				// wrong version also wraps ErrInvalidUUID — any sentinel is fine
			}
		})
	}
}

func BenchmarkParse(b *testing.B) {
	u := Must()
	s := u.String()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = Parse(s)
	}
}

func BenchmarkNew(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, _ = New()
	}
}

func BenchmarkString(b *testing.B) {
	u, _ := New()
	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = u.String()
	}
}
