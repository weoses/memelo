package ffmpeg

import (
	"context"
	"os/exec"
	"strconv"

	"github.com/weoses/memelo/storage-service/conf"
)

// buildCmd constructs an exec.Cmd for ffmpeg, applying optional thread limit
// and optionally wrapping the invocation with cpulimit.
func buildCmd(ctx context.Context, cfg *conf.FfmpegConfig, args ...string) *exec.Cmd {
	if cfg.ThreadsLimit > 0 {
		args = append([]string{"-threads", strconv.Itoa(cfg.ThreadsLimit)}, args...)
	}

	binary := cfg.Binary
	if binary == "" {
		binary = "ffmpeg"
	}

	if cfg.CpuLimit > 0 {
		cpuArgs := append([]string{"-l", strconv.Itoa(cfg.CpuLimit), "--", binary}, args...)
		return exec.CommandContext(ctx, "cpulimit", cpuArgs...)
	}

	return exec.CommandContext(ctx, binary, args...)
}
