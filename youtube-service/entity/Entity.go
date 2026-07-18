package entity

import (
	"sync"
	"time"

	"github.com/google/uuid"
)

type DownloadRequest struct {
	YoutubeURL string
	AccountId  uuid.UUID
}

type VideoCreateResult struct {
	Id              uuid.UUID
	DuplicateStatus string
	Caption         string
	Tags            []string
	OcrResult       string
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
	Result    *VideoCreateResult
	Error     error
	CreatedAt time.Time
	Mu        sync.RWMutex
}
