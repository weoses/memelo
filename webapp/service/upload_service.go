package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	commonservice "github.com/weoses/memelo/common/service"
	"github.com/weoses/memelo/webapp/conf"
)

type GetUploadUrlResult struct {
	UploadUrl  string            `json:"upload_url"`
	FormFields map[string]string `json:"form_fields"`
	Token      string            `json:"token"`
}

type UploadService interface {
	GetUploadUrl(ctx context.Context, mime string, size int64) (*GetUploadUrlResult, error)
	ParseByToken(ctx context.Context, accountId, token string) (*MemeResult, error)
}

type uploadService struct {
	tmp       commonservice.TmpDataService
	proxy     StorageProxy
	jwtSecret []byte
	log       *slog.Logger
}

func NewUploadService(tmp commonservice.TmpDataService, proxy StorageProxy, cfg *conf.Config) (UploadService, error) {
	return &uploadService{
		tmp:       tmp,
		proxy:     proxy,
		jwtSecret: []byte(cfg.Jwt.Secret),
		log:       slog.With("service", "upload_service"),
	}, nil
}

func (s *uploadService) GetUploadUrl(ctx context.Context, mime string, size int64) (*GetUploadUrlResult, error) {
	result, err := s.tmp.GetUploadUrl(ctx, mime, size)
	if err != nil {
		return nil, fmt.Errorf("get upload url: %w", err)
	}

	claims := jwt.MapClaims{
		"s3path": result.S3Path,
		"mime":   mime,
		"exp":    time.Now().Add(6 * time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return nil, fmt.Errorf("sign token: %w", err)
	}

	return &GetUploadUrlResult{
		UploadUrl:  result.URL,
		FormFields: result.FormFields,
		Token:      signed,
	}, nil
}

func (s *uploadService) ParseByToken(ctx context.Context, accountId, tokenStr string) (*MemeResult, error) {
	token, err := jwt.Parse(tokenStr,
		func(t *jwt.Token) (interface{}, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return s.jwtSecret, nil
		})
	if err != nil || !token.Valid {
		return nil, fmt.Errorf("invalid token: %w", err)
	}

	claims, ok := token.Claims.(jwt.MapClaims)
	if !ok {
		return nil, fmt.Errorf("invalid token claims")
	}

	s3path, ok := claims["s3path"].(string)
	if !ok || s3path == "" {
		return nil, fmt.Errorf("missing s3path claim")
	}
	mime, ok := claims["mime"].(string)
	if !ok || mime == "" {
		return nil, fmt.Errorf("missing mime claim")
	}

	return s.proxy.UploadByS3Path(ctx, accountId, s3path, mime)
}
