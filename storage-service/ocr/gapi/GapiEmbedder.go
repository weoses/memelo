package gapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"log/slog"
	"net/url"
	"strings"
	"time"

	aiplatform "cloud.google.com/go/aiplatform/apiv1beta1"
	"cloud.google.com/go/aiplatform/apiv1beta1/aiplatformpb"
	"github.com/googleapis/gax-go/v2"
	"github.com/weoses/memelo/common/helper"
	"github.com/weoses/memelo/common/temp"

	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"golang.org/x/time/rate"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

const embeddingRatePerSecond = 2

type GcloudImageEmbeddingExtractorImpl struct {
	client           *aiplatform.PredictionClient
	endpoint         string
	dimension        int
	model            string
	videoIntervalSec int
	limiter          *rate.Limiter
	slogger          *slog.Logger
}

// toGcsUri converts a Google Storage HTTP URL to a gs:// URI.
// Accepts gs:// URIs as-is, and converts storage.googleapis.com / storage.google.com
// HTTP URLs to gs://bucket/path form. Returns ("", false) for non-GCS URLs.
func toGcsUri(rawURL string) (string, bool) {
	if strings.HasPrefix(rawURL, "gs://") {
		return rawURL, true
	}
	// Normalise: add a scheme so url.Parse works for scheme-less URLs.
	normalized := rawURL
	if !strings.Contains(rawURL, "://") {
		normalized = "https://" + rawURL
	}
	u, err := url.Parse(normalized)
	if err != nil {
		return "", false
	}
	host := strings.ToLower(u.Hostname())
	if host != "storage.googleapis.com" && host != "storage.google.com" {
		return "", false
	}
	// Path is /bucket/rest/of/object — trim leading slash.
	return "gs:/" + u.Path, true
}

func (i *GcloudImageEmbeddingExtractorImpl) GetImageEmbedding(ctx context.Context, rawImage temp.Data) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetImageEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	var embeddingItem *entity.EmbeddingItem
	var err error
	if s3data, ok := rawImage.(temp.S3BackedData); ok {
		if s3path, pathErr := s3data.GetS3Path(ctx); pathErr == nil {
			if gcsURI, ok := toGcsUri(s3path); ok {
				i.slogger.InfoContext(ctx, "GetImageEmbedding using gcsUri", "uri", gcsURI)
				embeddingItem, err = i.generateWithLowerDimension(nil, &gcsURI)
				if err != nil {
					return nil, fmt.Errorf("failed to generate embedding: %w", err)
				}
				i.slogger.InfoContext(ctx, "GetImageEmbedding done")
				return embeddingItem, nil
			}
		}
	}
	{
		i.slogger.InfoContext(ctx, "GetImageEmbedding using base64")
		bufBase64 := bytes.NewBufferString("")
		base64encoder := base64.NewEncoder(base64.RawStdEncoding, bufBase64)
		var reader io.ReadCloser
		reader, err = rawImage.Reader()
		if err != nil {
			return nil, fmt.Errorf("error reading temp: %w", err)
		}
		defer helper.QuietClose(reader, i.slogger)
		if _, err = io.Copy(base64encoder, reader); err != nil {
			return nil, fmt.Errorf("error encoding temp: %w", err)
		}
		base64Str := bufBase64.String()
		embeddingItem, err = i.generateWithLowerDimension(&base64Str, nil)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	i.slogger.InfoContext(ctx, "GetImageEmbedding done")
	return embeddingItem, nil
}

// generateWithLowerDimension shows how to generate lower-dimensional embeddings for text and image inputs.
// Exactly one of dataImageBase64 or gcsUri must be non-nil.
func (i *GcloudImageEmbeddingExtractorImpl) generateWithLowerDimension(
	dataImageBase64 *string,
	gcsUri *string,
) (*entity.EmbeddingItem, error) {
	ctx := context.Background()

	imagePayload := map[string]any{}
	if gcsUri != nil {
		imagePayload["gcsUri"] = *gcsUri
	} else {
		imagePayload["bytesBase64Encoded"] = *dataImageBase64
	}

	// This is the input to the model's prediction call. For schema, see:
	// https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/multimodal-embeddings-api#request_body
	instance, err := structpb.NewValue(map[string]any{
		"image": imagePayload,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to construct request payload: %w", err)
	}

	// TODO(developer): Try different dimensions: 128, 256, 512, 1408
	//outputDimensionality := 128
	params, err := structpb.NewValue(map[string]any{
		"dimension": i.dimension,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct request params: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint: i.endpoint,
		// The model supports only 1 instance per request.
		Instances:  []*structpb.Value{instance},
		Parameters: params,
	}

	resp, err := i.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embeddings: %w", err)
	}

	instanceEmbeddingsJson, err := protojson.Marshal(resp.GetPredictions()[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf value to JSON: %w", err)
	}
	// For response schema, see:
	// https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/multimodal-embeddings-api#response-body
	var instanceEmbeddings struct {
		ImageEmbeddings []float32 `json:"imageEmbedding"`
		TextEmbeddings  []float32 `json:"textEmbedding"`
	}
	if err := json.Unmarshal(instanceEmbeddingsJson, &instanceEmbeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	imageEmbedding := instanceEmbeddings.ImageEmbeddings
	//textEmbedding := instanceEmbeddings.TextEmbeddings

	return &entity.EmbeddingItem{
		Data:  imageEmbedding,
		Model: i.model,
		Type:  entity.EmbeddingTypeImage,
	}, nil
}

func (i *GcloudImageEmbeddingExtractorImpl) GetVideoEmbedding(ctx context.Context, video temp.Data) ([]*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetVideoEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}

	videoPayload := map[string]any{}
	if s3data, ok := video.(temp.S3BackedData); ok {
		if s3path, pathErr := s3data.GetS3Path(ctx); pathErr == nil {
			if gcsUri, ok := toGcsUri(s3path); ok {
				i.slogger.InfoContext(ctx, "GetVideoEmbedding using gcsUri", "uri", gcsUri)
				videoPayload["gcsUri"] = gcsUri
			}
		}
	}
	if _, hasGcs := videoPayload["gcsUri"]; !hasGcs {
		i.slogger.InfoContext(ctx, "GetVideoEmbedding using base64")
		bufBase64 := bytes.NewBufferString("")
		base64encoder := base64.NewEncoder(base64.RawStdEncoding, bufBase64)
		reader, err := video.Reader()
		if err != nil {
			return nil, fmt.Errorf("error reading temp: %w", err)
		}
		defer helper.QuietClose(reader, i.slogger)
		if _, err = io.Copy(base64encoder, reader); err != nil {
			return nil, fmt.Errorf("error encoding temp: %w", err)
		}
		videoPayload["bytesBase64Encoded"] = bufBase64.String()
	}

	videoPayload["videoSegmentConfig"] = map[string]any{
		"intervalSec": i.videoIntervalSec,
	}
	// This is the input to the model's prediction call. For schema, see:
	// https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/multimodal-embeddings-api#request_body
	instance, err := structpb.NewValue(map[string]any{
		"video": videoPayload,
	})

	if err != nil {
		return nil, fmt.Errorf("failed to construct request payload: %w", err)
	}

	params, err := structpb.NewValue(map[string]any{
		"dimension": i.dimension,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct request params: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:   i.endpoint,
		Instances:  []*structpb.Value{instance},
		Parameters: params,
	}

	resp, err := i.client.Predict(ctx, req, gax.WithTimeout(40*time.Second))
	if err != nil {
		return nil, fmt.Errorf("failed to generate video embeddings: %w", err)
	}

	predictionJSON, err := protojson.Marshal(resp.GetPredictions()[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf value to JSON: %w", err)
	}

	var prediction struct {
		VideoEmbeddings []struct {
			Embedding []float32 `json:"embedding"`
		} `json:"videoEmbeddings"`
	}
	if err := json.Unmarshal(predictionJSON, &prediction); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	items := make([]*entity.EmbeddingItem, len(prediction.VideoEmbeddings))
	for idx, ve := range prediction.VideoEmbeddings {
		items[idx] = &entity.EmbeddingItem{
			Data:  ve.Embedding,
			Model: i.model,
			Type:  entity.EmbeddingTypeVideo,
		}
	}

	i.slogger.InfoContext(ctx, "GetVideoEmbedding done", "segments", len(items))
	return items, nil
}

func (i *GcloudImageEmbeddingExtractorImpl) GetTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	i.slogger.InfoContext(ctx, "GetTextEmbedding start")
	if err := i.limiter.Wait(ctx); err != nil {
		return nil, fmt.Errorf("rate limiter: %w", err)
	}
	embedding, err := i.generateTextEmbedding(ctx, text)
	if err != nil {
		return nil, fmt.Errorf("failed to generate embedding: %w", err)
	}
	i.slogger.InfoContext(ctx, "GetTextEmbedding done")
	return embedding, nil
}

func (i *GcloudImageEmbeddingExtractorImpl) generateTextEmbedding(ctx context.Context, text string) (*entity.EmbeddingItem, error) {
	instance, err := structpb.NewValue(map[string]any{
		"text": text,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct request payload: %w", err)
	}

	params, err := structpb.NewValue(map[string]any{
		"dimension": i.dimension,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to construct request params: %w", err)
	}

	req := &aiplatformpb.PredictRequest{
		Endpoint:   i.endpoint,
		Instances:  []*structpb.Value{instance},
		Parameters: params,
	}

	resp, err := i.client.Predict(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("failed to generate text embeddings: %w", err)
	}

	instanceEmbeddingsJson, err := protojson.Marshal(resp.GetPredictions()[0])
	if err != nil {
		return nil, fmt.Errorf("failed to convert protobuf value to JSON: %w", err)
	}

	var instanceEmbeddings struct {
		TextEmbeddings []float32 `json:"textEmbedding"`
	}
	if err := json.Unmarshal(instanceEmbeddingsJson, &instanceEmbeddings); err != nil {
		return nil, fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	textEmbedding := instanceEmbeddings.TextEmbeddings
	return &entity.EmbeddingItem{
		Data:  textEmbedding,
		Model: i.model,
		Type:  entity.EmbeddingTypeText,
	}, nil
}

func NewImageEmbeddingExtractor(cfg *conf.Config) (ocr.EmbeddingExtractor, error) {
	cnf := cfg.MediaEmbedding
	cnfEmbeddings := cfg.Embeddings
	client, err := aiplatform.NewPredictionClient(context.Background(), option.WithEndpoint(cnf.ApiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to construct API client: %w", err)
	}

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", cnf.ProjectName, cnf.ApiLocation, cnf.Model)

	return &GcloudImageEmbeddingExtractorImpl{
		client:           client,
		endpoint:         endpoint,
		dimension:        cnfEmbeddings.Dimensions,
		model:            cnf.Model,
		videoIntervalSec: cnfEmbeddings.VideoEmbeddingIntervalSec,
		limiter:          rate.NewLimiter(embeddingRatePerSecond, embeddingRatePerSecond),
		slogger:          slog.With("service", "EmbeddingExtractor"),
	}, nil
}
