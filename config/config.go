package config

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

type Config struct {
	DB DbConfig
}

type DbConfig struct {
	Host     string
	Name     string
	User     string
	Password string
	Port     int
}

func getEnv(key string) string {
	val := os.Getenv(key)
	return val
}

func getEnvAsInt(key string) int {
	val := os.Getenv(key)
	num, err := strconv.ParseInt(val, 10, 32)
	if err != nil {
		return -1
	}
	return int(num)
}

func Load() (*Config, error) {
	return &Config{
		DB: DbConfig{
			Host:     getEnv("DBHOST"),
			Name:     getEnv("DBNAME"),
			User:     getEnv("DBUSER"),
			Port:     getEnvAsInt("DBPORT"),
			Password: getEnv("DBPASS"),
		},
	}, nil
}

func (c *Config) DatabaseURL() string {
	return fmt.Sprintf(
		"%s:%s@%s:%d/%s",
		c.DB.User,
		strings.ReplaceAll(c.DB.Password, "#", "%23"),
		c.DB.Host,
		c.DB.Port,
		c.DB.Name,
	)
}
