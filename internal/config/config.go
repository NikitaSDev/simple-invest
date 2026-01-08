package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
)

const timeout int = 5

type Config struct {
	StoragePath string
	Port        string
	Timeout     int
}

func MustLoad() *Config {
	err := godotenv.Load()
	if err != nil {
		log.Print(err)
		panic(err)
	}

	return &Config{
		StoragePath: storagePath(),
		Port:        os.Getenv("APP_PORT"),
		Timeout:     timeout,
	}
}

func storagePath() string {
	return fmt.Sprintf("user=%s password=%s dbname=%s sslmode=%s",
		os.Getenv("DB_USER"),
		os.Getenv("DB_PASSWORD"),
		os.Getenv("DB_NAME"),
		os.Getenv("DB_SSLMODE"),
	)
}
