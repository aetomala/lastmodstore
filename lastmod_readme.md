# LastModifiedStore - In-Memory Key-Value Store with Last-Modified Tracking

A thread-safe, in-memory key-value store implementation in Go that tracks per-item last-modified timestamps, featuring lifecycle management and a worker pool pattern for background operations.

## Project Overview

This project is designed as a **progressive development exercise** to practice:
- Thread-safe data structure implementation
- Lifecycle management patterns (Start/Stop/IsRunning)
- Worker pool architecture
- Concurrent operations with proper synchronization
- Test-Driven Development (TDD) with Ginkgo/Gomega
- Progressive, phase-based testing methodology

## Features

### Core Capabilities
- **Thread-Safe Operations**: All operations protected by RWMutex for concurrent access
- **Per-Item Metadata**: Each stored item tracks its last modification timestamp
- **Lifecycle Management**: Proper Start/Shutdown patterns with state tracking
- **Worker Pool Pattern**: Background job processing with configurable worker count
- **Capacity Management**: Optional MaxSize enforcement with overflow protection
- **Cleanup Operations**: Periodic background maintenance tasks

### Operations Supported
- `Set(key, value)` - Store or update items with automatic timestamp tracking
- `Get(key)` - Retrieve items with metadata
- `Delete(key)` - Remove items from store
- `List()` - Get all items with metadata
- `Clear()` - Remove all items
- `Size()` - Get current item count

## Project Structure

```
lastmodstore/
├── README.md                    # This file
├── store.go                     # Main implementation (to be created)
├── worker_pool.go               # Worker pool implementation (to be created)
├── errors.go                    # Error definitions (to be created)
├── types.go                     # Type definitions (to be created)
├── store_test.go                # Ginkgo test suite (provided)
└── specification.go             # Project specification (provided)
```

## Getting Started

### Prerequisites
```bash
go version  # Requires Go 1.21 or higher
```

### Installation

1. **Clone or create the project directory**:
```bash
mkdir lastmodstore
cd lastmodstore
go mod init github.com/yourusername/lastmodstore
```

2. **Install testing dependencies**:
```bash
go get github.com/onsi/ginkgo/v2
go get github.com/onsi/gomega
```

3. **Initialize Ginkgo** (if not already done):
```bash
go run github.com/onsi/ginkgo/v2/ginkgo bootstrap
```

## Development Approach

This project follows a **progressive, phase-based development methodology**. Each phase builds incrementally on the previous phase, allowing you to:
- Develop one feature at a time
- Run tests after each phase to verify correctness
- Build confidence as you progress
- Avoid overwhelming complexity

### Phase-Based Implementation Guide

#### **PHASE 1: Basic Construction** 
**Goal**: Implement the constructor with configuration validation

**What to implement**:
```go
// types.go
type Config struct {
    MaxSize         int
    WorkerCount     int
    CleanupInterval time.Duration
}

type StoreItem struct {
    Value        interface{}
    LastModified time.Time
}

type LastModifiedStore struct {
    mu           sync.RWMutex
    items        map[string]*StoreItem
    config       Config
    running      bool
    // Other fields added in later phases
}

// store.go
func NewLastModifiedStore(config Config) (*LastModifiedStore, error)
```

**Test command**:
```bash
ginkgo --focus="Constructor" ./...
```

**Success criteria**:
- ✅ All constructor tests pass
- ✅ Configuration validation works correctly
- ✅ Default values are set properly
- ✅ Invalid configs return appropriate errors

---

#### **PHASE 2: Core Initialization**
**Goal**: Implement Start() and IsRunning() methods

**What to implement**:
```go
func (s *LastModifiedStore) Start(ctx context.Context) error
func (s *LastModifiedStore) IsRunning() bool
```

**Test command**:
```bash
ginkgo --focus="Start|IsRunning" ./...
```

**Success criteria**:
- ✅ Store can be started successfully
- ✅ Running state is tracked correctly
- ✅ Double-start is prevented
- ✅ Nil context is rejected
- ✅ Thread-safe state checking works

---

#### **PHASE 3: Basic Functionality**
**Goal**: Implement core CRUD operations

**What to implement**:
```go
func (s *LastModifiedStore) Set(key string, value interface{}) error
func (s *LastModifiedStore) Get(key string) (*StoreItem, error)
func (s *LastModifiedStore) Delete(key string) error
func (s *LastModifiedStore) Size() (int, error)
func (s *LastModifiedStore) List() (map[string]*StoreItem, error)
func (s *LastModifiedStore) Clear() error
```

**Test command**:
```bash
ginkgo --focus="Core Operations" ./...
```

**Success criteria**:
- ✅ Can store and retrieve items
- ✅ Last-modified timestamps are accurate
- ✅ Delete removes items correctly
- ✅ Size reflects actual item count
- ✅ List returns all items
- ✅ Clear removes everything

---

#### **PHASE 4: Error Handling**
**Goal**: Ensure operations fail gracefully when store not running

**What to implement**:
```go
// errors.go
var (
    ErrNotRunning     = errors.New("store is not running")
    ErrAlreadyRunning = errors.New("store is already running")
    ErrNotFound       = errors.New("key not found")
    ErrStoreFull      = errors.New("store has reached maximum capacity")
    ErrInvalidConfig  = errors.New("invalid configuration")
    ErrNilContext     = errors.New("context cannot be nil")
)
```

**Test command**:
```bash
ginkgo --focus="Operations When Not Running" ./...
```

**Success criteria**:
- ✅ All operations return ErrNotRunning when not started
- ✅ Error messages are clear and actionable

---

#### **PHASE 5: Concurrent Operations**
**Goal**: Verify thread-safety under concurrent load

**What to verify**:
- RWMutex is properly used in all operations
- No race conditions exist
- Concurrent reads work simultaneously
- Writes are properly serialized

**Test command**:
```bash
go test -race ./...
ginkgo --focus="Concurrent Access" ./...
```

**Success criteria**:
- ✅ No race conditions detected
- ✅ Concurrent operations complete successfully
- ✅ Data integrity maintained under load

---

#### **PHASE 6: Capacity Management**
**Goal**: Implement MaxSize enforcement

**What to implement**:
- Check size before adding new items
- Allow updates to existing items even when full
- Return ErrStoreFull when appropriate

**Test command**:
```bash
ginkgo --focus="MaxSize Enforcement" ./...
```

**Success criteria**:
- ✅ Cannot exceed MaxSize with new items
- ✅ Can update existing items when at capacity
- ✅ Can add after deletion frees space

---

#### **PHASE 7: Worker Pool Integration**
**Goal**: Implement worker pool for background operations

**What to implement**:
```go
// worker_pool.go
type Job struct {
    ID          string
    Type        JobType
    Data        interface{}
    SubmittedAt time.Time
}

type JobType int

const (
    JobTypeCleanup JobType = iota
    JobTypeValidation
)

type WorkerPool struct {
    workers  []*Worker
    jobQueue chan Job
    resultCh chan JobResult
    wg       sync.WaitGroup
    ctx      context.Context
}

func NewWorkerPool(ctx context.Context, workerCount int) *WorkerPool
func (wp *WorkerPool) Start() error
func (wp *WorkerPool) Submit(job Job) error
func (wp *WorkerPool) Stop() error
```

**Integration**:
- Start worker pool in Store.Start()
- Launch cleanup goroutine if CleanupInterval > 0
- Submit periodic cleanup jobs

**Test command**:
```bash
ginkgo --focus="Worker Pool" ./...
```

**Success criteria**:
- ✅ Worker pool starts and stops cleanly
- ✅ Cleanup goroutine runs when enabled
- ✅ Store remains functional during cleanup

---

#### **ADVANCED FEATURES**
**Goal**: Implement sophisticated features

**Test command**:
```bash
ginkgo --focus="Advanced Features" ./...
```

**What to implement**:
- Performance optimization
- Advanced cleanup logic
- Statistics tracking
- Monitoring hooks

---

#### **SHUTDOWN SUITE** (Implement Last)
**Goal**: Graceful shutdown with resource cleanup

**What to implement**:
```go
func (s *LastModifiedStore) Shutdown() error
```

**Must handle**:
- Stop accepting new operations
- Wait for in-flight operations
- Stop worker pool gracefully
- Cancel cleanup goroutine
- Close all channels
- Cancel internal context

**Test command**:
```bash
ginkgo --focus="Shutdown" ./...
```

**Success criteria**:
- ✅ Graceful shutdown with no goroutine leaks
- ✅ In-flight operations complete
- ✅ Idempotent shutdown behavior
- ✅ Context cancellation handled properly
- ✅ All resources cleaned up

---

## Running Tests

### Run All Tests
```bash
ginkgo -v ./...
```

### Run Specific Phase
```bash
# Phase 1
ginkgo --focus="Constructor" ./...

# Phase 2
ginkgo --focus="Start|IsRunning" ./...

# Phase 3
ginkgo --focus="Core Operations" ./...

# Phase 4
ginkgo --focus="Operations When Not Running" ./...

# Phase 5
ginkgo --focus="Concurrent Access" ./...

# Phase 6
ginkgo --focus="MaxSize" ./...

# Phase 7
ginkgo --focus="Worker Pool" ./...

# Advanced
ginkgo --focus="Advanced Features" ./...

# Shutdown
ginkgo --focus="Shutdown" ./...
```

### Run with Race Detection
```bash
go test -race ./...
```

### Run with Coverage
```bash
ginkgo -cover -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

## Implementation Tips

### Thread Safety
- **Use RWMutex appropriately**:
  - `RLock()`/`RUnlock()` for read operations (Get, List, Size)
  - `Lock()`/`Unlock()` for write operations (Set, Delete, Clear)
- **Keep critical sections small**: Lock, do work, unlock immediately
- **Avoid nested locks**: Can cause deadlocks

### Timestamp Tracking
```go
item := &StoreItem{
    Value:        value,
    LastModified: time.Now(),
}
```

### Worker Pool Pattern
```go
// Worker processes jobs from queue
func (w *Worker) run() {
    for {
        select {
        case job := <-w.jobQueue:
            // Process job
            w.processJob(job)
        case <-w.ctx.Done():
            return
        }
    }
}
```

### Graceful Shutdown
```go
func (s *LastModifiedStore) Shutdown() error {
    s.mu.Lock()
    if !s.running {
        s.mu.Unlock()
        return ErrNotRunning
    }
    s.running = false
    s.mu.Unlock()

    // Cancel context to stop goroutines
    s.cancel()

    // Stop worker pool
    if s.workers != nil {
        s.workers.Stop()
    }

    // Close channels
    close(s.shutdownCh)

    return nil
}
```

## Common Pitfalls to Avoid

1. **Forgetting to check running state**: Always check `s.running` in operations
2. **Holding locks too long**: Can cause performance issues
3. **Not copying data in List()**: Return a copy, not the internal map
4. **Ignoring context cancellation**: Check `ctx.Done()` in long-running operations
5. **Goroutine leaks**: Ensure all goroutines exit on shutdown
6. **Not handling double-shutdown**: Make Shutdown() idempotent

## Example Usage

```go
package main

import (
    "context"
    "fmt"
    "time"
    "yourproject/lastmodstore"
)

func main() {
    // Create store
    config := lastmodstore.Config{
        MaxSize:         100,
        WorkerCount:     4,
        CleanupInterval: 30 * time.Second,
    }
    
    store, err := lastmodstore.NewLastModifiedStore(config)
    if err != nil {
        panic(err)
    }
    
    // Start store
    ctx := context.Background()
    if err := store.Start(ctx); err != nil {
        panic(err)
    }
    defer store.Shutdown()
    
    // Store data
    store.Set("user:123", map[string]string{
        "name":  "John Doe",
        "email": "john@example.com",
    })
    
    // Retrieve data
    item, err := store.Get("user:123")
    if err != nil {
        panic(err)
    }
    
    fmt.Printf("Value: %v\n", item.Value)
    fmt.Printf("Last Modified: %v\n", item.LastModified)
    
    // List all items
    items, _ := store.List()
    for key, item := range items {
        fmt.Printf("%s: %v (modified: %v)\n", 
            key, item.Value, item.LastModified)
    }
}
```

## Development Checklist

Track your progress through the implementation:

- [ ] **Phase 1**: Constructor and config validation
- [ ] **Phase 2**: Start/IsRunning implementation
- [ ] **Phase 3**: Core CRUD operations
- [ ] **Phase 4**: Error handling
- [ ] **Phase 5**: Concurrent operations (race-free)
- [ ] **Phase 6**: Capacity management
- [ ] **Phase 7**: Worker pool integration
- [ ] **Advanced**: Performance and monitoring
- [ ] **Shutdown**: Graceful cleanup
- [ ] **Documentation**: Add code comments
- [ ] **Examples**: Create usage examples
- [ ] **Benchmarks**: Add performance tests

## Testing Strategy

### Unit Tests (Provided)
- Progressive phase-based Ginkgo tests
- Covers all functionality incrementally
- Tests thread-safety and concurrency
- Validates error handling
- Ensures graceful shutdown

### What You Should Add
1. **Benchmark Tests**:
```go
func BenchmarkStoreSet(b *testing.B) {
    store, _ := NewLastModifiedStore(Config{})
    ctx := context.Background()
    store.Start(ctx)
    defer store.Shutdown()
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
        store.Set(fmt.Sprintf("key%d", i), i)
    }
}
```

2. **Integration Tests**: Test with real-world usage patterns

3. **Stress Tests**: High-volume concurrent operations

## Learning Objectives

By completing this project, you will practice:

✅ **Concurrent Programming**: Thread-safe data structures with mutexes  
✅ **Lifecycle Management**: Proper start/stop patterns  
✅ **Worker Pool Pattern**: Background job processing architecture  
✅ **Context Usage**: Cancellation and timeout handling  
✅ **Error Handling**: Comprehensive error management  
✅ **Test-Driven Development**: Progressive, phase-based testing  
✅ **Resource Management**: Preventing goroutine and memory leaks  
✅ **API Design**: Clean, intuitive interfaces  

## Additional Resources

- [Go Concurrency Patterns](https://go.dev/blog/pipelines)
- [Effective Go - Concurrency](https://go.dev/doc/effective_go#concurrency)
- [Ginkgo Documentation](https://onsi.github.io/ginkgo/)
- [Gomega Matchers](https://onsi.github.io/gomega/)

## License

This is an educational project for learning purposes.

## Contributing

This is a practice project. Feel free to:
- Extend functionality
- Add more test cases
- Improve documentation
- Create example applications

---

**Happy Coding! Start with Phase 1 and progress incrementally through each phase.** 🚀