package functional_test

import (
	"context"
	"sync"
	"sync/atomic"

	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
)

const mockEmbeddingDimensions = 1408

type MockLlmExtractor struct {
	mu                      sync.Mutex
	result                  *ocr.MediaExtractResult
	callCount               atomic.Int32
	checkDuplicateResult    atomic.Bool
	checkDuplicateCallCount atomic.Int32
}

func (m *MockLlmExtractor) ProcessAudio(_ context.Context, _ temp.Data) (*ocr.MediaExtractResult, error) {
	m.callCount.Add(1)
	return m.get(), nil
}

func (m *MockLlmExtractor) SetResult(r *ocr.MediaExtractResult) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.result = r
}

func (m *MockLlmExtractor) CallCount() int32 { return m.callCount.Load() }

func (m *MockLlmExtractor) SetCheckDuplicateResult(v bool) { m.checkDuplicateResult.Store(v) }
func (m *MockLlmExtractor) CheckDuplicateCallCount() int32 {
	return m.checkDuplicateCallCount.Load()
}

func (m *MockLlmExtractor) Reset() {
	m.callCount.Store(0)
	m.checkDuplicateCallCount.Store(0)
	m.checkDuplicateResult.Store(false)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.result = nil
}

func (m *MockLlmExtractor) get() *ocr.MediaExtractResult {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.result != nil {
		return m.result
	}
	return &ocr.MediaExtractResult{Caption: "mock caption"}
}

func (m *MockLlmExtractor) ProcessImage(_ context.Context, _ temp.Data) (*ocr.MediaExtractResult, error) {
	m.callCount.Add(1)
	return m.get(), nil
}

func (m *MockLlmExtractor) ProcessVideo(_ context.Context, _ temp.Data) (*ocr.MediaExtractResult, error) {
	m.callCount.Add(1)
	return m.get(), nil
}

func (m *MockLlmExtractor) CombineResults(_ context.Context, _ []ocr.MediaExtractResult) (*ocr.MediaExtractResult, error) {
	return m.get(), nil
}

func (m *MockLlmExtractor) CheckDuplicate(_ context.Context, _ temp.Data, _ temp.Data) (bool, error) {
	m.checkDuplicateCallCount.Add(1)
	return m.checkDuplicateResult.Load(), nil
}

type MockEmbeddingExtractor struct {
	mu        sync.Mutex
	static    []float32    // nil = unique mode
	counter   atomic.Int32 // incremented per generateEmbedding call
	callCount atomic.Int32

	videoEmbeddingsPerCall int                      // embeddings returned per GetVideoEmbedding call (default 1)
	videoEmbeddingQueue    [][]entity.EmbeddingItem // when non-empty, dequeued one batch per call
}

// UseUniqueEmbedding switches to unique-per-call mode (default after Reset).
func (m *MockEmbeddingExtractor) UseUniqueEmbedding() {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.static = nil
}

// UseStaticEmbedding makes every call return the same vector.
func (m *MockEmbeddingExtractor) UseStaticEmbedding(v []float32) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.static = v
}

func (m *MockEmbeddingExtractor) CallCount() int32 { return m.callCount.Load() }

// SetVideoEmbeddingsPerCall configures how many embeddings GetVideoEmbedding returns per call
// when no queue entry is available. Default is 1.
func (m *MockEmbeddingExtractor) SetVideoEmbeddingsPerCall(n int) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.videoEmbeddingsPerCall = n
}

// QueueVideoEmbeddings enqueues a batch to be returned by the next GetVideoEmbedding call.
// Queued batches take priority over SetVideoEmbeddingsPerCall / UseStaticEmbedding.
func (m *MockEmbeddingExtractor) QueueVideoEmbeddings(items []entity.EmbeddingItem) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.videoEmbeddingQueue = append(m.videoEmbeddingQueue, items)
}

// Reset clears state and returns to unique-per-call mode.
func (m *MockEmbeddingExtractor) Reset() {
	m.counter.Store(0)
	m.callCount.Store(0)
	m.mu.Lock()
	defer m.mu.Unlock()
	m.static = nil
	m.videoEmbeddingsPerCall = 0
	m.videoEmbeddingQueue = nil
}

// generateEmbedding returns a one-hot vector at a position derived from an
// internal counter, ensuring each call produces a distinct embedding.
func (m *MockEmbeddingExtractor) generateEmbedding() []float32 {
	pos := int(m.counter.Add(1)-1) % mockEmbeddingDimensions
	v := make([]float32, mockEmbeddingDimensions)
	v[pos] = 1.0
	return v
}

func (m *MockEmbeddingExtractor) get() []float32 {
	m.mu.Lock()
	s := m.static
	m.mu.Unlock()
	if s != nil {
		return s
	}
	return m.generateEmbedding()
}

func (m *MockEmbeddingExtractor) GetImageEmbedding(_ context.Context, _ temp.Data) (*entity.EmbeddingItem, error) {
	m.callCount.Add(1)
	return &entity.EmbeddingItem{
		Data:  m.get(),
		Model: "mock-model",
		Type:  entity.EmbeddingTypeImage,
	}, nil
}

func (m *MockEmbeddingExtractor) GetTextEmbedding(_ context.Context, _ string) (*entity.EmbeddingItem, error) {
	m.callCount.Add(1)
	return &entity.EmbeddingItem{
		Data:  m.get(),
		Model: "mock-model",
		Type:  entity.EmbeddingTypeText,
	}, nil
}

func (m *MockEmbeddingExtractor) GetVideoEmbedding(_ context.Context, _ temp.Data) ([]entity.EmbeddingItem, error) {
	m.callCount.Add(1)

	m.mu.Lock()
	if len(m.videoEmbeddingQueue) > 0 {
		batch := m.videoEmbeddingQueue[0]
		m.videoEmbeddingQueue = m.videoEmbeddingQueue[1:]
		m.mu.Unlock()
		return batch, nil
	}
	n := m.videoEmbeddingsPerCall
	m.mu.Unlock()

	if n <= 0 {
		n = 1
	}
	items := make([]entity.EmbeddingItem, n)
	for i := range n {
		items[i] = entity.EmbeddingItem{
			Data:  m.get(),
			Model: "mock-model",
			Type:  entity.EmbeddingTypeVideo,
		}
	}
	return items, nil
}
