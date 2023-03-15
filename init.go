package main

import (
	"encoding/json"
	"fmt"
	"os"

	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// InitConfig initializing the config
func InitConfig() Config {

	var config Config

	file, err := os.Open("config.json")
	if err != nil {
		panic(err)
	}

	decoder := json.NewDecoder(file)

	err = decoder.Decode(&config)
	if err != nil {
		panic(err)
	}

	return config
}

// InitDB initializing DB connection
func InitDB(config Config) *gorm.DB {

	connStr := fmt.Sprintf("host=%s port=%d user=%s dbname=%s password=%s sslmode=disable", config.DB.Host, config.DB.Port, config.DB.User, config.DB.Name, config.DB.Password)

	db, err := gorm.Open(postgres.Open(connStr), &gorm.Config{
		DisableForeignKeyConstraintWhenMigrating: true,
	})

	if err != nil {
		panic(err)
	}
	fmt.Println("db connected successfully")
	return db.Debug()
}
