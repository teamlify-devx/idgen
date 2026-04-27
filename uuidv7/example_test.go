package uuidv7_test

import (
	"errors"
	"fmt"
	"log"

	"github.com/teamlify-devx/idgen/uuidv7"
)

// ExampleNew demonstrates basic UUID v7 generation.
func ExampleNew() {
	uuid, err := uuidv7.New()
	if err != nil {
		log.Fatal(err)
	}
	fmt.Printf("UUID v7: %s\n", uuid)
	fmt.Printf("Length: %d\n", len(uuid.String()))
	// Output example (non-deterministic, but always 36 chars):
	// UUID v7: xxxxxxxx-xxxx-7xxx-xxxx-xxxxxxxxxxxx
	// Length: 36
}

// ExampleMust demonstrates panic-free initialization for package-level vars.
func ExampleMust() {
	uuid := uuidv7.Must()
	fmt.Printf("Version nibble: %c\n", uuid.String()[14])
	// Output:
	// Version nibble: 7
}

// ExampleUUID_Time demonstrates timestamp extraction from a UUID v7.
func ExampleUUID_Time() {
	uuid, err := uuidv7.New()
	if err != nil {
		log.Fatal(err)
	}

	ts, err := uuid.Time()
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Has timestamp: %v\n", !ts.IsZero())
	// Output:
	// Has timestamp: true
}

// ExampleUUID_Bytes demonstrates raw byte access for database storage.
func ExampleUUID_Bytes() {
	uuid := uuidv7.Must()
	b := uuid.Bytes()
	fmt.Printf("Byte length: %d\n", len(b))
	// Output:
	// Byte length: 16
}

// ExampleGenerator demonstrates using a custom generator instance.
func ExampleGenerator() {
	gen := &uuidv7.Generator{}

	ids := make([]string, 3)
	for i := range ids {
		uuid, err := gen.New()
		if err != nil {
			log.Fatal(err)
		}
		ids[i] = uuid.String()
	}

	fmt.Printf("All unique: %v\n", ids[0] != ids[1] && ids[1] != ids[2])
	// Output:
	// All unique: true
}

// ExampleParse demonstrates parsing and validating a UUID v7 string.
func ExampleParse() {
	original := uuidv7.Must()
	s := original.String()

	parsed, err := uuidv7.Parse(s)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Roundtrip match: %v\n", parsed.String() == s)
	// Output:
	// Roundtrip match: true
}

// ExampleParse_wrongVersion demonstrates version mismatch error handling.
func ExampleParse_wrongVersion() {
	// A UUID v4 string fed into the v7 parser.
	uuidV4 := "550e8400-e29b-41d4-a716-446655440000"
	_, err := uuidv7.Parse(uuidV4)
	if errors.Is(err, uuidv7.ErrWrongVersion) {
		fmt.Println("caught: wrong UUID version")
	}
	// Output:
	// caught: wrong UUID version
}

// ExampleParse_invalidFormat demonstrates format error handling.
func ExampleParse_invalidFormat() {
	_, err := uuidv7.Parse("not-a-valid-uuid")
	if errors.Is(err, uuidv7.ErrInvalidUUID) {
		fmt.Println("caught: invalid UUID format")
	}
	// Output:
	// caught: invalid UUID format
}
