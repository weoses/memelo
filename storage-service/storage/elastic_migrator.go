package storage

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	iofs "io/fs"
	"log/slog"
	"net/http"
	"strings"

	"embed"

	elasticsearch8 "github.com/elastic/go-elasticsearch/v8"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types"
	"github.com/elastic/go-elasticsearch/v8/typedapi/types/enums/refresh"
	"github.com/weoses/memelo/common/helper"
)

const MigrationHistoryIndex = "elastic-migrations-history"

type ElasticMigrating interface {
	Migrate(ctx context.Context) error
}

type migrationVersion struct {
	IndexName     string `json:"IndexName"`
	LastMigration string `json:"LastMigration"`
}

type migration struct {
	Method string          `json:"method"`
	URL    string          `json:"url"`
	Body   json.RawMessage `json:"body"`
}

type ElasticMigrator struct {
	typedClient  *elasticsearch8.TypedClient
	rawClient    *elasticsearch8.Client
	migrationFS  embed.FS
	fsRoot       string
	index        string
	historyIndex string
	vars         map[string]string
	slogger      *slog.Logger
}

func NewElasticMigrator(
	esConfig *elasticsearch8.Config,
	fs embed.FS,
	fsRoot string,
	index string,
	historyIndex string,
	vars map[string]string,
	logger *slog.Logger,
) (*ElasticMigrator, error) {
	typedClient, err := elasticsearch8.NewTypedClient(*esConfig)
	if err != nil {
		return nil, fmt.Errorf("create typed elastic client failed: %w", err)
	}
	rawClient, err := elasticsearch8.NewClient(*esConfig)
	if err != nil {
		return nil, fmt.Errorf("create raw elastic client failed: %w", err)
	}

	return &ElasticMigrator{
		typedClient:  typedClient,
		rawClient:    rawClient,
		migrationFS:  fs,
		fsRoot:       fsRoot,
		index:        index,
		historyIndex: historyIndex,
		vars:         vars,
		slogger:      logger,
	}, nil
}

func (m *ElasticMigrator) Migrate(ctx context.Context) error {
	if err := m.ensureHistoryIndex(ctx); err != nil {
		return fmt.Errorf("ensure history index failed: %w", err)
	}

	entries, err := iofs.ReadDir(m.migrationFS, m.fsRoot)
	if err != nil {
		return fmt.Errorf("read migration dir %s failed: %w", m.fsRoot, err)
	}

	lastMigration, err := m.getLastMigration(ctx)
	if err != nil {
		return fmt.Errorf("get last migration failed: %w", err)
	}

	m.slogger.InfoContext(ctx, "Migrate: last applied migration",
		"index", m.index,
		"lastMigration", lastMigration,
	)

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if name <= lastMigration {
			continue
		}

		err = m.applyEntry(ctx, name)
		if err != nil {
			return fmt.Errorf("apply migration entry failed: %w", err)
		}
	}

	return nil
}

func (m *ElasticMigrator) applyEntry(ctx context.Context, name string) error {
	m.slogger.InfoContext(ctx, "Migrate: applying migration",
		"index", m.index,
		"migration", name,
	)

	data, err := m.migrationFS.ReadFile(m.fsRoot + "/" + name)
	if err != nil {
		return fmt.Errorf("read migration file %s failed: %w", name, err)
	}

	content := string(data)
	for k, v := range m.vars {
		content = strings.ReplaceAll(content, "{"+k+"}", v)
	}

	var mig migration
	if err := json.Unmarshal([]byte(content), &mig); err != nil {
		return fmt.Errorf("unmarshal migration %s failed: %w", name, err)
	}

	req, err := http.NewRequestWithContext(ctx, mig.Method, mig.URL, bytes.NewReader(mig.Body))
	if err != nil {
		return fmt.Errorf("create request for migration %s failed: %w", name, err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := m.rawClient.Perform(req)
	if err != nil {
		return fmt.Errorf("execute migration %s failed: %w", name, err)
	}
	defer helper.QuietClose(resp.Body, m.slogger)
	responseBodyBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read elastic response failed: %w", err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return fmt.Errorf("migration %s returned non-2xx status: %d with body %s", name, resp.StatusCode, string(responseBodyBytes))
	}

	if err := m.saveLastMigration(ctx, name); err != nil {
		return fmt.Errorf("save last migration %s failed: %w", name, err)
	}

	m.slogger.InfoContext(ctx, "Migrate: migration applied",
		"index", m.index,
		"migration", name,
	)
	return nil
}

func (m *ElasticMigrator) ensureHistoryIndex(ctx context.Context) error {
	exists, err := m.typedClient.Indices.Exists(m.historyIndex).Do(ctx)
	if err != nil {
		return err
	}
	if !exists {
		_, err = m.typedClient.Indices.Create(m.historyIndex).Do(ctx)
		return err
	}
	return nil
}

func (m *ElasticMigrator) getLastMigration(ctx context.Context) (string, error) {
	resp, err := m.typedClient.Get(m.historyIndex, m.index).Do(ctx)
	if err != nil {
		var esErr *types.ElasticsearchError
		if errors.As(err, &esErr) && esErr.Status == 404 {
			return "", nil
		}
		return "", err
	}
	if !resp.Found {
		return "", nil
	}
	var ver migrationVersion
	if err := json.Unmarshal(resp.Source_, &ver); err != nil {
		return "", err
	}
	return ver.LastMigration, nil
}

func (m *ElasticMigrator) saveLastMigration(ctx context.Context, name string) error {
	ver := migrationVersion{
		IndexName:     m.index,
		LastMigration: name,
	}
	_, err := m.typedClient.Index(m.historyIndex).
		Id(m.index).
		Document(ver).
		Refresh(refresh.True).
		Do(ctx)
	return err
}

func RunMigrations(migrators []ElasticMigrating) error {
	ctx := context.Background()
	for _, mg := range migrators {
		if err := mg.Migrate(ctx); err != nil {
			return err
		}
	}
	return nil
}
