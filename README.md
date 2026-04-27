# idgen
#### Distributed ID Generators for Go

--- 

A collection of zero-dependency, thread-safe ID generators for distributed systems.

| Package | Format | Sortable | Size |
|---------|--------|----------|------|
| `snowflake` | 64-bit integer | by time | 8 bytes |
| `uuidv4` | RFC 4122 UUID | no | 16 bytes |
| `uuidv7` | RFC 9562 UUID | by time | 16 bytes |
| `ulid` | Crockford base32 | by time | 16 bytes |

## Installation

```bash
go get github.com/teamlify-devx/idgen
```

Import only the sub-package(s) you need:

```go
import "github.com/teamlify-devx/idgen/snowflake"
import "github.com/teamlify-devx/idgen/uuidv4"
import "github.com/teamlify-devx/idgen/uuidv7"
import "github.com/teamlify-devx/idgen/ulid"
```

---

## Snowflake

Twitter-inspired 64-bit integer IDs. Time-ordered, node-aware, and extremely fast (~256K IDs/s per node by default).

### Bit layout

```
[63]      sign bit        → always 0 (positive)
[62..22]  41-bit epoch    → timestamp in ms (~69 years)
[21..8]   14-bit node     → machine/pod ID (0–16383)
[7..0]    8-bit sequence  → counter per ms (0–255)
```

### Quick start

```go
import "github.com/teamlify-devx/idgen/snowflake"

node, err := snowflake.NewNode()
if err != nil {
    log.Fatal(err)
}

id, err := node.Generate()
if err != nil {
    log.Fatal(err)
}

fmt.Println(id) // e.g. 7294523109237760
```

### Configuration

`NewNode` reads configuration from environment variables (prefix: `SNOWFLAKE_`).

| Env var | Default | Description |
|---------|---------|-------------|
| `SNOWFLAKE_EPOCH` | `1700000000000` | Custom epoch in milliseconds |
| `SNOWFLAKE_NODE_BITS` | `14` | Bits for node ID (1–31) |
| `SNOWFLAKE_SEQUENCE_BITS` | `8` | Bits for sequence (1–31) |
| `SNOWFLAKE_NODE_TYPE` | `single` | Node ID strategy: `single`, `manual`, `k8s` |
| `SNOWFLAKE_NODE_ID` | `0` | Node ID when `NODE_TYPE=manual` |

**single** (default) — always uses node ID 0, suitable for single-instance deployments.

**manual** — you provide the node ID explicitly:

```bash
export SNOWFLAKE_NODE_TYPE=manual
export SNOWFLAKE_NODE_ID=5
```

**k8s** — extracts node ID from the StatefulSet pod ordinal in `POD_NAME`:

```bash
export SNOWFLAKE_NODE_TYPE=k8s
export POD_NAME=myapp-3   # → node ID = 3
```

### Decoding an ID

```go
ts     := node.Time(id)     // time.Time when the ID was generated
nodeID := node.NodeID(id)   // which node generated it
seq    := node.Sequence(id) // sequence number within that millisecond
```

### Global singleton pattern

```go
var gen *snowflake.Node

func init() {
    var err error
    gen, err = snowflake.NewNode()
    if err != nil {
        log.Fatal(err)
    }
}

func newID() int64 {
    id, err := gen.Generate()
    if err != nil {
        log.Fatal(err)
    }
    return id
}
```

---

## UUID v4

Randomly generated UUID per RFC 4122. No dependencies beyond `crypto/rand`.

```go
import "github.com/teamlify-devx/idgen/uuidv4"

// Returns (UUID, error)
uuid, err := uuidv4.New()
if err != nil {
    log.Fatal(err)
}
fmt.Println(uuid) // e.g. 550e8400-e29b-41d4-a716-446655440000

// Panics on error — safe for package-level vars
uuid = uuidv4.Must()

// Raw bytes
b := uuid.Bytes() // []byte, len 16
```

### Parsing

```go
uuid, err := uuidv4.Parse("550e8400-e29b-41d4-a716-446655440000")
```

`Parse` validates the canonical `xxxxxxxx-xxxx-4xxx-xxxx-xxxxxxxxxxxx` format and checks the version nibble.

### Error handling

```go
import "errors"

uuid, err := uuidv4.Parse(someInput)
if errors.Is(err, uuidv4.ErrInvalidUUID) {
    // bad format, wrong length, invalid hex, or version != 4
}
if errors.Is(err, uuidv4.ErrRandomSource) {
    // crypto/rand failure during New()
}
```

---

## UUID v7

Time-ordered UUID per RFC 9562. Monotonic within the same millisecond (12-bit sequence counter).

```go
import "github.com/teamlify-devx/idgen/uuidv7"

// Package-level generator (thread-safe)
uuid, err := uuidv7.New()
if err != nil {
    log.Fatal(err)
}
fmt.Println(uuid) // e.g. 018e3b2a-1234-7abc-8def-000000000001

// Panics on error
uuid = uuidv7.Must()

// Extract the embedded timestamp
ts, err := uuid.Time() // time.Time

// Raw bytes
b := uuid.Bytes() // []byte, len 16
```

Custom generator instance (useful for testing or multiple independent streams):

```go
g := &uuidv7.Generator{}
uuid, err := g.New()
```

### Parsing

```go
uuid, err := uuidv7.Parse("018e3b2a-1234-7abc-8def-000000000001")
```

`Parse` validates format and verifies the version nibble is 7.

### Error handling

```go
import "errors"

uuid, err := uuidv7.Parse(someInput)
if errors.Is(err, uuidv7.ErrInvalidUUID) {
    // bad format, wrong length, or invalid hex
}
if errors.Is(err, uuidv7.ErrWrongVersion) {
    // parsed successfully but version nibble != 7
}
if errors.Is(err, uuidv7.ErrRandomSource) {
    // crypto/rand failure during New()
}
```

---

## ULID

Universally Unique Lexicographically Sortable Identifier. 48-bit timestamp + 80-bit random, encoded as a 26-character Crockford base32 string.

```go
import "github.com/teamlify-devx/idgen/ulid"

// Package-level generator (thread-safe)
id, err := ulid.New()
if err != nil {
    log.Fatal(err)
}
fmt.Println(id) // e.g. 01ARZ3NDEKTSV4RRFFQ69G5FAV

// Panics on error
id = ulid.Must()

// Extract timestamp
ts := id.Time() // time.Time

// Parse from string
id, err = ulid.Parse("01ARZ3NDEKTSV4RRFFQ69G5FAV")

// Raw bytes
b := id.Bytes() // []byte, len 16
```

Custom generator instance:

```go
g := &ulid.Generator{}
id, err := g.New()
```

---

## Choosing the right generator

- **Need an integer ID with node awareness?** → `snowflake`
- **Need a random ID and don't care about ordering?** → `uuidv4`
- **Need a UUID that sorts by creation time?** → `uuidv7`
- **Need a string ID that sorts lexicographically?** → `ulid`

---

## Testing

```bash
make test           # run all tests
make test-race      # run with race detector
make test-coverage  # generate coverage report
make bench          # run benchmarks
```
