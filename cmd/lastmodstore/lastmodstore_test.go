package lastmodstore_test

import (
	"context"
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
		store *LastModifiedStore
		//ctx    context.Context
		cancel context.CancelFunc
		config Config
	)
	BeforeEach(func() {
		//ctx, cancel = context.WithCancel(context.Background())
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
})
