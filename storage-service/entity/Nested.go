package entity

const (
	EmbeddingTypeAudio = "audio"
	EmbeddingTypeVideo = "video"
	EmbeddingTypeImage = "image"
	EmbeddingTypeText  = "text"
)

type EmbeddingItem struct {
	Data      []float32 `validator:"required"`
	Model     string    `validator:"required"`
	TimeStart int
	TimeEnd   int
	Type      string `validator:"required"`
}

type Sizes struct {
	Width  int `validator:"required"`
	Height int `validator:"required"`
}

type Result struct {
	OnScreenText    string
	AudioTranscript string
	AudioTrack      string
	Caption         string
}

type ResultPerVideoSlice struct {
	SliceNumber    int
	SliceStartTime int
	SliceEndTime   int
	Result         Result
}
