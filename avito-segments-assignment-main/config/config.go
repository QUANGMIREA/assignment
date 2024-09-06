package config

import (
	"fmt"

	"github.com/ilyakaznacheev/cleanenv"
)

type Config struct {
	UserSegmentator `yaml:"usersegmentator"`
	MySQL           `yaml:"mysql"`
	HTTP            `yaml:"http"`
	Report          `yaml:"report"`
	Segment         `yaml:"segment"`
}

type UserSegmentator struct {
	Name    string `yaml:"name"`
	Version string `yaml:"version"`
}

type MySQL struct {
	Name           string `env-required:"true"  env:"MYSQL_DATABASE"`
	Password       string `env-required:"true"  env:"MYSQL_ROOT_PASSWORD"`
	MaxConnections int    `yaml:"maxConns"`
	Host           string `yaml:"host"`
	Port           string `yaml:"port"`
	Timeout        int    `yaml:"conn_timeout"`
}

type HTTP struct {
	Host string `yaml:"host"`
	Port string `yaml:"port"`
}

type Report struct {
	FilePrefix string `yaml:"file_prefix"`
	FileExt    string `yaml:"file_ext"`
	StorageDir string `env-required:"true"  env:"REPORTS_STORAGE"`
}

type Segment struct {
	TTLCheckInterval int `yaml:"ttl_check_interval"`
}

func NewConfig() (*Config, error) {
	cfg := &Config{}

	err := cleanenv.ReadConfig("./config/config.yml", cfg)
	if err != nil {
		return nil, fmt.Errorf("config error: %w", err)
	}

	err = cleanenv.ReadEnv(cfg)
	if err != nil {
		return nil, err
	}

	return cfg, nil
}
