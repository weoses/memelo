package service

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"

	aiplatform "cloud.google.com/go/aiplatform/apiv1beta1"
	aiplatformpb "cloud.google.com/go/aiplatform/apiv1beta1/aiplatformpb"
	"google.golang.org/api/option"
	"google.golang.org/protobuf/encoding/protojson"
	"google.golang.org/protobuf/types/known/structpb"
	"mine.local/ocr-gallery/ocr-server/conf"
	"mine.local/ocr-gallery/ocr-server/entity"
)

type ImageEmbeddingExtractor interface {
	GetImageEmbeddingV1(ctx context.Context, image *entity.Image) (*EmbeddedImage, error)
}

type ImageEmbeddingExtractorImpl struct {
	client    *aiplatform.PredictionClient
	endpoint  string
	dimension int
	model     string
}
type EmbeddedImage struct {
	Data  []float32
	Model string
}

// GetIconMatrix implements ImageCompareKeyExtractor.
func (i *ImageEmbeddingExtractorImpl) GetImageEmbeddingV1(ctx context.Context, argImage *entity.Image) (*EmbeddedImage, error) {
	bufBase64Reader := bytes.NewBufferString("")
	bufBytesWriter := base64.NewEncoder(base64.RawStdEncoding, bufBase64Reader)
	bufBytesWriter.Write(*argImage.Data)
	bufBytesWriter.Close()
	base64Str := bufBase64Reader.String()

	return i.generateWithLowerDimension(&base64Str)
}

// generateWithLowerDimension shows how to generate lower-dimensional embeddings for text and image inputs.
func (i *ImageEmbeddingExtractorImpl) generateWithLowerDimension(
	dataImageBase64 *string,
) (*EmbeddedImage, error) {
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

	return &EmbeddedImage{
		Data:  imageEmbedding,
		Model: i.model,
	}, nil
}

func NewImageEmbeddingExtractor(cnf *conf.ImageEmbeddingConfig) (ImageEmbeddingExtractor, error) {
	apiEndpoint := fmt.Sprintf("%s-aiplatform.googleapis.com:443", cnf.ApiLocation)
	client, err := aiplatform.NewPredictionClient(context.Background(), option.WithEndpoint(apiEndpoint))
	if err != nil {
		return nil, fmt.Errorf("failed to construct API client: %w", err)
	}

	endpoint := fmt.Sprintf("projects/%s/locations/%s/publishers/google/models/%s", cnf.ProjectName, cnf.ApiLocation, cnf.Model)

	return &ImageEmbeddingExtractorImpl{
		client:    client,
		endpoint:  endpoint,
		dimension: cnf.Dimension,
		model:     cnf.Model,
	}, nil
}
