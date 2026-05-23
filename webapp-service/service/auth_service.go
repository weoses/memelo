package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/weoses/memelo/webapp/conf"
)

type AuthService interface {
	Login(ctx context.Context, username, password string) (accessToken, refreshToken string, err error)
	RefreshAccessToken(ctx context.Context, refreshToken string) (accessToken string, err error)
	ExtractUserID(tokenStr string) (userID string, err error)
}

type authService struct {
	jwtSecret   []byte
	log         *slog.Logger
	defaultUuid string
}

func NewAuthService(cfg *conf.Config) (AuthService, error) {
	return &authService{
		jwtSecret:   []byte(cfg.Jwt.Secret),
		log:         slog.With("service", "auth_service"),
		defaultUuid: cfg.Account.Id,
	}, nil
}

func (s *authService) Login(_ context.Context, _, _ string) (string, string, error) {
	access, err := s.signToken("access", s.defaultUuid, 15*time.Minute)
	if err != nil {
		return "", "", err
	}
	refresh, err := s.signToken("refresh", s.defaultUuid, 7*24*time.Hour)
	if err != nil {
		return "", "", err
	}
	return access, refresh, nil
}

func (s *authService) RefreshAccessToken(_ context.Context, refreshToken string) (string, error) {
	claims, err := s.parseToken(refreshToken)
	if err != nil {
		return "", fmt.Errorf("invalid refresh token: %w", err)
	}
	if claims["typ"] != "refresh" {
		return "", fmt.Errorf("token is not a refresh token")
	}
	sub, _ := claims["sub"].(string)
	return s.signToken("access", sub, 15*time.Minute)
}

func (s *authService) ExtractUserID(tokenStr string) (string, error) {
	claims, err := s.parseToken(tokenStr)
	if err != nil {
		return "", fmt.Errorf("invalid access token: %w", err)
	}
	if claims["typ"] != "access" {
		return "", fmt.Errorf("token is not an access token")
	}
	sub, _ := claims["sub"].(string)
	return sub, nil
}

func (s *authService) signToken(typ, sub string, ttl time.Duration) (string, error) {
	claims := jwt.MapClaims{
		"typ": typ,
		"sub": sub,
		"exp": time.Now().Add(ttl).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(s.jwtSecret)
	if err != nil {
		return "", fmt.Errorf("sign token: %w", err)
	}
	return signed, nil
}

func (s *authService) parseToken(tokenStr string) (jwt.MapClaims, error) {
	token, err := jwt.Parse(tokenStr, func(t *jwt.Token) (interface{}, error) {
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
		return nil, fmt.Errorf("invalid claims")
	}
	return claims, nil
}
