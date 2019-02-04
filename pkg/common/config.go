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
	Accounts map[string]Account
	Token    string
}

// Account is the configuration for an individual account
type Account struct {
	Region                             string
	Akid                               string
	Secret                             string
	DefaultSubnetGroup                 string
	DefaultDBParameterGroupName        map[string]string
	DefaultDBClusterParameterGroupName map[string]string
}

// LoadConfig loads the JSON configuration from the specified filename and returns a Config struct
func LoadConfig(filename string) (Config, error) {
	log.Printf("Loading configuration from %s", filename)

	configFile, err := os.Open(filename)
	if err != nil {
		log.Println("Unable to open config file", err)
		return Config{}, err
	}

	config, err := readConfig(bufio.NewReader(configFile))
	if err != nil {
		log.Printf("Unable to read configuration from %s. %+v", filename, err)
		return Config{}, err
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
