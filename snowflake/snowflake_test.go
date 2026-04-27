package snowflake

import (
	"os"
	"sync"
	"testing"
	"time"
)

func TestNewNode_DefaultValues(t *testing.T) {
	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	if node.nodeID != 0 {
		t.Errorf("Expected nodeID=0, got %d", node.nodeID)
	}

	if node.maxNodeID != 16383 {
		t.Errorf("Expected maxNodeID=16383, got %d", node.maxNodeID)
	}

	if node.maxSequence != 255 {
		t.Errorf("Expected maxSequence=255, got %d", node.maxSequence)
	}
}

func TestNewNode_InvalidBits(t *testing.T) {
	tests := []struct {
		name    string
		setup   func()
		cleanup func()
		wantErr bool
	}{
		{
			name: "nodeBits = 0",
			setup: func() {
				os.Setenv("SNOWFLAKE_NODE_BITS", "0")
			},
			cleanup: func() {
				os.Unsetenv("SNOWFLAKE_NODE_BITS")
			},
			wantErr: true,
		},
		{
			name: "nodeBits = 32",
			setup: func() {
				os.Setenv("SNOWFLAKE_NODE_BITS", "32")
			},
			cleanup: func() {
				os.Unsetenv("SNOWFLAKE_NODE_BITS")
			},
			wantErr: true,
		},
		{
			name: "nodeBits + sequenceBits >= 63",
			setup: func() {
				os.Setenv("SNOWFLAKE_NODE_BITS", "31")
				os.Setenv("SNOWFLAKE_SEQUENCE_BITS", "32")
			},
			cleanup: func() {
				os.Unsetenv("SNOWFLAKE_NODE_BITS")
				os.Unsetenv("SNOWFLAKE_SEQUENCE_BITS")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()
			defer tt.cleanup()

			_, err := NewNode()
			if (err != nil) != tt.wantErr {
				t.Errorf("NewNode() error = %v, wantErr %v", err, tt.wantErr)
			}
		})
	}
}

func TestGenerate_UniqueIDs(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	ids := make(map[int64]bool)
	count := 1000

	for i := 0; i < count; i++ {
		id, err := node.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}

		if ids[id] {
			t.Errorf("Duplicate ID generated: %d", id)
		}
		ids[id] = true
	}

	if len(ids) != count {
		t.Errorf("Expected %d unique IDs, got %d", count, len(ids))
	}
}

func TestGenerate_ThreadSafe(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	ids := make(chan int64, 1000)
	var wg sync.WaitGroup
	goroutines := 10
	idsPerGoroutine := 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func() {
			defer wg.Done()
			for j := 0; j < idsPerGoroutine; j++ {
				id, err := node.Generate()
				if err != nil {
					t.Errorf("Generate() failed: %v", err)
				}
				ids <- id
			}
		}()
	}

	wg.Wait()
	close(ids)

	seen := make(map[int64]bool)
	count := 0
	for id := range ids {
		if seen[id] {
			t.Errorf("Duplicate ID detected in concurrent generation: %d", id)
		}
		seen[id] = true
		count++
	}

	if count != goroutines*idsPerGoroutine {
		t.Errorf("Expected %d IDs, got %d", goroutines*idsPerGoroutine, count)
	}
}

func TestGenerate_Monotonic(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	var prevID int64
	for i := 0; i < 100; i++ {
		id, err := node.Generate()
		if err != nil {
			t.Fatalf("Generate() failed: %v", err)
		}

		if i > 0 && id <= prevID {
			t.Errorf("ID not monotonic: prevID=%d, currentID=%d", prevID, id)
		}
		prevID = id
	}
}

func TestNodeID_Extraction(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	extractedNodeID := node.NodeID(id)
	if extractedNodeID != node.nodeID {
		t.Errorf("Expected nodeID=%d, got %d", node.nodeID, extractedNodeID)
	}
}

func TestSequence_Extraction(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}

	seq := node.Sequence(id)
	if seq < 0 || seq > node.maxSequence {
		t.Errorf("Sequence out of range: %d (max=%d)", seq, node.maxSequence)
	}
}

func TestTime_Extraction(t *testing.T) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		t.Fatalf("NewNode() failed: %v", err)
	}

	beforeGen := time.Now()
	id, err := node.Generate()
	if err != nil {
		t.Fatalf("Generate() failed: %v", err)
	}
	afterGen := time.Now()

	extractedTime := node.Time(id)

	if extractedTime.Before(beforeGen.Add(-10*time.Millisecond)) || extractedTime.After(afterGen.Add(10*time.Millisecond)) {
		t.Errorf("Extracted time not within expected range. Got %v, expected between %v and %v", extractedTime, beforeGen, afterGen)
	}
}

func TestGetNodeIDFromPod_Valid(t *testing.T) {
	tests := []struct {
		podName    string
		expected   int64
		maxNodeID  int64
		shouldFail bool
	}{
		{"myapp-0", 0, 16383, false},
		{"myapp-1", 1, 16383, false},
		{"myapp-9999", 9999, 16383, false},
		{"app-pod-0", 0, 16383, false},
		{"app-pod-100", 100, 16383, false},
	}

	for _, tt := range tests {
		t.Run(tt.podName, func(t *testing.T) {
			os.Setenv("POD_NAME", tt.podName)
			defer os.Unsetenv("POD_NAME")

			nodeID, err := getNodeIDFromPod(tt.maxNodeID)
			if (err != nil) != tt.shouldFail {
				t.Errorf("getNodeIDFromPod(%s) error = %v, shouldFail %v", tt.podName, err, tt.shouldFail)
			}
			if err == nil && nodeID != tt.expected {
				t.Errorf("Expected nodeID=%d, got %d", tt.expected, nodeID)
			}
		})
	}
}

func TestGetNodeIDFromPod_Invalid(t *testing.T) {
	tests := []struct {
		name       string
		podName    string
		maxNodeID  int64
		shouldFail bool
	}{
		{"no_ordinal", "myapp", 16383, true},
		{"exceeds_max", "myapp-20000", 16383, true},
		{"not_numeric", "myapp-abc", 16383, true},
		{"empty", "", 16383, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			os.Setenv("POD_NAME", tt.podName)
			defer os.Unsetenv("POD_NAME")

			_, err := getNodeIDFromPod(tt.maxNodeID)
			if (err != nil) != tt.shouldFail {
				t.Errorf("Expected error, got %v", err)
			}
		})
	}
}

func BenchmarkGenerate(b *testing.B) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		b.Fatalf("NewNode() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, err := node.Generate()
		if err != nil {
			b.Fatalf("Generate() failed: %v", err)
		}
	}
}

func BenchmarkGenerateConcurrent(b *testing.B) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		b.Fatalf("NewNode() failed: %v", err)
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := node.Generate()
			if err != nil {
				b.Fatalf("Generate() failed: %v", err)
			}
		}
	})
}

func BenchmarkExtractTime(b *testing.B) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		b.Fatalf("NewNode() failed: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		b.Fatalf("Generate() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Time(id)
	}
}

func BenchmarkExtractNodeID(b *testing.B) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		b.Fatalf("NewNode() failed: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		b.Fatalf("Generate() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.NodeID(id)
	}
}

func BenchmarkExtractSequence(b *testing.B) {
	os.Setenv("SNOWFLAKE_NODE_TYPE", "single")
	defer os.Unsetenv("SNOWFLAKE_NODE_TYPE")

	node, err := NewNode()
	if err != nil {
		b.Fatalf("NewNode() failed: %v", err)
	}

	id, err := node.Generate()
	if err != nil {
		b.Fatalf("Generate() failed: %v", err)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = node.Sequence(id)
	}
}
