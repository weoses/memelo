package functional_test

import (
	"context"
	"fmt"
	"net"
	"time"

	"github.com/weoses/memelo/storage-service/app"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"go.uber.org/fx"
)

func startTestServer(cfg *conf.Config, extractor ocr.LlmMediaExtractor, embedder ocr.LlmEmbeddingExtractor) (addr string, stop func(), err error) {
	ln, err := net.Listen("tcp", ":0")
	if err != nil {
		return "", nil, fmt.Errorf("listen: %w", err)
	}
	addr = fmt.Sprintf("http://localhost:%d", ln.Addr().(*net.TCPAddr).Port)

	fxapp := fx.New(
		fx.Supply(cfg),
		app.Module(),
		// Replace the real Gemini implementations and listener with test doubles.
		fx.Decorate(func() net.Listener { return ln }),
		fx.Decorate(func() ocr.LlmMediaExtractor { return extractor }),
		fx.Decorate(
			fx.Annotate(
				func() ocr.LlmEmbeddingExtractor { return embedder },
				fx.ResultTags(`name:"raw_embedder"`),
			),
		),
		// ffmpeg-service is a separate Go module and can't be reached from these
		// in-process tests, so swap the RPC-backed adapters for local-exec doubles.
		fx.Decorate(func() ocr.Video2Mp4Converter { return localFfmpegVideo2Mp4{} }),
		fx.Decorate(func() ocr.Video2FrameExtractor { return localFfmpegFrameExtractor{} }),
		fx.Decorate(func() ocr.VideoSlicer { return localFfmpegVideoSlicer{} }),
	)

	if err := fxapp.Start(context.Background()); err != nil {
		return "", nil, fmt.Errorf("start app: %w", err)
	}

	stop = func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer cancel()
		_ = fxapp.Stop(ctx)
	}
	return addr, stop, nil
}
