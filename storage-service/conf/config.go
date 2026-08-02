package conf

import (
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

type CommonExtractingConfig struct {
	EmbeddingDimensions   int `mapstructure:"embedding-dimensions"`
	VideoSliceIntervalSec int `mapstructure:"video-slice-interval-sec"`
}

type SearchConfig struct {
	SemanticDuplicateThreshold        float64
	PercentageDuplicatePartsThreshold float64
	SemanticTextSearchThreshold       float64
	Fuzziness                         string
}

type MetadataDbConfig struct {
	Elastic                *elasticsearch8.Config
	Index                  string
	EmbeddingMatchTreshold float64
}

type TagDbConfig struct {
	Elastic *elasticsearch8.Config
	Index   string
}

type ImageConverterConfig struct {
	OriginalMaxSize int
	ThumbSize       int
}

type MediaEmbedderConfig struct {
	ApiEndpoint string
	ApiLocation string
	ProjectName string
	Model       string
}

type ImageOcrConfig struct {
	ApiEndpoint string
}

type AudioSttConfig struct {
	ApiEndpoint   string
	ApiLocation   string
	ProjectName   string
	Recognizer    string
	Model         string
	LanguageCodes []string
}

type FfmpegServiceConfig struct {
	Uri            string
	PollIntervalMs int
	PollMaxWaitSec int
}

type GeminiExtractorConfig struct {
	ApiKey                      string `mapstructure:"apikey"`
	ApiEndpoint                 string `mapstructure:"apiendpoint"`
	ModelAudio                  string `mapstructure:"model-audio"`
	ModelVideo                  string `mapstructure:"model-video"`
	ModelImage                  string `mapstructure:"model-image"`
	ImageExtractPrompt          string `mapstructure:"image-extract-prompt"`
	VideoExtractPrompt          string `mapstructure:"video-extract-prompt"`
	AudioExtractPrompt          string `mapstructure:"audio-extract-prompt"`
	CombinePrompt               string `mapstructure:"combine-prompt"`
	DuplicatePrompt             string `mapstructure:"duplicate-prompt"`
	OutputToolDescription       string `mapstructure:"output-tool-description"`
	OutputToolTranscriptionDesc string `mapstructure:"output-tool-transcription-desc"`
	OutputToolOnScreenTextDesc  string `mapstructure:"output-tool-on-screen-text-desc"`
	OutputToolAudioTrackDesc    string `mapstructure:"output-tool-audio-track-desc"`
	OutputToolCaptionDesc       string `mapstructure:"output-tool-caption-desc"`
}

type GeminiEmbeddingConfig struct {
	ApiKey      string `mapstructure:"apikey"`
	ApiEndpoint string `mapstructure:"apiendpoint"`
	Model       string `mapstructure:"model"`
}

type OpenRouterExtractorConfig struct {
	ApiKey                      string `mapstructure:"apikey"`
	ModelAudio                  string `mapstructure:"model-audio"`
	ModelVideo                  string `mapstructure:"model-video"`
	ModelImage                  string `mapstructure:"model-image"`
	ImageExtractPrompt          string `mapstructure:"image-extract-prompt"`
	VideoExtractPrompt          string `mapstructure:"video-extract-prompt"`
	AudioExtractPrompt          string `mapstructure:"audio-extract-prompt"`
	CombinePrompt               string `mapstructure:"combine-prompt"`
	DuplicatePrompt             string `mapstructure:"duplicate-prompt"`
	OutputToolDescription       string `mapstructure:"output-tool-description"`
	OutputToolTranscriptionDesc string `mapstructure:"output-tool-transcription-desc"`
	OutputToolOnScreenTextDesc  string `mapstructure:"output-tool-on-screen-text-desc"`
	OutputToolAudioTrackDesc    string `mapstructure:"output-tool-audio-track-desc"`
	OutputToolCaptionDesc       string `mapstructure:"output-tool-caption-desc"`
}

type OpenRouterEmbeddingConfig struct {
	ApiKey string `mapstructure:"apikey"`
	Model  string `mapstructure:"model"`
}

type Config struct {
	Server              *commonconfig.ServerConfig       `mapstructure:"server"`
	Log                 *commonconfig.LoggingConfig      `mapstructure:"log"`
	Search              *SearchConfig                    `mapstructure:"search"`
	Extracting          *CommonExtractingConfig          `mapstructure:"extracting"`
	MediaStorage        *commonconfig.MediaStorageConfig `mapstructure:"media-storage"`
	TempStorage         *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
	MetadataDb          *MetadataDbConfig                `mapstructure:"metadata-db"`
	TagDb               *TagDbConfig                     `mapstructure:"tag-db"`
	ImageConverter      *ImageConverterConfig            `mapstructure:"image-converter"`
	FfmpegService       *FfmpegServiceConfig             `mapstructure:"ffmpeg-service"`
	GeminiExtractor     *GeminiExtractorConfig           `mapstructure:"gemini-extractor"`
	GeminiEmbedding     *GeminiEmbeddingConfig           `mapstructure:"gemini-embedding"`
	ExtractorProvider   string                           `mapstructure:"extractor-provider"`
	EmbedderProvider    string                           `mapstructure:"embedder-provider"`
	OpenRouterExtractor *OpenRouterExtractorConfig       `mapstructure:"openrouter-extractor"`
	OpenRouterEmbedding *OpenRouterEmbeddingConfig       `mapstructure:"openrouter-embedding"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
