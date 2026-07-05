package config

import "os"

const DefaultServiceKey = "dev-service-key"

type Config struct {
	Port        string
	DatabaseURL string
	ServiceKey  string
	JWTSecret   string
}

func Load() *Config {

	port := "8081"
	portEnv := os.Getenv("port")
	if portEnv != "" {
		port = portEnv
	}

	dbUrlEnv := os.Getenv("DATABASE_URL")
	dbUrl := "postgres://postgres:postgres@localhost:5432/bookshelf?sslmode=disable"
	if dbUrlEnv != "" {
		dbUrl = dbUrlEnv
	}

	jwtEnv := os.Getenv("JWT_SECRET")
	jwtSecret := "0a6876f139eea1103c5d74dd72f09c92fe8d00e4806bb6f8006d596a139b506c"
	if jwtEnv != "" {
		jwtSecret = jwtEnv
	}

	svcSecretEnv := os.Getenv("SERVICE_KEY")
	svcSecret := DefaultServiceKey
	if svcSecretEnv != "" {
		svcSecret = svcSecretEnv
	}
	return &Config{
		Port:        port,
		DatabaseURL: dbUrl,
		ServiceKey:  svcSecret,
		JWTSecret:   jwtSecret,
	}
}
