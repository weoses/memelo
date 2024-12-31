package conf

import (
	"fmt"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/spf13/viper"
)

type MetadataStorageConfig struct {
	Elastic *elasticsearch8.Config
	Index   string
}
type ImageStorageConfig struct {
	Folder string
}

type OcrConfig struct {
	Url string
}

func InitConfig() {
	viper.SetConfigName("config")
	viper.SetConfigType("yaml")
	viper.AddConfigPath("$APPLICATION_CONFIGPATH")
	viper.AddConfigPath("$HOME/.appname")
	viper.AddConfigPath(".")
	err := viper.ReadInConfig()
	if err != nil {
		panic(fmt.Errorf("fatal error config file: %w", err))
	}
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
	err := viper.UnmarshalKey("storage", conf)
	return conf, err
}
