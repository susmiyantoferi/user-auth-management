package datastore

import (
	"fmt"
	"time"
	"user-auth-exam/infrastructure/config"
	"user-auth-exam/internal/entity"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDatabase(config *config.PostgresConfig) *gorm.DB {
	host := config.Host
	port := config.Port
	username := config.Username
	password := config.Password
	dbname := config.DbName
	idleConnect := config.IdleConnect
	maxConnect := config.MaxConnect
	lifeConnect := config.LifeConnect

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=Asia/Jakarta",
		host, username, password, dbname, port)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		panic(err)
	}

	connection, err := db.DB()
	if err != nil {
		panic(err)
	}

	connection.SetMaxIdleConns(idleConnect)
	connection.SetMaxOpenConns(maxConnect)
	connection.SetConnMaxLifetime(time.Second * time.Duration(lifeConnect))

	db.Exec(`CREATE EXTENSION IF NOT EXISTS "uuid-ossp";`)
	err = db.AutoMigrate(
		entity.User{},
	)
	if err != nil {
		panic(err)
	}

	return db
}
