package gapi

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	speech "cloud.google.com/go/speech/apiv2"
	"cloud.google.com/go/speech/apiv2/speechpb"
	"github.com/weoses/memelo/common/temp"

	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"google.golang.org/api/option"
)

type GcloudAudio2TextExtractorImpl struct {
	client        *speech.Client
	languageCodes []string
	recognizer    string
	model         string
	slogger       *slog.Logger
}

func (g *GcloudAudio2TextExtractorImpl) Transcript(ctx context.Context, audio temp.Data) (string, error) {
	g.slogger.InfoContext(ctx, "Transcript start")
	audioBytes, err := audio.ReadAll()
	if err != nil {
		return "", fmt.Errorf("transcript: read audio: %w", err)
	}

	resp, err := g.client.Recognize(ctx, &speechpb.RecognizeRequest{
		Recognizer: g.recognizer,
		Config: &speechpb.RecognitionConfig{
			DecodingConfig: &speechpb.RecognitionConfig_AutoDecodingConfig{},
			LanguageCodes:  g.languageCodes,
			Model:          g.model,
		},
		AudioSource: &speechpb.RecognizeRequest_Content{
			Content: audioBytes,
		},
	})
	if err != nil {
		return "", fmt.Errorf("transcript: recognize failed: %w", err)
	}

	var parts []string
	for _, result := range resp.Results {
		if len(result.Alternatives) > 0 {
			parts = append(parts, result.Alternatives[0].Transcript)
		}
	}
	result := strings.Join(parts, "\n")
	g.slogger.InfoContext(ctx, "Transcript done", "chars", len(result))
	return result, nil
}

func NewAudio2TextExtractor(cfg *conf.Config) (ocr.Audio2TextExtractor, error) {
	sttConfig := cfg.AudioStt
	client, err := speech.NewClient(
		context.Background(),
		option.WithEndpoint(sttConfig.ApiEndpoint),
	)
	recognizer := fmt.Sprintf("projects/%s/locations/%s/recognizers/%s", sttConfig.ProjectName, sttConfig.ApiLocation, sttConfig.Recognizer)

	if err != nil {
		return nil, fmt.Errorf("NewAudio2TextExtractor: create client: %w", err)
	}
	return &GcloudAudio2TextExtractorImpl{
		client:        client,
		languageCodes: sttConfig.LanguageCodes,
		recognizer:    recognizer,
		model:         sttConfig.Model,
		slogger:       slog.With("service", "GcloudAudio2TextExtractor"),
	}, nil
}
