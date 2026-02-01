# lastmodstore

**Thread-safe TTL cache with timestamp tracking - exploring Go concurrency patterns for distributed system state management**

## Purpose

Part of my ongoing platform engineering skill maintenance. This repo explores implementing a concurrent key-value store with last-modified timestamp tracking - a foundational pattern that appears in distributed caches, metadata stores, and state coordination systems.

Built incrementally using TDD (Ginkgo/Gomega) to practice Go's sync primitives, time-based eviction strategies, and thread-safe data structure design.

## What This Implements

A concurrent in-memory store demonstrating:

**Core patterns**:
- Thread-safe read/write operations using sync.RWMutex
- TTL-based automatic expiration and cleanup
- Last-modified timestamp tracking for cache invalidation
- Atomic operations for concurrent access
- Background goroutine for periodic cleanup

**Use case**: Metadata tracking for distributed systems - tracking when resources were last updated, coordinating cache invalidation, or maintaining session state with expiration.

## Architecture

```go
type Entry struct {
    Value        interface{}
    LastModified time.Time
    TTL          time.Duration
}

type LastModStore struct {
    mu           sync.RWMutex
    data         map[string]*Entry
    cleanupTicker *time.Ticker
    done         chan struct{}
}
```

**Key operations**:
- `Set(key, value, ttl)` - Store with automatic expiration
- `Get(key)` - Retrieve with freshness check
- `GetLastModified(key)` - Timestamp-only query for cache invalidation
- `Delete(key)` - Explicit removal
- `Cleanup()` - Background TTL enforcement

## Why This Pattern Matters

Last-modified tracking with TTL is fundamental to distributed systems:

- **Cache invalidation**: "Has this data changed since I last read it?"
- **Distributed coordination**: Consensus on resource freshness across nodes
- **Session management**: Expiring user sessions with timestamp tracking
- **Metadata stores**: Tracking when configurations last changed
- **Event sourcing**: Recording when state transitions occurred

This appears in Redis, etcd, Consul, and every distributed cache layer.

## Concurrency Patterns Explored

**Read-heavy optimization**:
```go
// RWMutex allows multiple concurrent readers
store.mu.RLock()
entry, exists := store.data[key]
store.mu.RUnlock()
```

**Write synchronization**:
```go
// Exclusive lock for modifications
store.mu.Lock()
store.data[key] = &Entry{...}
store.mu.Unlock()
```

**Background cleanup**:
```go
// Periodic TTL enforcement without blocking operations
ticker := time.NewTicker(cleanupInterval)
go func() {
    for {
        select {
        case <-ticker.C:
            store.cleanup()
        case <-store.done:
            return
        }
    }
}()
```

## Testing Approach

Progressive TDD with Ginkgo/Gomega:

1. **Phase 1**: Basic CRUD operations (single-threaded)
2. **Phase 2**: TTL expiration and cleanup logic
3. **Phase 3**: Concurrent access patterns (race detector)
4. **Phase 4**: Edge cases (expired entries, nil values, cleanup during read)
5. **Phase 5**: Performance validation (benchmark read vs write contention)

All tests run with `-race` flag to verify thread safety.

## Running

```bash
# Run tests with race detector
go test -race ./...

# Run with Ginkgo verbose output
ginkgo -v ./...

# Run benchmarks
go test -bench=. -benchmem ./...

# Build example usage
go build -o example ./cmd/example
```

## Example Usage

```go
// Create store with 1-minute cleanup interval
store := NewLastModStore(1 * time.Minute)
defer store.Close()

// Store value with 5-minute TTL
store.Set("user:123", userData, 5*time.Minute)

// Retrieve value
value, exists := store.Get("user:123")

// Check last modification time (for cache invalidation)
lastMod, exists := store.GetLastModified("user:123")
if exists && time.Since(lastMod) > threshold {
    // Refresh from source
}

// Explicit deletion
store.Delete("user:123")
```

## Design Decisions

**Why RWMutex over Mutex?**  
Read-heavy workloads benefit from concurrent readers. Metadata queries often outnumber writes 10:1 in distributed systems.

**Why background cleanup over lazy deletion?**  
Prevents unbounded memory growth when keys are set but never read again. Trade-off: cleanup goroutine overhead vs guaranteed memory bounds.

**Why time.Time over Unix timestamps?**  
Go's `time.Time` provides better API ergonomics and handles timezone/DST edge cases automatically.

**Why interface{} values?**  
Demonstrates generic storage pattern pre-Go 1.18. Real implementation would use generics for type safety.

## Real-World Applications

This pattern scales from in-process caches to distributed systems:

- **Service mesh metadata**: Tracking when service endpoints last updated
- **Configuration management**: Detecting stale config across deployments  
- **API rate limiting**: Windowed counters with TTL expiration
- **Feature flag systems**: Cached flags with invalidation timestamps
- **Distributed locks**: Lease expiration with last-held tracking

## Skills Demonstrated

**Go Fundamentals**:
- Mutex patterns (Mutex vs RWMutex trade-offs)
- Goroutine lifecycle management
- Channel-based signaling (done channel)
- Time-based operations and tickers

**Production Practices**:
- Graceful shutdown patterns
- Resource cleanup in destructors
- Thread-safe API design
- Memory-bounded data structures

**Testing Rigor**:
- Race condition detection
- Concurrent access scenarios
- TTL boundary testing
- Performance benchmarking

## Background

Senior Platform Engineer (28 years) maintaining Go fundamentals between infrastructure projects. This repo represents deliberate practice - isolating concurrency patterns that underpin production distributed systems.

**Why share practice code?** Because seeing how senior engineers approach continuous learning matters. Not everything needs to be a finished product - sometimes exploring the fundamentals is the point.

## Development Notes

Built using AI-assisted pair programming (Claude) to explore modern development workflows while maintaining rigorous TDD practices and deep technical fundamentals.

## License

MIT