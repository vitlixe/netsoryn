package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"time"

	"github.com/mitchellh/mapstructure"
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

type TCPCheck struct {
	Target  string        `mapstructure:"target"`
	Timeout time.Duration `mapstructure:"timeout"`
}

type Config struct {
	RefreshInterval time.Duration `mapstructure:"refresh_interval"`
	DefaultView     string        `mapstructure:"default_view"`
	LogLevel        string        `mapstructure:"log_level"`
	LogFile         string        `mapstructure:"log_file"`
	HTTPChecks      []HTTPCheck   `mapstructure:"http_checks"`
	HTTPTimeout     time.Duration `mapstructure:"http_timeout"`
	DNSChecks       []DNSCheck    `mapstructure:"dns_checks"`
	TCPChecks       []TCPCheck    `mapstructure:"tcp_checks"`
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
	v.SetDefault("http_timeout", 10)

	if cfgFile != "" {
		v.SetConfigFile(cfgFile)
	} else {
		v.SetConfigName("config")
		v.SetConfigType("yaml")
		v.AddConfigPath(".")
		if home, err := os.UserHomeDir(); err == nil {
			v.AddConfigPath(filepath.Join(home, ".config", "netsoryn"))
		}
		if cfgDir, err := os.UserConfigDir(); err == nil {
			v.AddConfigPath(filepath.Join(cfgDir, "netsoryn"))
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
	if err := v.Unmarshal(&cfg, viper.DecodeHook(
		mapstructure.ComposeDecodeHookFunc(
			durationDecodeHook(),
			mapstructure.StringToSliceHookFunc(","),
		),
	)); err != nil {
		return nil, err
	}

	if err := validateConfig(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

func validateConfig(cfg *Config) error {
	if cfg.RefreshInterval <= 0 {
		return fmt.Errorf("refresh_interval must be positive, got %v", cfg.RefreshInterval)
	}
	if cfg.HTTPTimeout < 0 {
		return fmt.Errorf("http_timeout must be non-negative, got %v", cfg.HTTPTimeout)
	}
	for i, check := range cfg.HTTPChecks {
		if check.Timeout < 0 {
			return fmt.Errorf("http_checks[%d].timeout must be non-negative, got %v", i, check.Timeout)
		}
	}
	for i, check := range cfg.TCPChecks {
		if check.Timeout < 0 {
			return fmt.Errorf("tcp_checks[%d].timeout must be non-negative, got %v", i, check.Timeout)
		}
	}
	return nil
}

// durationDecodeHook converts numeric YAML values to time.Duration treating
// them as seconds, and string values using time.ParseDuration ("500ms", "2s").
func durationDecodeHook() mapstructure.DecodeHookFuncType {
	durationType := reflect.TypeOf(time.Duration(0))

	return func(from reflect.Type, to reflect.Type, data interface{}) (interface{}, error) {
		if to != durationType {
			return data, nil
		}

		switch value := data.(type) {
		case string:
			if value == "" {
				return time.Duration(0), nil
			}
			d, err := time.ParseDuration(value)
			if err != nil {
				return nil, fmt.Errorf("invalid duration %q: %w", value, err)
			}
			return d, nil
		case int:
			return time.Duration(value) * time.Second, nil
		case int8:
			return time.Duration(value) * time.Second, nil
		case int16:
			return time.Duration(value) * time.Second, nil
		case int32:
			return time.Duration(value) * time.Second, nil
		case int64:
			return time.Duration(value) * time.Second, nil
		case uint:
			return time.Duration(value) * time.Second, nil
		case uint8:
			return time.Duration(value) * time.Second, nil
		case uint16:
			return time.Duration(value) * time.Second, nil
		case uint32:
			return time.Duration(value) * time.Second, nil
		case uint64:
			const maxSec = uint64(math.MaxInt64) / uint64(time.Second)
			if value > maxSec {
				return nil, fmt.Errorf("duration value %d seconds overflows", value)
			}
			return time.Duration(value) * time.Second, nil
		case float32:
			return durationFromFloat(float64(value))
		case float64:
			return durationFromFloat(value)
		default:
			return data, nil
		}
	}
}

func durationFromFloat(v float64) (time.Duration, error) {
	if math.IsInf(v, 0) || math.IsNaN(v) {
		return 0, fmt.Errorf("invalid duration value: %v", v)
	}
	const maxSec = float64(math.MaxInt64) / float64(time.Second)
	if v > maxSec {
		return 0, fmt.Errorf("duration value %v seconds overflows", v)
	}
	return time.Duration(v * float64(time.Second)), nil
}
