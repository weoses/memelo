package functional_test

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
	"github.com/testcontainers/testcontainers-go"
	tcElastic "github.com/testcontainers/testcontainers-go/modules/elasticsearch"
	tcMinio "github.com/testcontainers/testcontainers-go/modules/minio"
	commonconfig "github.com/weoses/memelo/common/config"
	"github.com/weoses/memelo/storage-service/conf"
)

const (
	testMediaBucket     = "test-media"
	testTempBucket      = "test-temp"
	testEsMetadataIndex = "metadata-db-test"
	testEsTagIndex      = "tag-db-test"
)

var (
	testAddr        string
	testStop        func()
	testMockEx      *MockLlmExtractor
	testMockEmb     *MockEmbeddingExtractor
	testEsClient    *elasticsearch8.Client
	testMinioClient *minio.Client
)

func TestMain(m *testing.M) {
	ctx := context.Background()

	// Start Elasticsearch
	esContainer, err := tcElastic.Run(ctx,
		"docker.elastic.co/elasticsearch/elasticsearch:8.16.2",
		testcontainers.WithEnv(map[string]string{
			"xpack.security.enabled": "false",
		}),
	)
	if err != nil {
		log.Fatalf("start elasticsearch: %v", err)
	}
	defer func() { _ = esContainer.Terminate(ctx) }()

	esAddress := esContainer.Settings.Address

	testEsClient, err = elasticsearch8.NewClient(elasticsearch8.Config{
		Addresses: []string{esAddress},
	})
	if err != nil {
		log.Fatalf("create es client: %v", err)
	}

	// Start MinIO
	minioContainer, err := tcMinio.Run(ctx, "minio/minio:latest")
	if err != nil {
		log.Fatalf("start minio: %v", err)
	}
	defer func() { _ = minioContainer.Terminate(ctx) }()

	minioEndpoint, err := minioContainer.ConnectionString(ctx)
	if err != nil {
		log.Fatalf("minio connection string: %v", err)
	}

	testMinioClient, err = minio.New(minioEndpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(minioContainer.Username, minioContainer.Password, ""),
		Secure: false,
	})
	if err != nil {
		log.Fatalf("create minio client: %v", err)
	}

	cfg := buildTestConfig(esAddress, minioEndpoint, minioContainer.Username, minioContainer.Password)

	testMockEx = &MockLlmExtractor{}
	testMockEmb = &MockEmbeddingExtractor{}

	testAddr, testStop, err = startTestServer(cfg, testMockEx, testMockEmb)
	if err != nil {
		log.Fatalf("start test server: %v", err)
	}
	defer testStop()

	os.Exit(m.Run())
}

func buildTestConfig(esAddr, minioEndpoint, minioUser, minioPass string) *conf.Config {
	storageConfig := func(bucket string) *commonconfig.MediaStorageConfig {
		return &commonconfig.MediaStorageConfig{
			Endpoint:  minioEndpoint,
			AccessKey: minioUser,
			SecretKey: minioPass,
			Bucket:    bucket,
			Secure:    false,
		}
	}

	return &conf.Config{
		Server: &commonconfig.ServerConfig{ListenAddress: ":0"},
		Log:    &commonconfig.LoggingConfig{Level: "warn"},
		Search: &conf.SearchConfig{
			SemanticDuplicateThreshold:  0.95,
			SemanticTextSearchThreshold: 0.7,
			Fuzziness:                   "AUTO",
		},
		Extracting: &conf.CommonExtractingConfig{
			EmbeddingDimensions:   mockEmbeddingDimensions,
			VideoSliceIntervalSec: 15,
		},
		MetadataDb: &conf.MetadataDbConfig{
			Elastic:                &elasticsearch8.Config{Addresses: []string{esAddr}},
			Index:                  testEsMetadataIndex,
			EmbeddingMatchTreshold: 0.9,
		},
		TagDb: &conf.TagDbConfig{
			Elastic: &elasticsearch8.Config{Addresses: []string{esAddr}},
			Index:   testEsTagIndex,
		},
		MediaStorage:   storageConfig(testMediaBucket),
		TempStorage:    storageConfig(testTempBucket),
		ImageConverter: &conf.ImageConverterConfig{ThumbSize: 360, OriginalMaxSize: 4096},
		Ffmpeg: &conf.FfmpegConfig{
			FfmpegBinary:  "ffmpeg",
			FfprobeBinary: "ffprobe",
			CpuLimit:      2,
			ThreadsLimit:  2,
		},
		GeminiExtractor: &conf.GeminiExtractorConfig{},
		GeminiEmbedding: &conf.GeminiEmbeddingConfig{},
	}
}

// resetState clears all Elasticsearch indices and S3 buckets between tests.
func resetState(t *testing.T) {
	t.Helper()
	ctx := context.Background()

	if err := clearElastic(ctx, testEsClient, testEsMetadataIndex, testEsTagIndex); err != nil {
		t.Logf("clearElastic: %v", err)
	}
	for _, bucket := range []string{testMediaBucket, testTempBucket} {
		if err := clearBucket(ctx, testMinioClient, bucket); err != nil {
			t.Logf("clearBucket %s: %v", bucket, err)
		}
	}

	testMockEx.Reset()
	testMockEmb.Reset()
}

// testClient returns a ready-to-use TestClient pointed at the running test server.
func testClient() *TestClient {
	return NewTestClient(testAddr)
}

// accountID returns a deterministic test account ID string.
// suffix must be in hex alphabet

func accountID(suffix string) string {
	return fmt.Sprintf("00000000-0000-0000-0000-%s", fmt.Sprintf("%012s", suffix))
}
