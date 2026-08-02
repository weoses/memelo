package ffmpeg

import (
	"context"
	"strings"
	"testing"

	"github.com/weoses/memelo/ffmpeg-service/conf"
)

func TestBuildCmd(t *testing.T) {
	tests := []struct {
		name     string
		cfg      *conf.FfmpegConfig
		args     []string
		wantPath string
		wantArgs []string
	}{
		{
			name:     "defaults to ffmpeg binary, no flags",
			cfg:      &conf.FfmpegConfig{},
			args:     []string{"-i", "in.mp4"},
			wantPath: "ffmpeg",
			wantArgs: []string{"-i", "in.mp4"},
		},
		{
			name:     "custom binary",
			cfg:      &conf.FfmpegConfig{FfmpegBinary: "/usr/bin/ffmpeg"},
			args:     []string{"-i", "in.mp4"},
			wantPath: "/usr/bin/ffmpeg",
			wantArgs: []string{"-i", "in.mp4"},
		},
		{
			name:     "threads limit prepends -threads",
			cfg:      &conf.FfmpegConfig{ThreadsLimit: 2},
			args:     []string{"-i", "in.mp4"},
			wantPath: "ffmpeg",
			wantArgs: []string{"-threads", "2", "-i", "in.mp4"},
		},
		{
			name:     "cpu limit wraps with cpulimit",
			cfg:      &conf.FfmpegConfig{CpuLimit: 50},
			args:     []string{"-i", "in.mp4"},
			wantPath: "cpulimit",
			wantArgs: []string{"-l", "50", "--", "ffmpeg", "-i", "in.mp4"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := buildCmd(context.Background(), tt.cfg, tt.args...)
			if !strings.HasSuffix(cmd.Path, tt.wantPath) && cmd.Args[0] != tt.wantPath {
				t.Errorf("path = %q, want suffix/arg0 %q (args=%v)", cmd.Path, tt.wantPath, cmd.Args)
			}
			gotArgs := cmd.Args[1:]
			if len(gotArgs) != len(tt.wantArgs) {
				t.Fatalf("args = %v, want %v", gotArgs, tt.wantArgs)
			}
			for i := range gotArgs {
				if gotArgs[i] != tt.wantArgs[i] {
					t.Errorf("args[%d] = %q, want %q (full: %v)", i, gotArgs[i], tt.wantArgs[i], gotArgs)
				}
			}
		})
	}
}
