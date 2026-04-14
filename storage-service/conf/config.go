package conf

import (
	"fmt"
	"time"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

type CommonEmbeddingsConfig struct {
	Dimensions                int
	VideoEmbeddingIntervalSec int
}

type SearchConfig struct {
	SemanticDuplicateThreshold  float64
	SemanticTextSearchThreshold float64
	Fuzziness                   string
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

type FfmpegConfig struct {
	Binary       string
	CpuLimit     int
	ThreadsLimit int
}

type GeminiExtractorConfig struct {
	ApiKey                      string `mapstructure:"apikey"`
	ApiEndpoint                 string `mapstructure:"apiendpoint"`
	Model                       string `mapstructure:"model"`
	Prompt                      string `mapstructure:"prompt"`
	OutputToolDescription       string `mapstructure:"output-tool-description"`
	OutputToolTranscriptionDesc string `mapstructure:"output-tool-transcription-desc"`
	OutputToolOnScreenTextDesc  string `mapstructure:"output-tool-on-screen-text-desc"`
	OutputToolAudioTrackDesc    string `mapstructure:"output-tool-audio-track-desc"`
	OutputToolCaptionDesc       string `mapstructure:"output-tool-caption-desc"`
}

type Config struct {
	Server            *commonconfig.ServerConfig       `mapstructure:"server"`
	Log               *commonconfig.LoggingConfig      `mapstructure:"log"`
	Search            *SearchConfig                    `mapstructure:"search"`
	Embeddings        *CommonEmbeddingsConfig          `mapstructure:"embeddings"`
	MediaStorage      *commonconfig.MediaStorageConfig `mapstructure:"media-storage"`
	TempStorage       *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
	TempStorageExpiry time.Duration                    `mapstructure:"temp-storage-expiry"`
	MetadataDb        *MetadataDbConfig                `mapstructure:"metadata-db"`
	TagDb             *TagDbConfig                     `mapstructure:"tag-db"`
	MediaEmbedding    *MediaEmbedderConfig             `mapstructure:"media-embedding"`
	ImageConverter    *ImageConverterConfig            `mapstructure:"image-converter"`
	ImageOcr          *ImageOcrConfig                  `mapstructure:"image-ocr"`
	AudioStt          *AudioSttConfig                  `mapstructure:"audio-stt"`
	Ffmpeg            *FfmpegConfig                    `mapstructure:"ffmpeg"`
	GeminiExtractor   *GeminiExtractorConfig           `mapstructure:"gemini-extractor"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
