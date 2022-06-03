package base

// config file struct

import (
	"os"

	yaml "gopkg.in/yaml.v2"
)

// Config is the global configuration
type Config struct {
	Airtable struct {
		APIKey string
		Base   struct {
			Production string
			Test       string
		}
		TableName       string
		Tasks           string
		ActivityInsight string `yaml:"activity_insight"`
	}
	Unpaywall struct {
		Email string
	}
	OpenAccessButton struct {
		Key string
	}
	ScholarSphere struct {
		APIKey     string
		Production string
		Test       string
	} `yaml:"scholarsphere"`
	RMDB struct {
		APIKey     string
		Production string
		Test       string
	} `yaml:"rmdb"`
	ArticlePath string `yaml:"article_path"`
}

func loadConfig(file string) (*Config, error) {
	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	var conf Config
	return &conf, yaml.NewDecoder(f).Decode(&conf)
}
