package middleware

import (
	"slices"
	"time"

	"github.com/gin-gonic/gin"
)

type CORSConfig struct {
	AllowedOrigins   []string
	AllowedMethods   []string
	AllowedHeaders   []string
	AllowCredentials bool
	MaxAge           time.Duration
}

func CORSMiddleware(config CORSConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		origin := c.Request.Header.Get("Origin")
		
		// Check if origin is allowed
		if len(config.AllowedOrigins) > 0 && config.AllowedOrigins[0] != "*" {
			allowed := slices.Contains(config.AllowedOrigins, origin)
			if !allowed {
				c.Next()
				return
			}
		}

		// Set CORS headers
		if len(config.AllowedOrigins) > 0 {
			if config.AllowedOrigins[0] == "*" {
				c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
			} else {
				c.Writer.Header().Set("Access-Control-Allow-Origin", origin)
			}
		}

		if len(config.AllowedMethods) > 0 {
			methods := ""
			for i, method := range config.AllowedMethods {
				if i > 0 {
					methods += ", "
				}
				methods += method
			}
			c.Writer.Header().Set("Access-Control-Allow-Methods", methods)
		}

		if len(config.AllowedHeaders) > 0 {
			headers := ""
			for i, header := range config.AllowedHeaders {
				if i > 0 {
					headers += ", "
				}
				headers += header
			}
			c.Writer.Header().Set("Access-Control-Allow-Headers", headers)
		}

		if config.AllowCredentials {
			c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")
		}

		if config.MaxAge > 0 {
			c.Writer.Header().Set("Access-Control-Max-Age", config.MaxAge.String())
		}

		// Handle preflight requests
		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}