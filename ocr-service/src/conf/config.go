package conf

import "github.com/spf13/viper"

type ImageConverterConfig struct {
	ThumbSize int
}

func NewImageConverterConfig() (*ImageConverterConfig, error) {
	conf := new(ImageConverterConfig)
	err := viper.UnmarshalKey("image-converter", conf)
	return conf, err
}
