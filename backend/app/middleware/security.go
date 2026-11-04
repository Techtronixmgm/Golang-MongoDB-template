package middleware

import (
	"os"

	"github.com/gin-gonic/gin"
	"github.com/unrolled/secure"
)

func SecurityMiddleware() gin.HandlerFunc {
	isDev := os.Getenv("ENV") == "development" || os.Getenv("APP_ENV") == "development"

	sec := secure.New(secure.Options{
		// Prevent browsers from MIME-sniffing the content type
		ContentTypeNosniff: true,

		// Prevent clickjacking via iframes
		FrameDeny: true,

		// Referrer policy
		ReferrerPolicy: "strict-origin-when-cross-origin",

		// HTTPS configuration.
		IsDevelopment: isDev,
		SSLRedirect:   !isDev,

		// HSTS configuration.
		STSSeconds:           31536000,
		STSIncludeSubdomains: false,
		STSPreload:           false,
	})

	return func(c *gin.Context) {
		if err := sec.Process(c.Writer, c.Request); err != nil {
			c.Abort()
			return
		}

		c.Next()
	}
}
