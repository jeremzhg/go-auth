package configs

import(
	"os"
	"strings"
)

type Config struct{
	Port string
	DSN string
}

func Load()(res Config, err error){
	port := os.Getenv("PORT");
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	dsn := os.Getenv("DB_DSN")
	return Config{Port: port, DSN: dsn}, nil
}