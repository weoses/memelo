package ocr

import (
	"context"

	"github.com/weoses/memelo/common/temp"
)

type MediaExtractResult struct {
	OnScreenText    string
	AudioTranscript string
	AudioTrack      string
	Caption         string
}

type LlmMediaExtractor interface {
	ProcessImage(ctx context.Context, data temp.Data) (*MediaExtractResult, error)
	ProcessVideo(ctx context.Context, data temp.Data) (*MediaExtractResult, error)
	CombineResults(ctx context.Context, results []MediaExtractResult) (*MediaExtractResult, error)
}
