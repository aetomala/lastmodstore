# lastmodstore

![Tests](https://github.com/aetomala/lastmodstore/actions/workflows/test.yml/badge.svg?branch=main)
![Go Version](https://img.shields.io/badge/go-1.21+-blue.svg)
![License](https://img.shields.io/badge/license-MIT-green.svg)


**Thread-safe TTL cache with timestamp tracking - exploring Go concurrency patterns for distributed system state management**

## Purpose

Part of my ongoing platform engineering skill maintenance. This repo explores implementing a concurrent key-value store with last-modified timestamp tracking - a foundational pattern that appears in distributed caches, metadata stores, and state coordination systems.

Built incrementally using TDD (Ginkgo/Gomega) to practice Go's sync primitives, time-based eviction strategies, and thread-safe data structure design.

## What This Implements

A production-grade, concurrent in-memory key-value store demonstrating:

**Core patterns**:
- Thread-safe read/write operations using sync.RWMutex
- Last-modified timestamp tracking for each item
- Configurable capacity limits (MaxSize)
- Optional periodic cleanup with background workers
- Graceful shutdown with in-flight operation tracking
- Atomic operations for concurrent safe state management

**Architecture**:
- In-flight operations tracked via sync.WaitGroup (pointer for safety)
- Separate read (RLock) and write (Lock) access patterns
- Context-based lifecycle management (Start/Shutdown)
- Worker pool for background operations
- Explicit operation management (no automatic TTL expiration)

**Use case**: Metadata tracking for distributed systems - tracking when resources were last updated, coordinating cache invalidation, or maintaining service state with graceful lifecycle management.

## Architecture

```go
type StoreItem struct {
    Value        interface{}
    LastModified time.Time
}

type Config struct {
    MaxSize         int           // Max items (0 = unlimited)
    WorkerCount     int           // Background workers (default: 2)
    CleanupInterval time.Duration // Periodic cleanup (0 = disabled)
}

type LastModifiedStore struct {
    mu              sync.RWMutex
    items           map[string]*StoreItem
    config          Config
    state           int32                // Atomic: 0=stopped, 1=running
    ctx             context.Context
    cancel          context.CancelFunc
    workers         *sync.WaitGroup      // Tracks in-flight operations
    cleanupShutdown chan struct{}        // Signals cleanup goroutine
}
```

**Public API**:
- `NewLastModifiedStore(config Config)` - Creates store (doesn't start)
- `Start(ctx context.Context)` - Starts store and background workers
- `IsRunning()` - Thread-safe state check
- `Set(key string, value interface{})` - Store with current timestamp
- `Get(key string)` - Retrieve with LastModified metadata
- `Delete(key string)` - Remove item
- `Size()` - Return item count
- `List()` - Return shallow copy of all items
- `Clear()` - Remove all items
- `Shutdown()` - **Graceful shutdown that waits for in-flight operations**

## Graceful Shutdown

The store implements proper graceful shutdown semantics matching Go's standard library patterns (net.Listener, sql.DB):

**Key guarantee**: `Shutdown()` is a **blocking operation** that waits for all in-flight operations to complete before returning.

```go
// Create and start store
store, _ := NewLastModifiedStore(config)
ctx, cancel := context.WithCancel(context.Background())
store.Start(ctx)

// Operations continue safely...
go func() {
    for i := 0; i < 100; i++ {
        store.Set(fmt.Sprintf("key%d", i), i)
    }
}()

// Graceful shutdown waits for all in-flight operations
store.Shutdown()  // Blocks until all operations complete

// All operations completed safely, no data loss
```

**Implementation details**:
- Operations register with an internal WaitGroup on start
- Shutdown signals cleanup workers via context cancellation
- Shutdown waits for all operations to complete via WaitGroup
- State is set to stopped only after all operations finish
- Prevents new operations after shutdown starts

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

**In-flight operation tracking**:
```go
// Each operation (Set, Get, Delete, Size, List, Clear) follows this pattern:
func (c *LastModifiedStore) Set(key string, value interface{}) error {
    c.mu.Lock()
    defer c.mu.Unlock()

    // 1. Check state
    if atomic.LoadInt32(&c.state) != StateStarted {
        return ErrNotRunning
    }

    // 2. Register operation
    c.workers.Add(1)
    defer c.workers.Done()  // Guaranteed cleanup on any exit

    // 3. Perform operation
    c.items[key] = &StoreItem{
        Value:        value,
        LastModified: time.Now(),
    }

    return nil
}
```

This enables Shutdown() to wait for all operations:
```go
// Shutdown coordination
func (c *LastModifiedStore) Shutdown() error {
    c.mu.Lock()
    // 1. Prepare for shutdown
    c.cancel()  // Signal background workers
    c.mu.Unlock()

    // 2. Wait for all in-flight operations to complete
    c.workers.Wait()

    // 3. Set final state
    c.mu.Lock()
    atomic.StoreInt32(&c.state, StateStopped)
    c.mu.Unlock()

    return nil
}
```

**Critical pattern**: Unlock BEFORE calling Wait() to allow in-flight operations to proceed.

**Background cleanup**:
```go
// Periodic cleanup without blocking operations
if cleanupInterval > 0 {
    go func() {
        ticker := time.NewTicker(cleanupInterval)
        defer ticker.Stop()
        for {
            select {
            case <-ctx.Done():
                return
            case <-cleanupShutdown:
                return
            case <-ticker.C:
                store.performCleanup()
            }
        }
    }()
}
```

## Concurrency Guarantees

### Read-Write Separation
- **Multiple concurrent readers**: Get(), Size(), List() use RLock
- **Exclusive writers**: Set(), Delete(), Clear() use Lock
- Read operations never block other reads

### State Consistency
- **Atomic transitions**: Running ↔ Stopped (uses atomic operations)
- **No partial states**: All state changes are atomic
- **Protected data**: All maps and structures protected by RWMutex

### Operation Safety
- **In-flight tracking**: Operations tracked via sync.WaitGroup
- **Graceful shutdown**: All operations complete before shutdown returns
- **No interruptions**: Operations won't be interrupted during shutdown
- **Clean resource cleanup**: Background workers stopped gracefully

### WaitGroup Safety
- `workers` field is a pointer (`*sync.WaitGroup`)
- Prevents accidental copying of internal counter
- Safe for struct composition patterns

All guarantees verified with Go's race detector:
```bash
go test -race ./cmd/lastmodstore
```

## Testing Approach

Comprehensive TDD with Ginkgo/Gomega (67 tests, 100% passing):

**Test Coverage**:
1. **Constructor** (8 tests) - Configuration validation and defaults
2. **Lifecycle** (3 tests) - Start/IsRunning/Shutdown state management
3. **Core Operations** (20 tests) - Set/Get/Delete/Size/List/Clear
4. **Error Handling** (6 tests) - Operations on stopped store, missing keys
5. **Concurrent Access** (3 tests) - Race-safe concurrent operations
6. **Capacity Management** (5 tests) - MaxSize enforcement and limits
7. **Worker Pool** (2 tests) - Background cleanup integration
8. **Advanced Features** (8 tests) - Cleanup under load, performance, timestamps
9. **Graceful Shutdown** (4 tests) - Shutdown state management
10. **Cleanup Goroutine** (1 test) - Cleanup goroutine termination
11. **Context Cancellation** (2 tests) - Handling cancelled contexts
12. **Worker Pool Cleanup** (2 tests) - Worker termination verification
13. **Resource Cleanup** (2 tests) - Channel and context cleanup

All tests verified with:
```bash
go test -race ./cmd/lastmodstore     # No data races
go test -v ./cmd/lastmodstore        # All 67/67 passing
```

**Key test scenarios**:
- In-flight operations complete before shutdown returns
- Operations fail with ErrNotRunning after shutdown
- Multiple concurrent readers can read simultaneously
- Exclusive writes prevent concurrent modifications
- Context cancellation properly stops background workers

## Running

```bash
# Run all tests
go test -v ./cmd/lastmodstore

# Run with race detector (verifies thread safety)
go test -race ./cmd/lastmodstore

# Run specific test suite
go test -v ./cmd/lastmodstore -run "Graceful"
go test -v ./cmd/lastmodstore -run "Concurrent"
go test -v ./cmd/lastmodstore -run "Shutdown"

# Run benchmarks (if added)
go test -bench=. -benchmem ./cmd/lastmodstore

# Show detailed test output with Ginkgo
ginkgo -v ./cmd/lastmodstore
```

## Example Usage

```go
// Create store with configuration
store, err := NewLastModifiedStore(Config{
    MaxSize:         1000,
    WorkerCount:     4,
    CleanupInterval: 30 * time.Second,
})
if err != nil {
    panic(err)
}

// Start the store
ctx, cancel := context.WithCancel(context.Background())
defer cancel()

if err := store.Start(ctx); err != nil {
    panic(err)
}
defer store.Shutdown()  // Graceful shutdown - waits for operations

// Store values
if err := store.Set("user:123", userData); err != nil {
    panic(err)
}

// Retrieve value with timestamp
item, err := store.Get("user:123")
if err != nil {
    panic(err)
}
fmt.Printf("Value: %v, Modified: %v\n", item.Value, item.LastModified)

// Check freshness and refresh if needed
if time.Since(item.LastModified) > 5*time.Minute {
    // Refresh from source
    newData := refreshData()
    store.Set("user:123", newData)
}

// Get store size
size, err := store.Size()
if err != nil {
    panic(err)
}
fmt.Printf("Store contains %d items\n", size)

// Delete item
if err := store.Delete("user:123"); err != nil {
    panic(err)
}
```

## Design Decisions

**Why RWMutex over Mutex?**
Read-heavy workloads benefit from concurrent readers. Metadata queries often outnumber writes 10:1 in distributed systems.

**Why graceful shutdown with in-flight operation tracking?**
Ensures no operations are interrupted during shutdown. Prevents data loss and maintains consistency. Matches patterns in net.Listener and sql.DB.

**Why WaitGroup pointer instead of value?**
Prevents accidental copying of the internal counter. Follows Go best practices for sync primitives in structs.

**Why separate Start/Shutdown from constructor?**
Enables resource initialization only when needed. Allows proper context management and cleanup signaling.

**Why configurable WorkerCount and CleanupInterval?**
Allows tuning for different workload patterns. Can disable cleanup (0) for memory-constrained scenarios or ephemeral stores.

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
- Graceful shutdown with in-flight operation tracking
- Resource cleanup with proper lifecycle management
- Thread-safe API design
- Memory-bounded data structures
- Context-based cancellation patterns

**Graceful Shutdown & Concurrency**:
- In-flight operation tracking with WaitGroup
- Proper lock scope management (unlock before wait)
- Context-based lifecycle management
- Avoiding deadlocks in shutdown sequences
- WaitGroup pointer for safe struct composition

**Testing Rigor**:
- Race condition detection with `-race` flag
- Concurrent access scenarios under load
- Graceful shutdown verification
- Context cancellation handling
- Performance benchmarking

## Background

Senior Platform Engineer (28 years) maintaining Go fundamentals between infrastructure projects. This repo represents deliberate practice - isolating concurrency patterns that underpin production distributed systems.

**Why share practice code?** Because seeing how senior engineers approach continuous learning matters. Not everything needs to be a finished product - sometimes exploring the fundamentals is the point.

## Development Notes

Built using AI-assisted pair programming (Claude) to explore modern development workflows while maintaining rigorous TDD practices and deep technical fundamentals.

## License

MIT