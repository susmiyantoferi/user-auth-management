package route

import (
	"user-auth-exam/internal/controller"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func NewRouter(users controller.UserController, authentication controller.AuthController) *gin.Engine {
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
			auth.POST("/register", authentication.Register)
			auth.POST("/login", authentication.Login)
		}

		usr := v1.Group("/users")
		{
			usr.PATCH("/:id", users.Update)
			usr.GET("/:id", users.GetByID)
			usr.POST("/:id", users.Deactive)
			usr.POST("/:id/change-email", users.ChangeEmail)

		}
	}

	return r
}
