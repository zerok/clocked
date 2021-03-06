package config

import (
	"io/ioutil"
	"os"

	"gopkg.in/yaml.v2"
)

type Config struct {
	BackupsPath  string `yaml:"backups_path"`
	JIRAUsername string `yaml:"jira_username"`
	JIRAURL      string `yaml:"jira_url"`
	JIRAPassword string
}

func Load(path string) (*Config, error) {
	var c Config
	raw, err := ioutil.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return &c, nil
		}
		return nil, err
	}
	if err := yaml.Unmarshal(raw, &c); err != nil {
		return nil, err
	}
	if c.JIRAURL != "" {
		pwd, err := loadJIRAPassword(c.JIRAURL, c.JIRAUsername)
		if err != nil {
			return nil, err
		}
		c.JIRAPassword = pwd
	}
	return &c, nil
}
