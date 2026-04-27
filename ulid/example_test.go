package ulid_test

import (
	"fmt"
	"log"

	"github.com/teamlify-devx/idgen/ulid"
)

// ExampleNew demonstrates basic ULID generation.
func ExampleNew() {
	u, err := ulid.New()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("ULID: %s\n", u)
	fmt.Printf("Length: %d\n", len(u.String()))
	// Output example (non-deterministic, but always 26 chars):
	// ULID: 01ARYZ3NDEKTSV4RRFFQ69G5FA
	// Length: 26
}

// ExampleMust demonstrates panic-free initialization for package-level vars.
func ExampleMust() {
	u := ulid.Must()
	fmt.Printf("Length: %d\n", len(u.String()))
	// Output:
	// Length: 26
}

// ExampleULID_Time demonstrates timestamp extraction.
func ExampleULID_Time() {
	u, err := ulid.New()
	if err != nil {
		log.Fatal(err)
	}

	ts := u.Time()
	fmt.Printf("Has timestamp: %v\n", !ts.IsZero())
	// Output:
	// Has timestamp: true
}

// ExampleULID_Bytes demonstrates raw byte access for database storage.
func ExampleULID_Bytes() {
	u := ulid.Must()
	b := u.Bytes()
	fmt.Printf("Byte length: %d\n", len(b))
	// Output:
	// Byte length: 16
}

// ExampleGenerator demonstrates using a custom generator instance (e.g., per-tenant isolation).
func ExampleGenerator() {
	gen := &ulid.Generator{}

	ids := make([]string, 3)
	for i := range ids {
		u, err := gen.New()
		if err != nil {
			log.Fatal(err)
		}
		ids[i] = u.String()
	}

	// IDs generated in order must be lexicographically non-decreasing.
	fmt.Printf("Sorted: %v\n", ids[0] <= ids[1] && ids[1] <= ids[2])
	// Output:
	// Sorted: true
}

// ExampleParse demonstrates parsing and roundtrip validation.
func ExampleParse() {
	original := ulid.Must()
	s := original.String()

	parsed, err := ulid.Parse(s)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Roundtrip match: %v\n", parsed.String() == s)
	// Output:
	// Roundtrip match: true
}

// ExampleParse_error demonstrates error handling.
func ExampleParse_error() {
	_, err := ulid.Parse("not-valid!")
	if err != nil {
		fmt.Println("caught: parse error")
	}
	// Output:
	// caught: parse error
}
