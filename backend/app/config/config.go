package config

import (
	"errors"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/joho/godotenv"
)

var (
	ErrTOTPEncryptionKeyMissing = errors.New("TOTP encryption key is required to enable 2FA")
	ErrTOTPEncryptionKeyTooWeak = errors.New("TOTP encryption key must be at least 32 bytes")
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

	CookieSameSite    string
	AllowedOrigins    []string
	TOTPIssuer        string
	TOTPEncryptionKey string
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

	// Cookie SameSite
	cookieSameSite, err := extractEnv(
		"COOKIE_SAME_SITE",
	)
	if err != nil {
		return Config{}, err
	}

	// Normalize Cookie SameSite value.
	switch strings.ToLower(cookieSameSite) {
	case "strict":
		cookieSameSite = "Strict"
	case "lax":
		cookieSameSite = "Lax"
	case "none":
		cookieSameSite = "None"
	default:
		return Config{}, errors.New(
			"COOKIE_SAME_SITE must be one of: Strict, Lax, None",
		)
	}

	// SameSite=None requires Secure=true.
	if cookieSameSite == "None" && !cookieSecure {
		return Config{}, errors.New(
			"COOKIE_SAME_SITE=None requires COOKIE_SECURE=true",
		)
	}

	ginMode, err := extractEnv("GIN_MODE")
	if err != nil {
		return Config{}, err
	}

	allowedOriginsStr, err := extractEnv("ALLOWED_ORIGINS")
	if err != nil {
		return Config{}, err
	}

	allowedOrigins := strings.Split(
		allowedOriginsStr,
		",",
	)

	for i := range allowedOrigins {
		allowedOrigins[i] = strings.TrimSpace(
			allowedOrigins[i],
		)

		if allowedOrigins[i] == "" {
			return Config{}, errors.New(
				"ALLOWED_ORIGINS contains an empty origin",
			)
		}
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

	tOTPIssuer, err := extractEnv("TOTP_ISSUER")
	if err != nil {
		return Config{}, err
	}

	// TOTP encryption key is intentionally optional at startup.
	// It is required only when 2FA is enabled or used.
	tOTPEncryptionKey := strings.TrimSpace(
		os.Getenv("TOTP_ENCRYPTION_KEY"),
	)

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
		CookieSameSite:                   cookieSameSite,
		AllowedOrigins:                   allowedOrigins,
		TOTPIssuer:                       tOTPIssuer,
		TOTPEncryptionKey:                tOTPEncryptionKey,
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

	if c.CookieSameSite == "" {
		return errors.New(
			"cookie SameSite setting is missing",
		)
	}

	// Validate supported SameSite values.
	switch c.CookieSameSite {
	case "Strict", "Lax", "None":
	default:
		return errors.New(
			"cookie SameSite must be one of: Strict, Lax, None",
		)
	}

	// SameSite=None requires Secure=true.
	if c.CookieSameSite == "None" && !c.CookieSecure {
		return errors.New(
			"cookie SameSite=None requires CookieSecure=true",
		)
	}

	if len(c.AllowedOrigins) == 0 {
		return errors.New("ALLOWED_ORIGINS missing")
	}

	// Validate that every configured origin is non-empty.
	for _, origin := range c.AllowedOrigins {
		if strings.TrimSpace(origin) == "" {
			return errors.New(
				"ALLOWED_ORIGINS contains an empty origin",
			)
		}
	}

	if strings.TrimSpace(c.TOTPIssuer) == "" {
		return errors.New("TOTP issuer is required")
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

// ValidateTOTPEncryptionKey validates the encryption secret
// before 2FA is enabled or used.

func (c Config) ValidateTOTPEncryptionKey() error {
	key := strings.TrimSpace(c.TOTPEncryptionKey)

	if key == "" {
		return ErrTOTPEncryptionKeyMissing
	}

	if len([]byte(key)) < 32 {
		return ErrTOTPEncryptionKeyTooWeak
	}

	return nil
}
