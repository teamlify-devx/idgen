package uuidv7

import (
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
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
	if parts[2][0] != '7' {
		t.Errorf("version nibble must be '7', got %c", parts[2][0])
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

func TestNew_Monotonic(t *testing.T) {
	prev, _ := New()
	for range 100 {
		curr, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		for j := range 16 {
			if curr[j] > prev[j] {
				break
			}
			if curr[j] < prev[j] {
				t.Fatalf("not monotonic: prev=%s curr=%s", prev.String(), curr.String())
			}
		}
		prev = curr
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
			t.Fatalf("duplicate in concurrent run: %s", s)
		}
		seen[s] = true
	}
}

func TestTime_Extraction(t *testing.T) {
	before := time.Now().Truncate(time.Millisecond)
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	after := time.Now().Add(time.Millisecond)

	ts, err := u.Time()
	if err != nil {
		t.Fatalf("Time() error: %v", err)
	}
	if ts.Before(before) || ts.After(after) {
		t.Errorf("extracted time %v not in [%v, %v]", ts, before, after)
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
		name    string
		input   string
		wantErr error
	}{
		{"empty", "", ErrInvalidUUID},
		{"too short", "01930d14-d40e-7f4e-8b9e", ErrInvalidUUID},
		{"no hyphens", "01930d14d40e7f4e8b9e0000000000000", ErrInvalidUUID},
		{"bad hex", "01930d14-d40e-7f4e-8b9e-zzzzzzzzzzzz", ErrInvalidUUID},
		{"wrong version (v4)", "550e8400-e29b-41d4-a716-446655440000", ErrWrongVersion},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			_, err := Parse(tc.input)
			if err == nil {
				t.Fatalf("Parse(%q) expected error, got nil", tc.input)
			}
			if !errors.Is(err, tc.wantErr) {
				t.Errorf("expected %v, got %v", tc.wantErr, err)
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

func BenchmarkNewConcurrent(b *testing.B) {
	b.ReportAllocs()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = New()
		}
	})
}
