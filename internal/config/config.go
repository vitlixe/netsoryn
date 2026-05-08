package config

import (
	"os"
	"path/filepath"
	"time"

	"github.com/spf13/viper"
)

type HTTPCheck struct {
	URL     string        `mapstructure:"url"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type DNSCheck struct {
	Domain  string   `mapstructure:"domain"`
	Servers []string `mapstructure:"servers"`
}

type Config struct {
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	DefaultView     string        `mapstructure:"default_view"`
	LogLevel        string        `mapstructure:"log_level"`
	LogFile         string        `mapstructure:"log_file"`
	HTTPChecks      []HTTPCheck   `mapstructure:"http_checks"`
	DNSChecks       []DNSCheck    `mapstructure:"dns_checks"`
	DockerSocket    string        `mapstructure:"docker_socket"`
	ProcessLimit    int           `mapstructure:"process_limit"`
	PortsListenOnly bool          `mapstructure:"ports_listen_only"`
}

func Load(cfgFile string) (*Config, error) {
	v := viper.New()

	v.SetDefault("refresh_interval", 2)
	v.SetDefault("default_view", "dashboard")
	v.SetDefault("log_level", "disabled")
	v.SetDefault("log_file", "")
	v.SetDefault("process_limit", 50)
	v.SetDefault("ports_listen_only", true)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "netsoryn"))
		}
		v.AddConfigPath("/etc/netsoryn")
	}

	v.SetEnvPrefix("NETSORYN")
	v.AutomaticEnv()

	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, err
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, err
	}

	if cfg.RefreshInterval < time.Second {
		cfg.RefreshInterval = cfg.RefreshInterval * time.Second
	}

	return &cfg, nil
}
