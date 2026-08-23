package entity

import (
	"context"
	"sync"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type DownloadRequest struct {
	YoutubeURL       string
	RetentionSeconds int32
}

type VideoDownloadResult struct {
	S3Path   string
	MimeType string
	S3Data   temp.S3BackedData
}

type JobState int32

const (
	JobStatePending JobState = 1
	JobStateRunning JobState = 2
	JobStateDone    JobState = 3
	JobStateFailed  JobState = 4
)

type DownloadJob struct {
	JobId        string
	State        JobState
	Result       *VideoDownloadResult
	Error        error
	CreatedAt    time.Time
	EffectiveTTL time.Duration // negative means never expire
	Mu           sync.RWMutex

	// Ctx is a detached (context.WithoutCancel), values-only context captured
	// once at job creation, deliberately stored here as a narrow exception to
	// the usual "don't store context.Context on a struct" guideline: it's
	// read only by startTTLCleaner's background sweep, long after the
	// request that created this job is gone, and never carries cancellation/
	// deadline or is used for live request work. Reusing it there beats
	// fabricating a fresh, disconnected context.Background() with nothing to
	// tie back to the job's own origin.
	Ctx context.Context
}
