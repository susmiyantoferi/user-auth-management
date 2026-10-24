package route

import (
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/controller"
	"user-auth-exam/internal/middleware"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func NewRouter(users controller.UserController, authentication controller.AuthController, cfgJwt *config.JwtConfig, rdsClient *redis.Client) *gin.Engine {
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOrigins:  []string{"*"},
		AllowMethods:  []string{"GET", "POST", "PUT", "DELETE", "PATCH", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
	}))

	v1 := r.Group("/api/v1")
	{
		auth := v1.Group("/auth")
		{
			//public route
			auth.POST("/register", authentication.Register)
			auth.POST("/login", authentication.Login)
			auth.POST("/refresh", authentication.Refresh)

			protected := auth.Group("")
			protected.Use(middleware.Authentication(cfgJwt, rdsClient))
			{
				protected.POST("/logout", authentication.Logout)
			}
		}

		usr := v1.Group("/users")
		usr.Use(middleware.Authentication(cfgJwt, rdsClient))
		usr.Use(middleware.RoleAccessMiddleware("user", "admin"))
		{
			usr.PATCH("/me", users.Update)
			usr.GET("/me", users.GetByID)
			usr.POST("/me/deactivate", users.Deactive)
			usr.POST("/me/change-email", users.ChangeEmail)

		}
	}

	return r
}
