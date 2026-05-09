package service

import (
	"sync"
	"time"

	"github.com/bluele/gcache"
)

type RecomputeState int32

const (
	RecomputeStatePending RecomputeState = iota
	RecomputeStateRunning
	RecomputeStateDone
	RecomputeStateFailed
)

type RecomputeJobError struct {
	ObjectId  string
	ErrorText string
}

type RecomputeJobState struct {
	JobId          string
	State          RecomputeState
	Processed      int32
	Total          int32
	Errors         []RecomputeJobError
	ProcessedIds   []string
	Mu             sync.RWMutex
	StateCondition *sync.Cond
}

type RecomputeJobStorage interface {
	Create(jobId string) *RecomputeJobState
	Get(jobId string) (*RecomputeJobState, bool)
}

type inMemoryRecomputeJobStorage struct {
	cache gcache.Cache
}

func (s *inMemoryRecomputeJobStorage) Create(jobId string) *RecomputeJobState {
	state := &RecomputeJobState{
		JobId:  jobId,
		State:  RecomputeStatePending,
		Errors: make([]RecomputeJobError, 0),
	}
	state.StateCondition = sync.NewCond(&state.Mu)
	_ = s.cache.Set(jobId, state)
	return state
}

func (s *inMemoryRecomputeJobStorage) Get(jobId string) (*RecomputeJobState, bool) {
	val, err := s.cache.Get(jobId)
	if err != nil {
		return nil, false
	}
	return val.(*RecomputeJobState), true
}

func NewInMemoryRecomputeJobStorage() RecomputeJobStorage {
	cache := gcache.New(256).LFU().Expiration(1 * time.Hour).Build()
	return &inMemoryRecomputeJobStorage{cache: cache}
}
