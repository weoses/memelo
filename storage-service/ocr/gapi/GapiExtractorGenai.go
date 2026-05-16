package gapi

import (
	"context"
	"fmt"
	"log/slog"
	"strings"

	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"golang.org/x/time/rate"
	"google.golang.org/genai"
)

const onScreenTextProp = "on_screen_text"
const audioTranscriptProp = "audio_transcript"
const audioTrackProp = "audio_track"
const captionProp = "caption"

type GeminiExtractor struct {
	client  *genai.Client
	cfg     *conf.GeminiExtractorConfig
	limiter *rate.Limiter
	slogger *slog.Logger
}

func NewGeminiExtractor(cfg *conf.Config) (ocr.LlmMediaExtractor, error) {
	c := cfg.GeminiExtractor
	clientCfg := &genai.ClientConfig{}
	if c.ApiKey != "" {
		clientCfg.APIKey = c.ApiKey
		clientCfg.Backend = genai.BackendGeminiAPI
	}
	if c.ApiEndpoint != "" {
		clientCfg.HTTPOptions = genai.HTTPOptions{BaseURL: c.ApiEndpoint}
	}
	client, err := genai.NewClient(context.Background(), clientCfg)
	if err != nil {
		return nil, fmt.Errorf("gemini client: %w", err)
	}

	logger := slog.With("service", "GeminiExtractor")

	return &GeminiExtractor{
		client:  client,
		cfg:     c,
		limiter: rate.NewLimiter(0.5, 1),
		slogger: logger,
	}, nil
}

func (g *GeminiExtractor) buildTool() *genai.Tool {
	c := g.cfg

	return &genai.Tool{
		FunctionDeclarations: []*genai.FunctionDeclaration{{
			Name:        "extract_metadata",
			Description: c.OutputToolDescription,
			Parameters: &genai.Schema{
				Type: genai.TypeObject,
				Properties: map[string]*genai.Schema{
					onScreenTextProp:    {Type: genai.TypeString, Description: c.OutputToolOnScreenTextDesc},
					audioTranscriptProp: {Type: genai.TypeString, Description: c.OutputToolTranscriptionDesc},
					audioTrackProp:      {Type: genai.TypeString, Description: c.OutputToolAudioTrackDesc},
					captionProp:         {Type: genai.TypeString, Description: c.OutputToolCaptionDesc},
				},
				Required:         []string{captionProp},
				PropertyOrdering: []string{onScreenTextProp, audioTranscriptProp, audioTrackProp, captionProp},
			},
		}},
	}
}

func (g *GeminiExtractor) buildMediaPart(ctx context.Context, data temp.Data, mimeType string) (*genai.Part, error) {
	if s3data, ok := data.(temp.S3BackedData); ok && s3data.IsGsSupported() {
		if url, err := s3data.GetPresignedUrl(ctx); err == nil {
			g.slogger.InfoContext(ctx, "buildMediaPart: using signed url", "uri", url)
			return genai.NewPartFromURI(url, mimeType), nil
		}
	}

	g.slogger.InfoContext(ctx, "buildMediaPart: uploading via Files API")
	reader, err := data.Reader()
	if err != nil {
		return nil, fmt.Errorf("read data: %w", err)
	}
	defer helper.QuietClose(reader, g.slogger)

	file, err := g.client.Files.Upload(ctx, reader, &genai.UploadFileConfig{MIMEType: mimeType})
	if err != nil {
		return nil, fmt.Errorf("upload file: %w", err)
	}

	return genai.NewPartFromURI(file.URI, mimeType), nil
}

func (g *GeminiExtractor) ProcessImage(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	return g.process(ctx, data, "image/jpeg")
}

func (g *GeminiExtractor) ProcessVideo(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	return g.process(ctx, data, "video/mp4")
}

func (g *GeminiExtractor) process(ctx context.Context, data temp.Data, mimeType string) (*ocr.MediaExtractResult, error) {
	g.slogger.InfoContext(ctx, "process start", "mimeType", mimeType)

	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	g.limiter.Allow()

	mediaPart, err := g.buildMediaPart(ctx, data, mimeType)
	if err != nil {
		return nil, err
	}

	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{
			{Text: g.cfg.Prompt},
			mediaPart,
		}, genai.RoleUser),
	}

	const toolName = "extract_metadata"
	resp, err := g.client.Models.GenerateContent(
		ctx,
		g.cfg.Model,
		contents,
		&genai.GenerateContentConfig{
			Tools: []*genai.Tool{g.buildTool()},
			ToolConfig: &genai.ToolConfig{
				FunctionCallingConfig: &genai.FunctionCallingConfig{
					Mode:                 genai.FunctionCallingConfigModeAny,
					AllowedFunctionNames: []string{toolName},
				},
			},
		},
	)
	if err != nil {
		return nil, fmt.Errorf("generate content: %w", err)
	}

	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.FunctionCall != nil && part.FunctionCall.Name == toolName {
				args := part.FunctionCall.Args
				g.slogger.InfoContext(ctx, "process done")
				return &ocr.MediaExtractResult{
					OnScreenText:    stringArg(args, onScreenTextProp),
					AudioTranscript: stringArg(args, audioTranscriptProp),
					AudioTrack:      stringArg(args, audioTrackProp),
					Caption:         stringArg(args, captionProp),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("model did not return %s call", toolName)
}

func (g *GeminiExtractor) CombineResults(ctx context.Context, results []ocr.MediaExtractResult) (*ocr.MediaExtractResult, error) {
	if err := g.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(g.cfg.CombinePrompt)
	sb.WriteString("\n\n")
	for i, r := range results {
		_, err := fmt.Fprintf(&sb, "Segment %d:\non_screen_text: %s\naudio_transcript: %s\naudio_track: %s\ncaption: %s\n\n",
			i+1, r.OnScreenText, r.AudioTranscript, r.AudioTrack, r.Caption)
		if err != nil {
			return nil, fmt.Errorf("failed to fprintf prompt: %w", err)
		}
	}

	g.slogger.DebugContext(ctx, "combine prompt", "prompt", sb.String())
	contents := []*genai.Content{
		genai.NewContentFromParts([]*genai.Part{{Text: sb.String()}}, genai.RoleUser),
	}

	const toolName = "extract_metadata"
	resp, err := g.client.Models.GenerateContent(ctx, g.cfg.Model, contents, &genai.GenerateContentConfig{
		Tools: []*genai.Tool{g.buildTool()},
		ToolConfig: &genai.ToolConfig{
			FunctionCallingConfig: &genai.FunctionCallingConfig{
				Mode:                 genai.FunctionCallingConfigModeAny,
				AllowedFunctionNames: []string{toolName},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("combine results generate content: %w", err)
	}

	for _, cand := range resp.Candidates {
		for _, part := range cand.Content.Parts {
			if part.FunctionCall != nil && part.FunctionCall.Name == toolName {
				args := part.FunctionCall.Args
				g.slogger.InfoContext(ctx, "combine results done")
				return &ocr.MediaExtractResult{
					OnScreenText:    stringArg(args, onScreenTextProp),
					AudioTranscript: stringArg(args, audioTranscriptProp),
					AudioTrack:      stringArg(args, audioTrackProp),
					Caption:         stringArg(args, captionProp),
				}, nil
			}
		}
	}
	return nil, fmt.Errorf("model did not return %s call during combine", toolName)
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
