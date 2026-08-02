package conf

import (
	"fmt"

	"github.com/spf13/viper"
	commonconfig "github.com/weoses/memelo/common/config"
)

type FfmpegConfig struct {
	FfmpegBinary  string
	FfprobeBinary string
	CpuLimit      int
	ThreadsLimit  int
}

type JobConfig struct {
	MaxConcurrentJobs int
	JobTtlSeconds     int
}

type Config struct {
	Server      *commonconfig.ServerConfig       `mapstructure:"server"`
	Log         *commonconfig.LoggingConfig      `mapstructure:"log"`
	TempStorage *commonconfig.MediaStorageConfig `mapstructure:"temp-storage"`
	Ffmpeg      *FfmpegConfig                    `mapstructure:"ffmpeg"`
	Job         *JobConfig                       `mapstructure:"job"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}
	if err := viper.Unmarshal(cfg); err != nil {
		return nil, fmt.Errorf("error reading config: %w", err)
	}
	return cfg, nil
}
