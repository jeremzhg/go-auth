package configs

import(
	"os"
	"strings"
)

type Config struct{
	Port string
}

func Load()(res Config, err error){
	port := os.Getenv("PORT");
	if port == "" {
		port = "8080"
	}
	if !strings.HasPrefix(port, ":") {
		port = ":" + port
	}

	return Config{Port: port}, nil
}