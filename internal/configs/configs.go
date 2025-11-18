package configs

import(
	"os"
	"strings"
)

type Config struct{
	Port string
	DSN string
	APIKey string
}

func Load()(res Config, err error){
	port := os.Getenv("PORT");
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}
	apiKey := os.Getenv("API_KEY")
	if apiKey == "" {
			apiKey = "aisdjifojwwefojaw342123" 
	}
	dsn := os.Getenv("DB_DSN")
	return Config{Port: port, DSN: dsn, APIKey: apiKey}, nil
}