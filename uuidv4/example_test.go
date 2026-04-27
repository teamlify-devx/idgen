package uuidv4_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/teamlify-devx/idgen/uuidv4"
)

// ExampleNew demonstrates basic UUID v4 generation.
func ExampleNew() {
	uuid, err := uuidv4.New()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("UUID v4: %s\n", uuid)
	fmt.Printf("Length: %d\n", len(uuid.String()))
	// Output example (non-deterministic, but always 36 chars):
	// UUID v4: xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx
	// Length: 36
}

// ExampleMust demonstrates panic-free initialization for package-level vars.
func ExampleMust() {
	// Suitable for top-level var declarations where error handling is impractical.
	uuid := uuidv4.Must()
	fmt.Printf("Version nibble: %c\n", uuid.String()[14])
	// Output:
	// Version nibble: 4
}

// ExampleUUID_String demonstrates the canonical string format.
func ExampleUUID_String() {
	uuid := uuidv4.Must()
	s := uuid.String()
	fmt.Printf("Format: xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx\n")
	fmt.Printf("Length: %d\n", len(s))
	// Output:
	// Format: xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx
	// Length: 36
}

// ExampleUUID_Bytes demonstrates raw byte access for database storage.
func ExampleUUID_Bytes() {
	uuid := uuidv4.Must()
	b := uuid.Bytes()
	fmt.Printf("Byte length: %d\n", len(b))
	// Output:
	// Byte length: 16
}

// ExampleParse demonstrates parsing and validating a UUID v4 string.
func ExampleParse() {
	original := uuidv4.Must()
	s := original.String()

	parsed, err := uuidv4.Parse(s)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Roundtrip match: %v\n", parsed.String() == s)
	// Output:
	// Roundtrip match: true
}

// ExampleParse_error demonstrates error handling with sentinel errors.
func ExampleParse_error() {
	_, err := uuidv4.Parse("not-a-valid-uuid")
	if errors.Is(err, uuidv4.ErrInvalidUUID) {
		fmt.Println("caught: invalid UUID format")
	}
	// Output:
	// caught: invalid UUID format
}
