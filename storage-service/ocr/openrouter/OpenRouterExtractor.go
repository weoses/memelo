package openrouter

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"strings"

	orsdk "github.com/OpenRouterTeam/go-sdk"
	"github.com/OpenRouterTeam/go-sdk/models/components"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/ocr"
)

const orOnScreenTextProp = "on_screen_text"
const orAudioTranscriptProp = "audio_transcript"
const orAudioTrackProp = "audio_track"
const orCaptionProp = "caption"
const orToolName = "extract_metadata"

type OpenRouterExtractor struct {
	client  *orsdk.OpenRouter
	cfg     *conf.OpenRouterExtractorConfig
	wrapper *ocr.ConnectorWrapper
	slogger *slog.Logger
}

func NewOpenRouterExtractor(cfg *conf.Config) (ocr.LlmMediaExtractor, error) {
	c := cfg.OpenRouterExtractor
	if c == nil {
		return nil, fmt.Errorf("openrouter-extractor config is missing")
	}
	client := orsdk.New(orsdk.WithSecurity(c.ApiKey))
	return &OpenRouterExtractor{
		client:  client,
		cfg:     c,
		wrapper: ocr.NewConnectorWrapper(2, 3),
		slogger: slog.With("service", "OpenRouterExtractor"),
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
			Text: e.cfg.ImageExtractPrompt,
		}),
		components.CreateChatContentItemsImageURL(components.ChatContentImage{
			Type:     components.ChatContentImageTypeImageURL,
			ImageURL: components.ChatContentImageImageURL{URL: dataURL},
		}),
	})
	return components.CreateChatMessagesUser(components.ChatUserMessage{Content: content}), nil
}

func (e *OpenRouterExtractor) buildVideoMessage(ctx context.Context, data temp.Data) (components.ChatMessages, error) {
	dataURL, err := toDataURL(data, "video/mp4", e.slogger)
	if err != nil {
		return components.ChatMessages{}, fmt.Errorf("build video message: %w", err)
	}

	content := components.CreateChatUserMessageContentArrayOfChatContentItems([]components.ChatContentItems{
		components.CreateChatContentItemsText(components.ChatContentText{
			Type: components.ChatContentTextTypeText,
			Text: e.cfg.VideoExtractPrompt,
		}),
		components.CreateChatContentItemsVideoURL(components.ChatContentVideo{
			Type:     components.ChatContentVideoTypeVideoURL,
			VideoURL: components.ChatContentVideoInput{URL: dataURL},
		}),
	})
	return components.CreateChatMessagesUser(components.ChatUserMessage{Content: content}), nil
}

func (e *OpenRouterExtractor) buildAudioMessage(ctx context.Context, data temp.Data) (components.ChatMessages, error) {
	r, err := data.Reader()
	if err != nil {
		return components.ChatMessages{}, fmt.Errorf("build audio message: read data: %w", err)
	}
	defer helper.QuietClose(r, e.slogger)

	raw, err := io.ReadAll(r)
	if err != nil {
		return components.ChatMessages{}, fmt.Errorf("build audio message: read all: %w", err)
	}

	content := components.CreateChatUserMessageContentArrayOfChatContentItems([]components.ChatContentItems{
		components.CreateChatContentItemsText(components.ChatContentText{
			Type: components.ChatContentTextTypeText,
			Text: e.cfg.AudioExtractPrompt,
		}),
		components.CreateChatContentItemsInputAudio(components.ChatContentAudio{
			InputAudio: components.ChatContentAudioInputAudio{
				Data:   base64.StdEncoding.EncodeToString(raw),
				Format: "mp3",
			},
		}),
	})
	return components.CreateChatMessagesUser(components.ChatUserMessage{Content: content}), nil
}

func (e *OpenRouterExtractor) ProcessAudio(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "ProcessAudio start")

	var result *ocr.MediaExtractResult
	if err := e.wrapper.Do(ctx, func() error {
		msg, err := e.buildAudioMessage(ctx, data)
		if err != nil {
			return fmt.Errorf("ProcessAudio: %w", err)
		}
		result, err = e.sendChat(ctx, []components.ChatMessages{msg}, e.cfg.ModelAudio)
		if err != nil {
			return fmt.Errorf("ProcessAudio: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "ProcessAudio done")
	return result, nil
}

func (e *OpenRouterExtractor) sendChat(ctx context.Context, messages []components.ChatMessages, model string) (*ocr.MediaExtractResult, error) {
	toolChoice := components.CreateChatToolChoiceChatNamedToolChoice(components.ChatNamedToolChoice{
		Type:     components.ChatNamedToolChoiceTypeFunction,
		Function: components.ChatNamedToolChoiceFunction{Name: orToolName},
	})

	resp, err := e.client.Chat.Send(ctx, components.ChatRequest{
		Model:      &model,
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

	var result *ocr.MediaExtractResult
	if err := e.wrapper.Do(ctx, func() error {
		msg, err := e.buildImageMessage(ctx, data)
		if err != nil {
			return err
		}
		result, err = e.sendChat(ctx, []components.ChatMessages{msg}, e.cfg.ModelImage)
		if err != nil {
			return fmt.Errorf("ProcessImage: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "ProcessImage done")
	return result, nil
}

func (e *OpenRouterExtractor) ProcessVideo(ctx context.Context, data temp.Data) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "ProcessVideo start")

	var result *ocr.MediaExtractResult
	if err := e.wrapper.Do(ctx, func() error {
		msg, err := e.buildVideoMessage(ctx, data)
		if err != nil {
			return fmt.Errorf("ProcessVideo: failed to build video message %w", err)
		}
		result, err = e.sendChat(ctx, []components.ChatMessages{msg}, e.cfg.ModelVideo)
		if err != nil {
			return fmt.Errorf("ProcessVideo: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "ProcessVideo done")
	return result, nil
}

func (e *OpenRouterExtractor) CombineResults(ctx context.Context, results []ocr.MediaExtractResult) (*ocr.MediaExtractResult, error) {
	e.slogger.InfoContext(ctx, "CombineResults start")

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

	var result *ocr.MediaExtractResult
	if err := e.wrapper.Do(ctx, func() error {
		msg := components.CreateChatMessagesUser(components.ChatUserMessage{
			Content: components.CreateChatUserMessageContentStr(sb.String()),
		})
		var err error
		result, err = e.sendChat(ctx, []components.ChatMessages{msg}, e.cfg.ModelImage)
		if err != nil {
			return fmt.Errorf("CombineResults: %w", err)
		}
		return nil
	}); err != nil {
		return nil, err
	}

	e.slogger.InfoContext(ctx, "CombineResults done")
	return result, nil
}

func (e *OpenRouterExtractor) CheckDuplicate(ctx context.Context, a temp.Data, b temp.Data) (bool, error) {
	e.slogger.InfoContext(ctx, "CheckDuplicate start")

	var isDuplicate bool
	if err := e.wrapper.Do(ctx, func() error {
		dataURLa, err := toDataURL(a, "image/jpeg", e.slogger)
		if err != nil {
			return fmt.Errorf("CheckDuplicate: data url a: %w", err)
		}
		dataURLb, err := toDataURL(b, "image/jpeg", e.slogger)
		if err != nil {
			return fmt.Errorf("CheckDuplicate: data url b: %w", err)
		}

		content := components.CreateChatUserMessageContentArrayOfChatContentItems([]components.ChatContentItems{
			components.CreateChatContentItemsText(components.ChatContentText{
				Type: components.ChatContentTextTypeText,
				Text: e.cfg.DuplicatePrompt,
			}),
			components.CreateChatContentItemsImageURL(components.ChatContentImage{
				Type:     components.ChatContentImageTypeImageURL,
				ImageURL: components.ChatContentImageImageURL{URL: dataURLa},
			}),
			components.CreateChatContentItemsImageURL(components.ChatContentImage{
				Type:     components.ChatContentImageTypeImageURL,
				ImageURL: components.ChatContentImageImageURL{URL: dataURLb},
			}),
		})
		msg := components.CreateChatMessagesUser(components.ChatUserMessage{Content: content})

		const toolName = "check_duplicate"
		desc := "Check whether the two provided images are duplicates"
		tool := components.CreateChatFunctionToolChatFunctionToolFunction(components.ChatFunctionToolFunction{
			Type: components.ChatFunctionToolTypeFunction,
			Function: components.ChatFunctionToolFunctionFunction{
				Name:        toolName,
				Description: &desc,
				Parameters: map[string]any{
					"type": "object",
					"properties": map[string]any{
						"is_duplicate": map[string]any{"type": "boolean"},
					},
					"required": []string{"is_duplicate"},
				},
			},
		})

		toolChoice := components.CreateChatToolChoiceChatNamedToolChoice(components.ChatNamedToolChoice{
			Type:     components.ChatNamedToolChoiceTypeFunction,
			Function: components.ChatNamedToolChoiceFunction{Name: toolName},
		})

		resp, err := e.client.Chat.Send(ctx, components.ChatRequest{
			Model:      &e.cfg.ModelImage,
			Messages:   []components.ChatMessages{msg},
			Tools:      []components.ChatFunctionTool{tool},
			ToolChoice: &toolChoice,
		})
		if err != nil {
			return fmt.Errorf("CheckDuplicate chat send: %w", err)
		}
		if resp.ChatResult == nil || len(resp.ChatResult.Choices) == 0 {
			return fmt.Errorf("CheckDuplicate: chat returned no choices")
		}

		toolCalls := resp.ChatResult.Choices[0].Message.ToolCalls
		if len(toolCalls) == 0 {
			return fmt.Errorf("model did not return %s call", toolName)
		}

		var args map[string]any
		if err := json.Unmarshal([]byte(toolCalls[0].Function.Arguments), &args); err != nil {
			return fmt.Errorf("CheckDuplicate: parse tool arguments: %w", err)
		}

		isDuplicate = boolArg(args, "is_duplicate")
		return nil
	}); err != nil {
		return false, err
	}

	e.slogger.InfoContext(ctx, "CheckDuplicate done")
	return isDuplicate, nil
}

func stringArg(args map[string]any, key string) string {
	if v, ok := args[key]; ok {
		if s, ok := v.(string); ok {
			return s
		}
	}
	return ""
}

func boolArg(args map[string]any, key string) bool {
	if v, ok := args[key]; ok {
		if b, ok := v.(bool); ok {
			return b
		}
	}
	return false
}
