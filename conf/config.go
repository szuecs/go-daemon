// Package conf handles the configuration of the applications. Yaml
// files are mapped with the struct
package conf

import (
	"fmt"
	"io/ioutil"
	"net/url"
	"os"
	"time"

	"gopkg.in/yaml.v1"

	"github.com/golang/glog"
)

// Config is the configuration struct. The config file config.yaml
// will unmarshaled to this struct.
type Config struct {
	DebugEnabled     bool          `yaml:"debug_enabled,omitempty"`
	Oauth2Enabled    bool          `yaml:"oauth2_enabled,omitempty"`
	ProfilingEnabled bool          `yaml:"profiling_enabled,omitempty"`
	LogFlushInterval time.Duration `yaml:"log_flush_interval,omitempty"`
	URL              string        `yaml:"url,omitempty"`
	RealURL          *url.URL      //RealURL to our service endpoint parsed from URL
	AuthURL          string        `yaml:"auth_url,omitempty"`
	TokenURL         string        `yaml:"token_url,omitempty"`
	Username         string        `yaml:"user,omitempty"`
}

// shared state for configuration
var conf *Config

// New returns the loaded configuration or panic
func New() (*Config, error) {
	var err error
	if conf == nil {
		conf, err = configInit("config.yaml")
	}
	return conf, err
}

// PROJECTNAME TODO: should be replaced in your application
const PROJECTNAME string = "go-daemon"

func readFile(filepath string) ([]byte, bool) {
	b, err := ioutil.ReadFile(filepath)
	if err != nil {
		return b, false
	}
	return b, true
}

// FIXME: not windows compatible
func configInit(filename string) (*Config, error) {
	globalConfig := fmt.Sprintf("/etc/%s/config.yaml", PROJECTNAME)
	homeConfig := fmt.Sprintf("%s/.config/%s/config.yaml", os.ExpandEnv("$HOME"), PROJECTNAME)
	b, ok := readFile(homeConfig)
	if !ok {
		b, ok = readFile(globalConfig)
	}
	if !ok {
		return nil, fmt.Errorf("No file readable in %v nor in %v", globalConfig, homeConfig)
	}
	var config Config
	err := yaml.Unmarshal(b, &config)
	if err != nil {
		glog.Fatalf("configuration could not be unmarshaled, caused by: %s", err)
	}
	return &config, err
}
