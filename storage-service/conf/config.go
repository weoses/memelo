package conf

import (
	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
)

type MetadataStorageConfig struct {
	Elastic                *elasticsearch8.Config
	Index                  string
	EmbeddingV1Dimensions  int
	EmbeddingMatchTreshold float64
}

type ElasticTagConfig struct {
	Elastic               *elasticsearch8.Config
	Index                 string
	EmbeddingV1Dimensions int
}

type ImageS3StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
	Secure    bool
}

type ImageStorageConfig struct {
	S3 *ImageS3StorageConfig
}

type ImageConverterConfig struct {
	OriginalMaxSize int
	ThumbSize       int
}

type ImageEmbeddingConfig struct {
	ApiEndpoint string
	ApiLocation string
	ProjectName string
	Dimension   int
	Model       string
}

type ImageOcrConfig struct {
	ApiEndpoint string
}

func NewImageConverterConfig() (*ImageConverterConfig, error) {
	conf := new(ImageConverterConfig)
	err := viper.UnmarshalKey("image-converter", conf)
	return conf, err
}

func NewImageEmbeddingConfig() (*ImageEmbeddingConfig, error) {
	conf := new(ImageEmbeddingConfig)
	err := viper.UnmarshalKey("image-embedding", conf)
	return conf, err
}

func NewMetadataStorageConfig() (*MetadataStorageConfig, error) {
	conf := &MetadataStorageConfig{}
	err := viper.UnmarshalKey("metadata-storage", conf)
	return conf, err
}

func NewImageStorageConfig() (*ImageStorageConfig, error) {
	conf := &ImageStorageConfig{}
	err := viper.UnmarshalKey("image-storage", conf)
	return conf, err
}

func NewImageOcrConfig() (*ImageOcrConfig, error) {
	conf := &ImageOcrConfig{}
	err := viper.UnmarshalKey("image-ocr", conf)
	return conf, err
}

func NewElasticTagConfig() (*ElasticTagConfig, error) {
	conf := &ElasticTagConfig{}
	err := viper.UnmarshalKey("tag-storage", conf)
	return conf, err
}
