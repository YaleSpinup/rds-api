package common

import (
	"bufio"
	"encoding/json"
	"io"
	"log"
	"os"

	"github.com/pkg/errors"
)

// Config is representation of the configuration data
type Config struct {
	Account       Account
	AccountsMap   map[string]string
	DefaultConfig CommonConfig
	Token         string
	Org           string
}

// Account is the configuration for an individual account
type Account struct {
	Endpoint   string
	ExternalID string
	Akid       string
	Secret     string
	Region     string
	Role       string
}

type CommonConfig struct {
	DefaultSubnetGroup                 string
	DefaultDBParameterGroupName        map[string]string
	DefaultDBClusterParameterGroupName map[string]string
}

// LoadConfig loads the JSON configuration from the specified filename and returns a Config struct
func LoadConfig(filename string) (Config, error) {
	log.Printf("Loading configuration from %s", filename)

	configFile, err := os.Open(filename)
	if err != nil {
		return Config{}, err
	}

	config, err := readConfig(bufio.NewReader(configFile))
	if err != nil {
		return Config{}, err
	}

	if config.Org == "" {
		return Config{}, errors.New("'org' cannot be empty in the config")
	}

	if config.Token == "" {
		return Config{}, errors.New("'token' cannot be empty in the config")
	}

	return config, nil
}

// readConfig decodes the configuration from an io Reader
func readConfig(r io.Reader) (Config, error) {
	var c Config
	if err := json.NewDecoder(r).Decode(&c); err != nil {
		return c, errors.Wrap(err, "unable to decode JSON message")
	}
	return c, nil
}
