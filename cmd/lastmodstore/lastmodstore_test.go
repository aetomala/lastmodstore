package lastmodstore_test

import (
	"context"
	"sync"
	"testing"
	"time"

	. "github.com/aetomala/lastmodstore/cmd/lastmodstore"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

func TestLastModStore(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "LastModifiedStore Suite")
}

var _ = Describe("LastModifiedStore", func() {
	var (
		store  *LastModifiedStore
		ctx    context.Context
		cancel context.CancelFunc
		config Config
	)
	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		config = Config{
			MaxSize:         0,
			WorkerCount:     2,
			CleanupInterval: 0,
		}
	})

	AfterEach(func() {
		if store != nil && store.IsRunning() {
			err := store.Shutdown()
			Expect(err).NotTo(HaveOccurred())
		}
		if cancel != nil {
			cancel()
		}
	})

	// === PHASE 1: Basic Constructor ===
	Describe("Constructor", func() {
		Context("with valid configuration", func() {
			It("should create store with default config", func() {
				store, err := NewLastModifiedStore(Config{})
				Expect(err).NotTo(HaveOccurred())
				Expect(store).NotTo(BeNil())
			})
			It("shoud create Store with custom MaxSize", func() {
				config.MaxSize = 100
				store, err := NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(store).NotTo(BeNil())
			})
			It("should create store with custom WorkerCount", func() {
				config.WorkerCount = 5
				store, err := NewLastModifiedStore(config)
				Expect(err).ToNot(HaveOccurred())
				Expect(store).NotTo(BeNil())
			})
			It("should create store with CleanupInterval", func() {
				config.CleanupInterval = 30 * time.Second
				store, err := NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(store).NotTo(BeNil())
			})
			It("should crete store with default to 2 when zero", func() {
				config.WorkerCount = 0
				store, err := NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(store).NotTo(BeNil())
				//Implementation should default to 2 workers
			})
		})
		Context("with invalid configuration", func() {
			It("should return error for negative MaxSize", func() {
				config.MaxSize = -1
				store, err := NewLastModifiedStore(config)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrInvalidConfig))
				Expect(store).To(BeNil())
			})
			It("should return error for negative WorkerCount", func() {
				config.WorkerCount = -1
				store, err := NewLastModifiedStore(config)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrInvalidConfig))
				Expect(store).To(BeNil())
			})
			It("should return error for nevative CleanupInterval", func() {
				config.CleanupInterval = -1 * time.Second
				store, err := NewLastModifiedStore(config)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrInvalidConfig))
				Expect(store).To(BeNil())
			})
		})
		Context("initial state", func() {
			It("should not be running after constructor", func() {
				store, err := NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				Expect(store.IsRunning()).To(BeFalse())
			})
		})
	})

	// === PHASE 2: Core Initialization ===
	Describe("Start", func() {
		BeforeEach(func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
		})

		Context("with valid context", func() {
			It("should start successfully", func() {
				err := store.Start(ctx)
				Expect(err).ToNot(HaveOccurred())
			})
			It("should set running state to true", func() {
				err := store.Start(ctx)
				Expect(err).NotTo(HaveOccurred())
				Expect(store.IsRunning()).To(BeTrue())
			})
			It("should initilize worker pool", func() {
				err := store.Start(ctx)
				// Worker pool functionality will be tested in Phase 7
				// For now, just verify Start succeeded and store is running
				Expect(err).NotTo(HaveOccurred())
				// Worker pool should be running
			})
		})

		Context("when already running", func() {
			It("should return error on a second start", func() {
				err := store.Start(ctx)
				Expect(err).NotTo(HaveOccurred())

				err = store.Start(ctx)
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrAlreadyRunning))
			})
		})
	})

	Describe("IsRunning", func() {
		BeforeEach(func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return false before start", func() {
			Expect(store.IsRunning()).To(BeFalse())
		})

		It("should return true after start", func() {
			err := store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
			Expect(store.IsRunning()).To(BeTrue())
		})

		It("should be thread-safe", func() {
			err := store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup

			for i := 0; i < 100; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					_ = store.IsRunning()
				}()
			}
			wg.Wait()
		})
	})

	// === PHASE 3: Basic Functionality ===
	Describe("Core Operations", func() {
		BeforeEach(func() {
			var err error
			config.MaxSize = 3
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		Describe("Set", func() {
			It("should store a new item", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update existing item", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(10 * time.Millisecond)

				err = store.Set("key1", "value2")
				Expect(err).NotTo(HaveOccurred())
			})

			It("should update last-modified timestamp on update", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				item1, err := store.Get("key1")
				Expect(err).NotTo(HaveOccurred())
				firstModified := item1.LastModified

				time.Sleep(10 * time.Millisecond)

				err = store.Set("key1", "value2")
				Expect(err).NotTo(HaveOccurred())

				item2, err := store.Get("key1")
				Expect(err).NotTo(HaveOccurred())
				Expect(item2.LastModified).To(BeTemporally(">", firstModified))
			})

			It("should accept various value types", func() {
				err := store.Set("string", "vlue")
				Expect(err).NotTo(HaveOccurred())

				err = store.Set("int", 42)
				Expect(err).NotTo(HaveOccurred())

				err = store.Set("struct", struct{ Name string }{"test"})
				Expect(err).NotTo(HaveOccurred())
			})
		})
		Describe("Get", func() {
			It("should retrieve stored item", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				item, err := store.Get("key1")
				Expect(err).NotTo(HaveOccurred())
				Expect(item).NotTo(BeNil())
				Expect(item.Value).To(Equal("value1"))
			})

			It("should return item with last-modified timestamp", func() {
				before := time.Now()
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())
				after := time.Now()

				item, err := store.Get("key1")
				Expect(err).NotTo(HaveOccurred())
				Expect(item.LastModified).To(BeTemporally(">=", before))
				Expect(item.LastModified).To(BeTemporally("<=", after))
			})

			It("should return error for non-existent key", func() {
				item, err := store.Get("nonexistent")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrNotFound))
				Expect(item).To(BeNil())
			})
		})

		Describe("Delete", func() {
			It("should remove existing item", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				err = store.Delete("key1")
				Expect(err).NotTo(HaveOccurred())

				_, err = store.Get("key1")
				Expect(err).To(MatchError(ErrNotFound))
			})

			It("should return error for non-existing key", func() {
				err := store.Delete("nonexistent")
				Expect(err).To(HaveOccurred())
				Expect(err).To(MatchError(ErrNotFound))
			})
		})

		Describe("Size", func() {
			It("should return correct size after adding items", func() {
				err := store.Set("key1", "value1")
				Expect(err).ToNot(HaveOccurred())
				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))
			})

			It("should return correct size after adding items", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))

				err = store.Set("key2", "value2")
				Expect(err).NotTo(HaveOccurred())

				size, err = store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(2))
			})

			It("should return correct size after deletion", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())
				err = store.Set("key2", "value2")
				Expect(err).NotTo(HaveOccurred())

				err = store.Delete("key1")
				Expect(err).NotTo(HaveOccurred())

				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))
			})
		})

		Describe("List", func() {
			It("should return empty map for empty store", func() {
				items, err := store.List()
				Expect(err).NotTo(HaveOccurred())
				Expect(items).NotTo(BeNil())
				Expect(items).To(BeEmpty())
			})

			It("should return all items", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())
				err = store.Set("key2", "value2")
				Expect(err).NotTo(HaveOccurred())

				items, err := store.List()
				Expect(err).NotTo(HaveOccurred())
				Expect(items).To(HaveLen(2))
				Expect(items).To(HaveKey("key1"))
				Expect(items).To(HaveKey("key2"))
			})

			It("should return shallow copy", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				items, err := store.List()
				Expect(err).NotTo(HaveOccurred())

				// Modifyying returned map shouldn't affect the store
				delete(items, "key1")

				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))
			})
		})

		Describe("Clear", func() {
			It("should remove all items", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())
				err = store.Set("key2", "value2")
				Expect(err).NotTo(HaveOccurred())

				err = store.Clear()
				Expect(err).NotTo(HaveOccurred())

				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(0))
			})
		})
	})

	// === PHASE 4: Error Handling ===
	Describe("Operations When Not Running", func() {
		BeforeEach(func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
			// Note: NOT calling Start()
		})

		It("Set should return error", func() {
			err := store.Set("key1", "value1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})

		It("Get should return error", func() {
			_, err := store.Get("key1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})

		It("Delete should return error", func() {
			err := store.Delete("key1")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})

		It("Size should return error", func() {
			_, err := store.Size()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})

		It("List should return error", func() {
			_, err := store.List()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})

		It("Clear should return error", func() {
			err := store.Clear()
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})
	})

	// === PHASE 5: Concurrent Operations ===
	Describe("Concurrent Access", func() {
		BeforeEach(func() {
			var err error
			config.MaxSize = 100
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})

		It("should handle concurrent Set operations", func() {
			var wg sync.WaitGroup
			numOps := 100

			for i := 0; i < numOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					err := store.Set(string(rune('A'+idx%26)), idx)
					Expect(err).NotTo(HaveOccurred())
				}(i)
			}
			wg.Wait()

			size, err := store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(BeNumerically(">", 0))
		})

		It("should handle concurrent Get operations", func() {
			err := store.Set("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			var wg sync.WaitGroup
			numReads := 100

			for i := 0; i < numReads; i++ {
				wg.Add(1)
				go func() {
					defer wg.Done()
					item, err := store.Get("key1")
					Expect(err).NotTo(HaveOccurred())
					Expect(item.Value).To(Equal("value1"))
				}()
			}
			wg.Wait()
		})

		It("should handle mixed concurrent operations", func() {
			var wg sync.WaitGroup
			numOps := 50

			// Concurrent writes
			for i := 0; i < numOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					_ = store.Set(string(rune('A'+idx%10)), idx)
				}(i)
			}

			// Concurrent reads
			for i := 0; i < numOps; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					_, _ = store.Get(string(rune('A' + idx%10)))
				}(i)
			}

			wg.Wait()
		})
	})

	// === PHASE 6: Capacity Management ===
	Describe("MaxSize Enforcement", func() {
		BeforeEach(func() {
			config.MaxSize = 3
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())
			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())
		})
		AfterEach(func() {
			err := store.Clear()
			Expect(err).NotTo(HaveOccurred())
		})

		It("should allow adding items up to the MaxSize", func() {
			err := store.Set("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key2", "value2")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key3", "value3")
			Expect(err).NotTo(HaveOccurred())
		})

		It("should return error when exceeding MaxSize", func() {
			err := store.Set("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key2", "value2")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key3", "value3")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key4", "value4")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrStoreFull))
		})

		It("should allow updating existing keys when at MaxSize", func() {
			err := store.Set("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key2", "value2")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key3", "value3")
			Expect(err).NotTo(HaveOccurred())

			// Update existing key should succeed
			err = store.Set("key1", "updated")
			Expect(err).NotTo(HaveOccurred())

			item, err := store.Get("key1")
			Expect(err).NotTo(HaveOccurred())
			Expect(item.Value).To(Equal("updated"))
		})

		It("should allow adding after deletion", func() {
			err := store.Set("key1", "value1")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key2", "value2")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key3", "value3")
			Expect(err).NotTo(HaveOccurred())

			err = store.Delete("key1")
			Expect(err).NotTo(HaveOccurred())

			err = store.Set("key4", "value4")
			Expect(err).NotTo(HaveOccurred())
		})
	})

	// === PHASE 7: Worker Pool Integration ===
	Describe("Worker Pool", func() {
		Context("with cleanup enabled", func() {
			BeforeEach(func() {
				config.MaxSize = 1
				config.CleanupInterval = 50 * time.Millisecond
				var err error
				store, err = NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				err = store.Start(ctx)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should start cleanup goroutine", func() {
				// Add some items
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				// Wait for at least once clenup cycle
				time.Sleep(100 * time.Millisecond)

				// Store should still be functional
				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))
			})
		})

		Context("without cleanup", func() {
			BeforeEach(func() {
				config.MaxSize = 1
				config.CleanupInterval = 0
				var err error
				store, err = NewLastModifiedStore(config)
				Expect(err).NotTo(HaveOccurred())
				err = store.Start(ctx)
				Expect(err).NotTo(HaveOccurred())
			})

			It("should not start cleanup goroutine", func() {
				err := store.Set("key1", "value1")
				Expect(err).NotTo(HaveOccurred())

				time.Sleep(50 * time.Millisecond)

				size, err := store.Size()
				Expect(err).NotTo(HaveOccurred())
				Expect(size).To(Equal(1))
			})
		})
	})
})

// === Advanced Features Suite ===
var _ = Describe("LastModifiedStore Advanced Features", func() {
	var (
		store  *LastModifiedStore
		ctx    context.Context
		cancel context.CancelFunc
		config Config
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		config = Config{
			MaxSize:         0,
			WorkerCount:     4,
			CleanupInterval: 100 * time.Millisecond,
		}
		var err error
		store, err = NewLastModifiedStore(config)
		Expect(err).NotTo(HaveOccurred())
		err = store.Start(ctx)
		Expect(err).NotTo(HaveOccurred())
	})

	AfterEach(func() {
		if store != nil && store.IsRunning() {
			err := store.Shutdown()
			Expect(err).NotTo(HaveOccurred())
		}
		if cancel != nil {
			cancel()
		}
	})

	Describe("Cleanup Operations", func() {
		It("should periodically run cleanup jobs", func() {
			// Add items
			for i := 0; i < 10; i++ {
				err := store.Set(string(rune('A'+i)), i)
				Expect(err).NotTo(HaveOccurred())
			}

			// Wait for multiple cleanup cycles
			time.Sleep(250 * time.Millisecond)

			// Store should remain functional
			size, err := store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(10))
		})

		It("should handle cleanup with concurrent operations", func() {
			var wg sync.WaitGroup
			done := make(chan struct{})

			// Continuous writers
			for i := 0; i < 5; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					ticker := time.NewTicker(10 * time.Millisecond)
					defer ticker.Stop()

					for {
						select {
						case <-done:
							return
						case <-ticker.C:
							_ = store.Set(string(rune('A'+idx)), idx)
						}
					}
				}(i)
			}

			// Let operations run during cleanup cycles
			time.Sleep(300 * time.Millisecond)
			close(done)
			wg.Wait()

			// Verify store is still consistent
			size, err := store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(BeNumerically("<=", 5))
		})
	})

	Describe("Performance Characteristics", func() {
		It("should handle high-volume operations", func() {
			numOps := 1000
			start := time.Now()

			for i := 0; i < numOps; i++ {
				err := store.Set(string(rune('A'+i%26))+string(rune('A'+i/26)), i)
				Expect(err).NotTo(HaveOccurred())
			}

			duration := time.Since(start)
			Expect(duration).To(BeNumerically("<", 1*time.Second))
		})

		It("should maintain performance under concurrent load", func() {
			var wg sync.WaitGroup
			numGoroutines := 10
			opsPerGoroutine := 100

			start := time.Now()

			for g := 0; g < numGoroutines; g++ {
				wg.Add(1)
				go func(gid int) {
					defer wg.Done()
					for i := 0; i < opsPerGoroutine; i++ {
						key := string(rune('A'+gid)) + string(rune('0'+i%10))
						_ = store.Set(key, i)
						_, _ = store.Get(key)
					}
				}(g)
			}

			wg.Wait()
			duration := time.Since(start)

			Expect(duration).To(BeNumerically("<", 2*time.Second))
		})
	})

	Describe("Worker Pool Behavior", func() {
		It("should distribute work across workers", func() {
			// Submit multiple items to trigger worker pool activity
			for i := 0; i < 50; i++ {
				err := store.Set(string(rune('A'+i%26)), i)
				Expect(err).NotTo(HaveOccurred())
			}

			// Allow worker pool to process
			time.Sleep(200 * time.Millisecond)

			// Verify all items are present
			size, err := store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(26)) // 26 unique keys (A-Z)
		})
	})

	Describe("LastModified Tracking", func() {
		It("should maintain accurate timestamps under load", func() {
			baseTime := time.Now()

			// Set items with known timing
			err := store.Set("first", "value1")
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(50 * time.Millisecond)

			err = store.Set("second", "value2")
			Expect(err).NotTo(HaveOccurred())

			time.Sleep(50 * time.Millisecond)

			err = store.Set("third", "value3")
			Expect(err).NotTo(HaveOccurred())

			// Verify timestamp ordering
			first, err := store.Get("first")
			Expect(err).NotTo(HaveOccurred())

			second, err := store.Get("second")
			Expect(err).NotTo(HaveOccurred())

			third, err := store.Get("third")
			Expect(err).NotTo(HaveOccurred())

			Expect(first.LastModified).To(BeTemporally(">=", baseTime))
			Expect(second.LastModified).To(BeTemporally(">", first.LastModified))
			Expect(third.LastModified).To(BeTemporally(">", second.LastModified))
		})

		It("should update timestamps on every modification", func() {
			err := store.Set("key", "value1")
			Expect(err).NotTo(HaveOccurred())

			timestamps := make([]time.Time, 5)
			item, err := store.Get("key")
			Expect(err).NotTo(HaveOccurred())
			timestamps[0] = item.LastModified

			for i := 1; i < 5; i++ {
				time.Sleep(20 * time.Millisecond)
				err = store.Set("key", i)
				Expect(err).NotTo(HaveOccurred())

				item, err = store.Get("key")
				Expect(err).NotTo(HaveOccurred())
				timestamps[i] = item.LastModified

				Expect(timestamps[i]).To(BeTemporally(">", timestamps[i-1]))
			}
		})
	})

	Describe("Memory Management", func() {
		It("should not leak memory with Clear operations", func() {
			// Add many items
			for i := 0; i < 100; i++ {
				err := store.Set(string(rune('A'+i%26))+string(rune('0'+i/26)), i)
				Expect(err).NotTo(HaveOccurred())
			}

			size, err := store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(BeNumerically(">", 0))

			// Clear
			err = store.Clear()
			Expect(err).NotTo(HaveOccurred())

			size, err = store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(0))

			// Verify can add again
			err = store.Set("after-clear", "value")
			Expect(err).NotTo(HaveOccurred())

			size, err = store.Size()
			Expect(err).NotTo(HaveOccurred())
			Expect(size).To(Equal(1))
		})
	})
})

// === Shutdown Suite ===
var _ = Describe("LastModifiedStore Shutdown", func() {
	var (
		store  *LastModifiedStore
		ctx    context.Context
		cancel context.CancelFunc
		config Config
	)

	BeforeEach(func() {
		ctx, cancel = context.WithCancel(context.Background())
		config = Config{
			MaxSize:         0,
			WorkerCount:     2,
			CleanupInterval: 100 * time.Millisecond,
		}
	})

	AfterEach(func() {
		if cancel != nil {
			cancel()
		}
	})

	Describe("Graceful Shutdown", func() {
		It("should shutdown when not running without error", func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())

			// Note: Should be able to call Shutdown even if not started
			// Implementation should handle this gracefully
			err = store.Shutdown()
			// Depending on implementation, this might return ErrNotRunning
			// or succeed as a no-op. Document expected behavior.
		})

		It("should shutdown cleanly after start", func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())

			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = store.Shutdown()
			Expect(err).NotTo(HaveOccurred())
			Expect(store.IsRunning()).To(BeFalse())
		})

		It("should be idempotent", func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())

			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = store.Shutdown()
			Expect(err).NotTo(HaveOccurred())

			// Second shutdown should not panic or cause issues
			err = store.Shutdown()
			// Should either succeed as no-op or return ErrNotRunning
			// Document expected behavior
		})

		It("should stop accepting new operations after shutdown", func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())

			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			err = store.Shutdown()
			Expect(err).NotTo(HaveOccurred())

			// Operations should fail
			err = store.Set("key", "value")
			Expect(err).To(HaveOccurred())
			Expect(err).To(MatchError(ErrNotRunning))
		})
	})

	Describe("Shutdown with Active Operations", func() {
		It("should wait for in-flight operations", func() {
			var err error
			store, err = NewLastModifiedStore(config)
			Expect(err).NotTo(HaveOccurred())

			err = store.Start(ctx)
			Expect(err).NotTo(HaveOccurred())

			// Start long-running operations
			var wg sync.WaitGroup
			operationComplete := make(chan bool, 10)

			for i := 0; i < 10; i++ {
				wg.Add(1)
				go func(idx int) {
					defer wg.Done()
					// Simulate some work
					time.Sleep(50 * time.Millisecond)
					err := store.Set(string(rune('A'+idx)), idx)
					if err == nil {
						operationComplete <- true
					} else {
						operationComplete <- false
					}
				}(i)
			}

			// Give operations time to start
			time.Sleep(10 * time.Millisecond)

			// Shutdown should wait
			shutdownComplete := make(chan bool)
			go func() {
				_ = store.Shutdown()
				shutdownComplete <- true
			}()

			// Wait for operations
			wg.Wait()

			// Shutdown should complete
			Eventually(shutdownComplete, 2*time.Second).Should(Receive(BeTrue()))
			close(operationComplete)

			// At least some operations should have completed
			successCount := 0
			for success := range operationComplete {
				if success {
					successCount++
				}
			}
			Expect(successCount).To(BeNumerically(">", 0))
		})
	})
})
