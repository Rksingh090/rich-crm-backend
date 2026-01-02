package config

import (
	"log"
	"os"

	"github.com/joho/godotenv"
)

type Config struct {
	Port        string
	JWTSecret   string
	MongoURI    string
	DBName      string
	SkipAuth    bool
	Environment string
	AppId       string
	FSPath      string // Physical directory for file uploads
	FSURL       string // URL path prefix for file access
}

// LoadConfig loads configuration from environment variables
func LoadConfig() (*Config, error) {
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found")
	} else {
		log.Println("Loaded .env file successfully")
	}

	return &Config{
		Port:        getEnv("PORT", "8080"),
		JWTSecret:   getEnv("JWT_SECRET", "secret"),
		MongoURI:    getEnv("MONGO_URI", "mongodb://localhost:27017"),
		DBName:      getEnv("DB_NAME", "go-crm"),
		SkipAuth:    getEnv("SKIP_AUTH", "false") == "true",
		Environment: getEnv("ENVIRONMENT", "development"),
		AppId:       getEnv("APP_ID", "go-crm"),
		FSPath:      getEnv("FS_PATH", "./uploads"),
		FSURL:       getEnv("FS_URL", "/fs/uploads"),
	}, nil
}

func getEnv(key, fallback string) string {
	if value, exists := os.LookupEnv(key); exists {
		return value
	}
	return fallback
}
