package gapi

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1beta1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1beta1/aiplatformpb"
	"github.com/weoses/memelo/storage-service/conf"
	"github.com/weoses/memelo/storage-service/entity"
	"github.com/weoses/memelo/storage-service/ocr"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
)

type GcloudImageEmbeddingExtractorImpl struct {
	client    *aiplatform.PredictionClient
	endpoint  string
	dimension int
	model     string
}

// GetIconMatrix implements ImageCompareKeyExtractor.
func (i *GcloudImageEmbeddingExtractorImpl) GetImageEmbeddingV1(ctx context.Context, rawImage []byte) (*entity.ElasticEmbeddingV1, error) {
	bufBase64Reader := bytes.NewBufferString("")
	bufBytesWriter := base64.NewEncoder(base64.RawStdEncoding, bufBase64Reader)
	bufBytesWriter.Write(rawImage)
	bufBytesWriter.Close()
	base64Str := bufBase64Reader.String()

	return i.generateWithLowerDimension(&base64Str)
}

// generateWithLowerDimension shows how to generate lower-dimensional embeddings for text and image inputs.
func (i *GcloudImageEmbeddingExtractorImpl) generateWithLowerDimension(
	dataImageBase64 *string,
) (*entity.ElasticEmbeddingV1, error) {
	// location = "us-central1"
	ctx := context.Background()

	// This is the input to the model's prediction call. For schema, see:
	// https://cloud.google.com/vertex-ai/generative-ai/docs/model-reference/multimodal-embeddings-api#request_body
	instance, err := structpb.NewValue(map[string]any{
		"image": map[string]any{
			// Image input can be provided either as a Google Cloud Storage URI or as
			// base64-encoded bytes using the "bytesBase64Encoded" field.
			//"gcsUri": "gs://cloud-samples-data/vertex-ai/llm/prompts/landmark1.png",
			"bytesBase64Encoded": *dataImageBase64,
		},
	})

	if err != nil {
		return nil, fmt.Errorf("failed to construct request payload: %w", err)
	}

	// TODO(developer): Try different dimenions: 128, 256, 512, 1408
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

	return &entity.ElasticEmbeddingV1{
		Data:  &imageEmbedding,
		Model: i.model,
	}, nil
}

func (i *GcloudImageEmbeddingExtractorImpl) GetTextEmbeddingV1(ctx context.Context, text string) (*entity.ElasticEmbeddingV1, error) {
	return i.generateTextEmbedding(ctx, text)
}

func (i *GcloudImageEmbeddingExtractorImpl) generateTextEmbedding(ctx context.Context, text string) (*entity.ElasticEmbeddingV1, error) {
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
	return &entity.ElasticEmbeddingV1{
		Data:  &textEmbedding,
		Model: i.model,
	}, nil
}

func NewImageEmbeddingExtractor(cnf *conf.ImageEmbeddingConfig) (ocr.EmbeddingExtractor, error) {
	apiEndpoint := cnf.ApiEndpoint
	client, err := aiplatform.NewPredictionClient(context.Background(), option.WithEndpoint(apiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to construct API client: %w", err)
	}

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", cnf.ProjectName, cnf.ApiLocation, cnf.Model)

	return &GcloudImageEmbeddingExtractorImpl{
		client:    client,
		endpoint:  endpoint,
		dimension: cnf.Dimension,
		model:     cnf.Model,
	}, nil
}
