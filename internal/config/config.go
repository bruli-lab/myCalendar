package config

import "github.com/caarlos0/env/v11"

type Config struct {
	ClientID     string `env:"CLIENT_ID,required"`
	ClientSecret string `env:"CLIENT_SECRET,required"`
	CallbackHost string `env:"CALLBACK_HOST,required"`
	CallbackURL  string `env:"CALLBACK_URL,required"`
	TokensFile   string `env:"TOKENS_FILE,required"`
	ServerHost   string `env:"SERVER_HOST,required"`
	LogLevel     string `env:"LOG_LEVEL,required"`
	NtfyURL      string `env:"NTFY_URL,required"`
	NtfyTopic    string `env:"NTFY_TOPIC,required"`
	NtfyUser     string `env:"NTFY_USER,required"`
	NtfyPassword string `env:"NTFY_PASSWORD,required"`
}

func New() (*Config, error) {
	var cfg Config
	if err := env.Parse(&cfg); err != nil {
		return nil, err
	}
	return &cfg, nil
}
