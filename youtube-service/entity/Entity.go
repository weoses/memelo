package entity

import (
	"sync"
	"time"

	"github.com/weoses/memelo/common/temp"
)

type DownloadRequest struct {
	YoutubeURL string
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
	JobId     string
	State     JobState
	Result    *VideoDownloadResult
	Error     error
	CreatedAt time.Time
	Mu        sync.RWMutex
}
