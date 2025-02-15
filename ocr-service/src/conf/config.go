package conf

import "github.com/spf13/viper"

type ImageConverterConfig struct {
	ThumbSize int
}

type ImageEmbeddingConfig struct {
	ApiLocation string
	ProjectName string
	Dimension   int
	Model       string
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
