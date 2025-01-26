package conf

import (
	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
)

type MetadataStorageConfig struct {
	Elastic *elasticsearch8.Config
	Index   string
}

type ImageS3StorageConfig struct {
	Endpoint  string
	AccessKey string
	SecretKey string
	Bucket    string
}

type ImageStorageConfig struct {
	S3 *ImageS3StorageConfig
}

type OcrConfig struct {
	Url string
}

func NewMetadataStorageConfig() (*MetadataStorageConfig, error) {
	conf := &MetadataStorageConfig{}
	err := viper.UnmarshalKey("metadata-storage", conf)
	return conf, err
}

func NewOcrConfig() (*OcrConfig, error) {
	conf := &OcrConfig{}
	err := viper.UnmarshalKey("ocr", conf)
	return conf, err
}

func NewImageStorageConfig() (*ImageStorageConfig, error) {
	conf := &ImageStorageConfig{}
	err := viper.UnmarshalKey("image-storage", conf)
	return conf, err
}
