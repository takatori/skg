package internal

import (
	"github.com/kelseyhightower/envconfig"
)

type Config struct {
	Env      RunEnv `envconfig:"ENV" default:"development"`
	EchoAddr string `envconfig:"ECHO_ADDR" default:":8080"`
	SolrUrl  string `envconfig:"SOLR_URL" default:"http://solr:8983/solr"`
}

func LoadConfig() (*Config, error) {
	var cfg Config
	err := envconfig.Process("", &cfg)
	if err != nil {
		return nil, err
	}
	return &cfg, nil
}
