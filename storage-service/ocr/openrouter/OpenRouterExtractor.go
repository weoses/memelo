package openrouter

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"strings"

	orsdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
	"golang.org/x/time/rate"
)

const orOnScreenTextProp = "on_screen_text"
const orAudioTranscriptProp = "audio_transcript"
const orAudioTrackProp = "audio_track"
const orCaptionProp = "caption"
const orToolName = "extract_metadata"

type OpenRouterExtractor struct {
	client       *orsdk.OpenRouter
	cfg          *conf.OpenRouterExtractorConfig
	frameExtract ocr.Video2FrameExtractor
	limiter      *rate.Limiter
	slogger      *slog.Logger
}

func NewOpenRouterExtractor(cfg *conf.Config, fe ocr.Video2FrameExtractor) (ocr.LlmMediaExtractor, error) {
	c := cfg.OpenRouterExtractor
	if c == nil {
		return nil, fmt.Errorf("openrouter-extractor config is missing")
	}
	client := orsdk.New(orsdk.WithSecurity(c.ApiKey))
	return &OpenRouterExtractor{
		client:       client,
		cfg:          c,
		frameExtract: fe,
		limiter:      rate.NewLimiter(2, 2),
		slogger:      slog.With("service", "OpenRouterExtractor"),
	}, nil
}

func (e *OpenRouterExtractor) buildTool() components.ChatFunctionTool {
	const fieldType = "type"
	const stringTyp = "string"
	const fieldDescription = "description"
	desc := e.cfg.OutputToolDescription
	return components.CreateChatFunctionToolChatFunctionToolFunction(components.ChatFunctionToolFunction{
		Type: components.ChatFunctionToolTypeFunction,
		Function: components.ChatFunctionToolFunctionFunction{
			Name:        orToolName,
			Description: &desc,
			Parameters: map[string]any{
				fieldType: "object",
				"properties": map[string]any{
					orOnScreenTextProp:    map[string]any{fieldType: stringTyp, fieldDescription: e.cfg.OutputToolOnScreenTextDesc},
					orAudioTranscriptProp: map[string]any{fieldType: stringTyp, fieldDescription: e.cfg.OutputToolTranscriptionDesc},
					orAudioTrackProp:      map[string]any{fieldType: stringTyp, fieldDescription: e.cfg.OutputToolAudioTrackDesc},
					orCaptionProp:         map[string]any{fieldType: stringTyp, fieldDescription: e.cfg.OutputToolCaptionDesc},
				},
				"required": []string{orCaptionProp},
			},
		},
	})
}

func (e *OpenRouterExtractor) buildImageMessage(ctx context.Context, data temp.Data) (components.ChatMessages, error) {
	dataURL, err := toDataURL(data, "image/jpeg", e.slogger)
	if err != nil {
		return components.ChatMessages{}, fmt.Errorf("build image message: %w", err)
	}

	content := components.CreateChatUserMessageContentArrayOfChatContentItems([]components.ChatContentItems{
		components.CreateChatContentItemsText(components.ChatContentText{
			Type: components.ChatContentTextTypeText,
			Text: e.cfg.Prompt,
		}),
		components.CreateChatContentItemsImageURL(components.ChatContentImage{
			Type:     components.ChatContentImageTypeImageURL,
			ImageURL: components.ChatContentImageImageURL{URL: dataURL},
		}),
	})
	return components.CreateChatMessagesUser(components.ChatUserMessage{Content: content}), nil
}

func (e *OpenRouterExtractor) sendChat(ctx context.Context, messages []components.ChatMessages) (*ocr.MediaExtractResult, error) {
	toolChoice := components.CreateChatToolChoiceChatNamedToolChoice(components.ChatNamedToolChoice{
		Type:     components.ChatNamedToolChoiceTypeFunction,
		Function: components.ChatNamedToolChoiceFunction{Name: orToolName},
	})

	resp, err := e.client.Chat.Send(ctx, components.ChatRequest{
		Model:      &e.cfg.Model,
		Messages:   messages,
		Tools:      []components.ChatFunctionTool{e.buildTool()},
		ToolChoice: &toolChoice,
	})
	if err != nil {
		return nil, fmt.Errorf("chat send: %w", err)
	}
	if resp.ChatResult == nil || len(resp.ChatResult.Choices) == 0 {
		return nil, fmt.Errorf("chat returned no choices")
	}

	toolCalls := resp.ChatResult.Choices[0].Message.ToolCalls
	if len(toolCalls) == 0 {
		return nil, fmt.Errorf("model did not return %s call", orToolName)
	}

	var args map[string]any
	if err := json.Unmarshal([]byte(toolCalls[0].Function.Arguments), &args); err != nil {
		return nil, fmt.Errorf("parse tool arguments: %w", err)
	}

	return &ocr.MediaExtractResult{
		OnScreenText:    stringArg(args, orOnScreenTextProp),
		AudioTranscript: stringArg(args, orAudioTranscriptProp),
		AudioTrack:      stringArg(args, orAudioTrackProp),
		Caption:         stringArg(args, orCaptionProp),
	}, nil
}

func (e *OpenRouterExtractor) ProcessImage(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "ProcessImage start")
	if err := e.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	msg, err := e.buildImageMessage(ctx, data)
	if err != nil {
		return nil, err
	}

	result, err := e.sendChat(ctx, []components.ChatMessages{msg})
	if err != nil {
		return nil, fmt.Errorf("ProcessImage: %w", err)
	}

	e.slogger.InfoContext(ctx, "ProcessImage done")
	return result, nil
}

func (e *OpenRouterExtractor) ProcessVideo(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "ProcessVideo start")

	frame, err := e.frameExtract.ExtractOneFrame(ctx, data)
	if err != nil {
		return nil, fmt.Errorf("ProcessVideo: extract frame: %w", err)
	}
	defer helper.QuietClose(frame, e.slogger)

	result, err := e.ProcessImage(ctx, frame)
	if err != nil {
		return nil, fmt.Errorf("ProcessVideo: %w", err)
	}

	e.slogger.InfoContext(ctx, "ProcessVideo done")
	return result, nil
}

func (e *OpenRouterExtractor) CombineResults(ctx context.Context, results []ocr.MediaExtractResult) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "CombineResults start")
	if err := e.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var sb strings.Builder
	sb.WriteString(e.cfg.CombinePrompt)
	sb.WriteString("\n\n")
	for i, r := range results {
		_, err := fmt.Fprintf(&sb, "Segment %d:\non_screen_text: %s\naudio_transcript: %s\naudio_track: %s\ncaption: %s\n\n",
			i+1, r.OnScreenText, r.AudioTranscript, r.AudioTrack, r.Caption)
		if err != nil {
			return nil, fmt.Errorf("CombineResults: build prompt: %w", err)
		}
	}

	msg := components.CreateChatMessagesUser(components.ChatUserMessage{
		Content: components.CreateChatUserMessageContentStr(sb.String()),
	})

	result, err := e.sendChat(ctx, []components.ChatMessages{msg})
	if err != nil {
		return nil, fmt.Errorf("CombineResults: %w", err)
	}

	e.slogger.InfoContext(ctx, "CombineResults done")
	return result, nil
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}
