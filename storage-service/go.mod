module github.com/weoses/memelo/storage-service

go 1.25.0

require (
	cloud.google.com/go/aiplatform v1.117.0
	cloud.google.com/go/speech v1.30.0
	cloud.google.com/go/vision v1.2.0
	cloud.google.com/go/vision/v2 v2.9.6
	connectrpc.com/connect v1.19.1
	github.com/agnivade/levenshtein v1.2.1
	github.com/google/uuid v1.6.0
	github.com/googleapis/gax-go/v2 v2.17.0
	github.com/h2non/bimg v1.1.9
	github.com/pkg/errors v0.9.1
	github.com/spf13/viper v1.19.0
	github.com/weoses/memelo/common v0.0.0-00010101000000-000000000000
	github.com/weoses/memelo/gen v0.0.0-00010101000000-000000000000
	go.uber.org/fx v1.23.0
	golang.org/x/time v0.14.0
	google.golang.org/api v0.265.0
	google.golang.org/genai v1.54.0
	google.golang.org/protobuf v1.36.11
)

require (
	cloud.google.com/go v0.123.0 // indirect
	cloud.google.com/go/auth v0.18.1 // indirect
	cloud.google.com/go/auth/oauth2adapt v0.2.8 // indirect
	cloud.google.com/go/compute/metadata v0.9.0 // indirect
	cloud.google.com/go/iam v1.5.3 // indirect
	cloud.google.com/go/longrunning v0.8.0 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/dustin/go-humanize v1.0.1 // indirect
	github.com/elastic/elastic-transport-go/v8 v8.6.0 // indirect
	github.com/felixge/httpsnoop v1.0.4 // indirect
	github.com/gabriel-vasile/mimetype v1.4.8 // indirect
	github.com/go-ini/ini v1.67.0 // indirect
	github.com/go-logr/logr v1.4.3 // indirect
	github.com/go-logr/stdr v1.2.2 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.4 // indirect
	github.com/google/go-cmp v0.7.0 // indirect
	github.com/google/s2a-go v0.1.9 // indirect
	github.com/googleapis/enterprise-certificate-proxy v0.3.11 // indirect
	github.com/gorilla/websocket v1.5.3 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	github.com/klauspost/compress v1.18.2 // indirect
	github.com/klauspost/cpuid/v2 v2.2.9 // indirect
	github.com/leodido/go-urn v1.4.0 // indirect
	github.com/minio/md5-simd v1.1.2 // indirect
	github.com/rs/xid v1.6.0 // indirect
	go.opentelemetry.io/auto/sdk v1.2.1 // indirect
	go.opentelemetry.io/contrib/instrumentation/google.golang.org/grpc/otelgrpc v0.61.0 // indirect
	go.opentelemetry.io/contrib/instrumentation/net/http/otelhttp v0.64.0 // indirect
	go.opentelemetry.io/otel v1.39.0 // indirect
	go.opentelemetry.io/otel/metric v1.39.0 // indirect
	go.opentelemetry.io/otel/trace v1.39.0 // indirect
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/goleak v1.3.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/crypto v0.47.0 // indirect
	golang.org/x/oauth2 v0.34.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	google.golang.org/genproto v0.0.0-20260128011058-8636f8732409 // indirect
	google.golang.org/genproto/googleapis/api v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260203192932-546029d2fa20 // indirect
	google.golang.org/grpc v1.78.0 // indirect
)

require (
	github.com/elastic/go-elasticsearch/v8 v8.16.0
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/gdexlab/go-render v1.0.1
	github.com/go-playground/validator/v10 v10.24.0
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.10 // indirect
	github.com/minio/minio-go/v7 v7.0.84
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20250813145105-42675adae3e6 // indirect
	golang.org/x/net v0.49.0
	golang.org/x/sys v0.40.0 // indirect
	golang.org/x/text v0.33.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/weoses/memelo/common => ../common
	github.com/weoses/memelo/gen => ../gen
)
