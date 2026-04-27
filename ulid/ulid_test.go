package ulid

import (
	"sync"
	"testing"
	"time"
)

func TestNew_Length(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	s := u.String()
	if len(s) != 26 {
		t.Fatalf("expected 26 chars, got %d: %s", len(s), s)
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
			t.Fatalf("duplicate ULID: %s", s)
		}
		seen[s] = true
	}
}

func TestNew_Monotonic(t *testing.T) {
	prev, _ := New()
	prevStr := prev.String()
	for range 100 {
		curr, err := New()
		if err != nil {
			t.Fatalf("New() error: %v", err)
		}
		currStr := curr.String()
		if currStr < prevStr {
			t.Fatalf("not monotonic: prev=%s curr=%s", prevStr, currStr)
		}
		prevStr = currStr
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

	ts := u.Time()
	if ts.Before(before) || ts.After(after) {
		t.Errorf("extracted time %v not in [%v, %v]", ts, before, after)
	}
}

func TestParseRoundtrip(t *testing.T) {
	u, err := New()
	if err != nil {
		t.Fatalf("New() error: %v", err)
	}
	s := u.String()
	u2, err := Parse(s)
	if err != nil {
		t.Fatalf("Parse() error: %v", err)
	}
	if u2.String() != s {
		t.Fatalf("roundtrip mismatch: %s != %s", u2.String(), s)
	}
}

func TestParse_Invalid(t *testing.T) {
	cases := []string{
		"",
		"tooshort",
		"TOOLONGSTRINGTHATEXCEEDSTWENTYSIXCHARS",
		"01ARZ3NDEKTSV4RRFFQ69G5FAV!",
	}
	for _, c := range cases {
		_, err := Parse(c)
		if err == nil {
			t.Errorf("Parse(%q) expected error, got nil", c)
		}
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
