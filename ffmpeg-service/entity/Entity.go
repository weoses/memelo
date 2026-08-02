package entity

import (
	"sync"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type ActionType int32

const (
	ActionExtractThumbnail ActionType = 1
	ActionConvertToMp4     ActionType = 2
	ActionSliceVideo       ActionType = 3
)

type JobRequest struct {
	InputS3Path      string
	RetentionSeconds int32
	Action           ActionType

	// slice_video only
	IntervalSeconds int32
	OverlapSeconds  int32
}

type SliceResult struct {
	S3Path    string
	StartTime int
	EndTime   int
}

type JobResult struct {
	// extract_thumbnail / convert_to_mp4
	OutputS3Path string
	// slice_video
	Slices []SliceResult

	// S3Data holds ownership of every S3-backed object this job produced,
	// so the job's TTL cleanup goroutine can close them all.
	S3Data []temp.S3BackedData
}

type JobState int32

const (
	JobStatePending JobState = 1
	JobStateRunning JobState = 2
	JobStateDone    JobState = 3
	JobStateFailed  JobState = 4
)

type Job struct {
	JobId        string
	Action       ActionType
	State        JobState
	Result       *JobResult
	Error        error
	CreatedAt    time.Time
	EffectiveTTL time.Duration // negative means never expire
	Mu           sync.RWMutex
}
