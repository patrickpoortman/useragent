package useragent

import (
	"gopkg.in/yaml.v3"
	"os"
)

type BadBotsList struct {
	Bots []string `yaml:"nginx_bad_agents_default"`
}

// LoadBadBots loads the list of bad bot substrings from a YAML file.
func LoadBadBots(path string) ([]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	var list BadBotsList
	dec := yaml.NewDecoder(f)
	if err := dec.Decode(&list); err != nil {
		return nil, err
	}
	return list.Bots, nil
}
