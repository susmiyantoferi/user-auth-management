package middleware

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/dto"
	"user-auth-exam/internal/response"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/redis/go-redis/v9"
)

func Authentication(cfg *config.JwtConfig, rdsClient *redis.Client) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		authHeader := ctx.GetHeader("Authorization")

		if authHeader == "" {
			ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("unauthorization"))
			ctx.Abort()
			return
		}

		tokenPart := strings.Fields(authHeader)
		if len(tokenPart) != 2 || tokenPart[0] != "Bearer" {
			ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("invalid authorization format"))
			ctx.Abort()
			return
		}

		tokenStr := tokenPart[1]
		jwtSecret := []byte(cfg.SecretKey)
		claim := &dto.TokenClaim{}

		token, err := jwt.ParseWithClaims(tokenStr, claim, func(t *jwt.Token) (any, error) {
			if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
			}
			return jwtSecret, nil
		})

		if err != nil || !token.Valid {
			ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("invalid token"))
			ctx.Abort()
			return
		}

		//cek token in rediss
		key := fmt.Sprintf("jwt:%s", claim.UserID.String())

		redisToken, err := rdsClient.Get(ctx.Request.Context(), key).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("session expired or logged out"))

				ctx.Abort()
				return
			}
			ctx.JSON(http.StatusInternalServerError, response.ErrorResponse("internal server error"))

			ctx.Abort()
			return
		}

		if redisToken != tokenStr {
			ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("invalid session"))
			ctx.Abort()
			return
		}
		ctx.Set("user", claim)
		ctx.Next()
	}

}

func RoleAccessMiddleware(allowedRoles ...string) gin.HandlerFunc {
	return func(ctx *gin.Context) {
		userClaims, exist := ctx.Get("user")
		if !exist {
			ctx.JSON(http.StatusUnauthorized, response.ErrorResponse("user not found"))
			ctx.Abort()
			return
		}

		user := userClaims.(*dto.TokenClaim)
		role := user.Role

		for _, v := range allowedRoles {
			if role == v {
				ctx.Next()
				return
			}
		}

		ctx.JSON(http.StatusForbidden, response.ErrorResponse("insufficient role"))
		ctx.Abort()
	}
}
