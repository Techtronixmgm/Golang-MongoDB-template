package auth

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

type RefreshFunc func(
	ctx context.Context,
	refreshToken string,
) (
	accessToken string,
	newRefreshToken string,
	refreshExpiry time.Time,
	err error,
)

func unauthorized(c *gin.Context) {
	c.JSON(http.StatusUnauthorized, gin.H{
		"error": "unauthorized",
	})
	c.Abort()
}

func AuthMiddleware(
	secret string,
	accessCookieName string,
	refreshCookieName string,
	cookieSecure bool,
	accessTokenExpiryHours int,
	refreshToken RefreshFunc,
) gin.HandlerFunc {

	return func(c *gin.Context) {

		accessToken, err := c.Cookie(accessCookieName)

		// Access token is missing.
		// Try to authenticate using the refresh token.
		if err != nil {
			refreshTokenValue, refreshErr := c.Cookie(refreshCookieName)
			if refreshErr != nil {
				unauthorized(c)
				return
			}

			newAccessToken, newRefreshToken, refreshExpiry, refreshErr :=
				refreshToken(
					c.Request.Context(),
					refreshTokenValue,
				)

			if refreshErr != nil {
				unauthorized(c)
				return
			}

			claims, err := ValidateToken(
				newAccessToken,
				secret,
			)
			if err != nil {
				unauthorized(c)
				return
			}

			setAuthCookie(
				c,
				accessCookieName,
				newAccessToken,
				time.Now().UTC().Add(
					time.Duration(accessTokenExpiryHours)*time.Hour,
				),
				cookieSecure,
			)

			setAuthCookie(
				c,
				refreshCookieName,
				newRefreshToken,
				refreshExpiry,
				cookieSecure,
			)

			c.Set("userID", claims.UserID)
			c.Set("role", claims.Role)

			c.Next()
			return
		}

		claims, err := ValidateToken(
			accessToken,
			secret,
		)

		if err != nil {
			// If the token is present but expired, refresh it.
			if !errors.Is(err, jwt.ErrTokenExpired) {
				unauthorized(c)
				return
			}

			refreshTokenValue, refreshErr := c.Cookie(refreshCookieName)
			if refreshErr != nil {
				unauthorized(c)
				return
			}

			newAccessToken, newRefreshToken, refreshExpiry, refreshErr :=
				refreshToken(
					c.Request.Context(),
					refreshTokenValue,
				)

			if refreshErr != nil {
				unauthorized(c)
				return
			}

			claims, err = ValidateToken(
				newAccessToken,
				secret,
			)
			if err != nil {
				unauthorized(c)
				return
			}

			setAuthCookie(
				c,
				accessCookieName,
				newAccessToken,
				time.Now().UTC().Add(
					time.Duration(accessTokenExpiryHours)*time.Hour,
				),
				cookieSecure,
			)

			setAuthCookie(
				c,
				refreshCookieName,
				newRefreshToken,
				refreshExpiry,
				cookieSecure,
			)
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}

func setAuthCookie(
	c *gin.Context,
	name string,
	value string,
	expiresAt time.Time,
	secure bool,
) {
	http.SetCookie(c.Writer, &http.Cookie{
		Name:     name,
		Value:    value,
		Path:     "/",
		Expires:  expiresAt,
		MaxAge:   int(time.Until(expiresAt).Seconds()),
		HttpOnly: true,
		Secure:   secure,
	})
}

func OptionalAuthMiddleware(
	secret string,
	accessCookieName string,
	refreshCookieName string,
	cookieSecure bool,
	accessTokenExpiryHours int,
	refreshToken RefreshFunc,
) gin.HandlerFunc {

	return func(c *gin.Context) {
		accessToken, err := c.Cookie(accessCookieName)

		// No access token.
		// Try refresh, but allow anonymous access if refresh also fails.
		if err != nil {
			refreshTokenValue, refreshErr := c.Cookie(refreshCookieName)
			if refreshErr != nil {
				c.Next()
				return
			}

			newAccessToken, newRefreshToken, refreshExpiry, refreshErr :=
				refreshToken(
					c.Request.Context(),
					refreshTokenValue,
				)

			if refreshErr != nil {
				c.Next()
				return
			}

			claims, err := ValidateToken(
				newAccessToken,
				secret,
			)
			if err != nil {
				c.Next()
				return
			}

			setAuthCookie(
				c,
				accessCookieName,
				newAccessToken,
				time.Now().UTC().Add(
					time.Duration(accessTokenExpiryHours)*time.Hour,
				),
				cookieSecure,
			)

			setAuthCookie(
				c,
				refreshCookieName,
				newRefreshToken,
				refreshExpiry,
				cookieSecure,
			)

			c.Set("userID", claims.UserID)
			c.Set("role", claims.Role)

			c.Next()
			return
		}

		claims, err := ValidateToken(
			accessToken,
			secret,
		)

		if err != nil {
			// Access token is invalid for reasons other than expiry.
			// Treat the request as anonymous.
			if !errors.Is(err, jwt.ErrTokenExpired) {
				c.Next()
				return
			}

			// Access token expired. Try refresh.
			refreshTokenValue, refreshErr := c.Cookie(refreshCookieName)
			if refreshErr != nil {
				c.Next()
				return
			}

			newAccessToken, newRefreshToken, refreshExpiry, refreshErr :=
				refreshToken(
					c.Request.Context(),
					refreshTokenValue,
				)

			if refreshErr != nil {
				c.Next()
				return
			}

			claims, err = ValidateToken(
				newAccessToken,
				secret,
			)
			if err != nil {
				c.Next()
				return
			}

			setAuthCookie(
				c,
				accessCookieName,
				newAccessToken,
				time.Now().UTC().Add(
					time.Duration(accessTokenExpiryHours)*time.Hour,
				),
				cookieSecure,
			)

			setAuthCookie(
				c,
				refreshCookieName,
				newRefreshToken,
				refreshExpiry,
				cookieSecure,
			)
		}

		c.Set("userID", claims.UserID)
		c.Set("role", claims.Role)

		c.Next()
	}
}
