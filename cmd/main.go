package main

import (
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/infrastructure/datastore"
	"user-auth-exam/infrastructure/logger"
	"user-auth-exam/internal/controller"
	"user-auth-exam/internal/repository"
	"user-auth-exam/internal/route"
	"user-auth-exam/internal/service"

	"github.com/go-playground/validator/v10"
)

func main() {
	config, err := config.NewViper()
	if err != nil {
		panic(err)
	}

	log := logger.NewLogrus(&config.Logger)
	db := datastore.NewDatabase(&config.Postgres)
	redis := datastore.NewRedis(&config.Redis)
	validate := validator.New()

	userRepo := repository.NewUserRepositoryImpl()

	userService := service.NewUserServiceImpl(userRepo, db, validate, log)
	authService := service.NewAuthServiceImpl(userRepo, db, validate, log, &config.Jwt, redis)

	userController := controller.NewUserControllerImpl(userService)
	authController := controller.NewAuthControllerImpl(authService)

	router := route.NewRouter(userController, authController)

	router.Run(config.App.Port)
}
