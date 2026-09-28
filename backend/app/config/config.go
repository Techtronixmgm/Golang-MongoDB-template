package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

type Config struct {
	MongoUri               string
	MongoDB                string
	ServerPort             string
	JWTSecret              string
	JWTExpiryHours         int
	RefreshTokenExpiryDays int

	AuthAccessCookie  string
	AuthRefreshCookie string

	CookieSecure                     bool
	GinMode                          string
	RefreshTokenRevokedRetentionDays int
}

func Load() (Config, error) {
	paths := []string{
		".env",
	}

	for _, p := range paths {
		if err := godotenv.Load(p); err == nil {
			break
		}
	}

	mongoURI, err := extractEnv("MONGO_URI")
	if err != nil {
		return Config{}, err
	}

	mongoDB, err := extractEnv("MONGO_DB_NAME")
	if err != nil {
		return Config{}, err
	}

	port, err := extractEnv("PORT")
	if err != nil {
		return Config{}, err
	}

	jwtSecret, err := extractEnv("JWT_SECRET")
	if err != nil {
		return Config{}, err
	}

	jwtExpiryHoursStr, err := extractEnv("JWT_EXPIRY_HOURS")
	if err != nil {
		return Config{}, err
	}

	jwtExpiryHours, err := strconv.Atoi(jwtExpiryHoursStr)
	if err != nil {
		return Config{}, fmt.Errorf(
			"invalid JWT_EXPIRY_HOURS: %w",
			err,
		)
	}

	refreshTokenExpiryDaysStr, err := extractEnv(
		"REFRESH_TOKEN_EXPIRY_DAYS",
	)
	if err != nil {
		return Config{}, err
	}

	refreshTokenExpiryDays, err := strconv.Atoi(
		refreshTokenExpiryDaysStr,
	)
	if err != nil {
		return Config{}, fmt.Errorf(
			"invalid REFRESH_TOKEN_EXPIRY_DAYS: %w",
			err,
		)
	}

	// Access cookie name
	authAccessCookie, err := extractEnv(
		"AUTH_ACCESS_COOKIE",
	)
	if err != nil {
		return Config{}, err
	}

	// Refresh cookie name
	authRefreshCookie, err := extractEnv(
		"AUTH_REFRESH_COOKIE",
	)
	if err != nil {
		return Config{}, err
	}

	cookieSecureStr, err := extractEnv("COOKIE_SECURE")
	if err != nil {
		return Config{}, err
	}

	cookieSecure, err := strconv.ParseBool(cookieSecureStr)
	if err != nil {
		return Config{}, fmt.Errorf(
			"invalid COOKIE_SECURE: %w",
			err,
		)
	}

	ginMode, err := extractEnv("GIN_MODE")
	if err != nil {
		return Config{}, err
	}

	// Refresh token revoked retention.
	// Defaults to 7 days when the environment variable is not set.
	refreshTokenRevokedRetentionDays := 7

	retentionDaysStr := strings.TrimSpace(
		os.Getenv("REFRESH_TOKEN_REVOKED_RETENTION_DAYS"),
	)

	if retentionDaysStr != "" {
		refreshTokenRevokedRetentionDays, err = strconv.Atoi(
			retentionDaysStr,
		)
		if err != nil {
			return Config{}, fmt.Errorf(
				"invalid REFRESH_TOKEN_REVOKED_RETENTION_DAYS: %w",
				err,
			)
		}
	}

	config := Config{
		MongoUri:                         mongoURI,
		MongoDB:                          mongoDB,
		ServerPort:                       port,
		JWTSecret:                        jwtSecret,
		JWTExpiryHours:                   jwtExpiryHours,
		AuthAccessCookie:                 authAccessCookie,
		AuthRefreshCookie:                authRefreshCookie,
		RefreshTokenExpiryDays:           refreshTokenExpiryDays,
		CookieSecure:                     cookieSecure,
		GinMode:                          ginMode,
		RefreshTokenRevokedRetentionDays: refreshTokenRevokedRetentionDays,
	}

	if err := config.Validate(); err != nil {
		return Config{}, err
	}

	return config, nil
}

// Validate validates the loaded configuration.
func (c Config) Validate() error {
	if c.MongoUri == "" {
		return errors.New("mongo uri missing")
	}

	if c.MongoDB == "" {
		return errors.New("mongo database missing")
	}

	if c.ServerPort == "" {
		return errors.New("server port missing")
	}

	if c.JWTSecret == "" {
		return errors.New("jwt secret missing")
	}

	if c.JWTExpiryHours <= 0 {
		return errors.New(
			"jwt expiry hours must be greater than zero",
		)
	}

	if c.RefreshTokenExpiryDays <= 0 {
		return errors.New(
			"refresh token expiry days must be greater than zero",
		)
	}

	if c.RefreshTokenRevokedRetentionDays <= 0 {
		return errors.New(
			"refresh token revoked retention days must be greater than zero",
		)
	}

	if c.AuthAccessCookie == "" {
		return errors.New(
			"access token cookie name is missing",
		)
	}

	if c.AuthRefreshCookie == "" {
		return errors.New(
			"refresh token cookie name is missing",
		)
	}

	if c.GinMode == "" {
		return errors.New("Gin mode is missing")
	}

	return nil
}

func extractEnv(key string) (string, error) {
	val := strings.TrimSpace(os.Getenv(key))

	if val == "" {
		return "", fmt.Errorf(
			"missing required environment variable: %s",
			key,
		)
	}

	return val, nil
}
