module github.com/weoses/memelo/telegram-service

go 1.25.0

require (
	github.com/golang-migrate/migrate/v4 v4.18.1
	github.com/google/uuid v1.6.0
	github.com/jackc/pgx/v5 v5.7.2
	github.com/spf13/viper v1.19.0
	github.com/weoses/memelo/common v0.0.0-00010101000000-000000000000
	github.com/weoses/memelo/gen v0.0.0-00010101000000-000000000000
)

require (
	connectrpc.com/connect v1.19.1 // indirect
	github.com/hashicorp/errwrap v1.1.0 // indirect
	github.com/hashicorp/go-multierror v1.1.1 // indirect
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20240606120523-5a60cdf6a761 // indirect
	github.com/jackc/puddle/v2 v2.2.2 // indirect
	github.com/joho/godotenv v1.5.1 // indirect
	go.uber.org/atomic v1.9.0 // indirect
	go.uber.org/dig v1.18.0 // indirect
	go.uber.org/zap v1.26.0 // indirect
	golang.org/x/crypto v0.44.0 // indirect
	golang.org/x/sync v0.18.0 // indirect
	google.golang.org/protobuf v1.36.11 // indirect
)

require (
	github.com/fsnotify/fsnotify v1.9.0 // indirect
	github.com/go-telegram-bot-api/telegram-bot-api/v5 v5.5.1
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mitchellh/mapstructure v1.5.0 // indirect
	github.com/pelletier/go-toml/v2 v2.2.2 // indirect
	github.com/sagikazarmark/locafero v0.4.0 // indirect
	github.com/sagikazarmark/slog-shim v0.1.0 // indirect
	github.com/sourcegraph/conc v0.3.0 // indirect
	github.com/spf13/afero v1.11.0 // indirect
	github.com/spf13/cast v1.6.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/subosito/gotenv v1.6.0 // indirect
	go.uber.org/fx v1.23.0
	go.uber.org/multierr v1.10.0 // indirect
	golang.org/x/exp v0.0.0-20230905200255-921286631fa9 // indirect
	golang.org/x/sys v0.38.0 // indirect
	golang.org/x/text v0.31.0 // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

replace (
	github.com/weoses/memelo/common => ../common
	github.com/weoses/memelo/gen => ../gen
)
