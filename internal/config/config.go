package config

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/mitchellh/mapstructure"
	"github.com/spf13/viper"
	"gopkg.in/yaml.v3"
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

type SSHHost struct {
	Name    string   `mapstructure:"name"`
	Host    string   `mapstructure:"host"`
	User    string   `mapstructure:"user"`
	Port    int      `mapstructure:"port"`
	Key     string   `mapstructure:"key"`
	Options []string `mapstructure:"options"`
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
	SSHHosts        []SSHHost     `mapstructure:"ssh_hosts"`
	DockerSocket    string        `mapstructure:"docker_socket"`
	ProcessLimit    int           `mapstructure:"process_limit"`
	PortsListenOnly bool          `mapstructure:"ports_listen_only"`
	ConfigFile      string        `mapstructure:"-"`
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
	cfg.ConfigFile = v.ConfigFileUsed()

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
	seenSSHHosts := make(map[string]int, len(cfg.SSHHosts))
	for i := range cfg.SSHHosts {
		host := &cfg.SSHHosts[i]
		normalizeSSHHost(host)
		if host.Name == "" {
			return fmt.Errorf("ssh_hosts[%d].name must not be empty", i)
		}
		if host.Host == "" {
			return fmt.Errorf("ssh_hosts[%d].host must not be empty", i)
		}
		if prev, ok := seenSSHHosts[host.Name]; ok {
			return fmt.Errorf("ssh_hosts[%d].name duplicates ssh_hosts[%d].name %q", i, prev, host.Name)
		}
		seenSSHHosts[host.Name] = i
		if host.Port == 0 {
			host.Port = 22
		}
		if host.Port < 0 || host.Port > 65535 {
			return fmt.Errorf("ssh_hosts[%d].port must be between 1 and 65535, got %d", i, host.Port)
		}
	}
	return nil
}

func normalizeSSHHost(host *SSHHost) {
	host.Name = strings.TrimSpace(host.Name)
	host.Host = strings.TrimSpace(host.Host)
	host.User = strings.TrimSpace(host.User)
	host.Key = strings.TrimSpace(host.Key)
	options := host.Options[:0]
	for _, opt := range host.Options {
		opt = strings.TrimSpace(opt)
		if opt != "" {
			options = append(options, opt)
		}
	}
	host.Options = options
}

func AddSSHHost(cfg *Config, host SSHHost) (string, error) {
	next := append([]SSHHost(nil), cfg.SSHHosts...)
	next = append(next, host)

	updated := *cfg
	updated.SSHHosts = next
	if err := validateConfig(&updated); err != nil {
		return "", err
	}

	path := cfg.ConfigFile
	if path == "" {
		var err error
		path, err = defaultUserConfigFile()
		if err != nil {
			return "", err
		}
	}

	doc, err := readConfigNode(path)
	if err != nil {
		return "", err
	}
	if err := setSSHHostsNode(doc, updated.SSHHosts); err != nil {
		return "", err
	}
	out, err := yaml.Marshal(doc)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return "", err
	}
	if err := os.WriteFile(path, out, 0o644); err != nil {
		return "", err
	}

	cfg.SSHHosts = updated.SSHHosts
	cfg.ConfigFile = path
	return path, nil
}

func defaultUserConfigFile() (string, error) {
	if dir, err := os.UserConfigDir(); err == nil && dir != "" {
		return filepath.Join(dir, "netsoryn", "config.yaml"), nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".config", "netsoryn", "config.yaml"), nil
}

func readConfigNode(path string) (*yaml.Node, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return emptyConfigNode(), nil
		}
		return nil, err
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return emptyConfigNode(), nil
	}

	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, err
	}
	if doc.Kind == 0 || len(doc.Content) == 0 {
		return emptyConfigNode(), nil
	}
	return &doc, nil
}

func emptyConfigNode() *yaml.Node {
	return &yaml.Node{
		Kind: yaml.DocumentNode,
		Content: []*yaml.Node{{
			Kind: yaml.MappingNode,
		}},
	}
}

func setSSHHostsNode(doc *yaml.Node, hosts []SSHHost) error {
	if doc.Kind != yaml.DocumentNode {
		return fmt.Errorf("config root must be a YAML document")
	}
	if len(doc.Content) == 0 {
		doc.Content = []*yaml.Node{{Kind: yaml.MappingNode}}
	}
	root := doc.Content[0]
	if root.Kind != yaml.MappingNode {
		return fmt.Errorf("config document must be a YAML mapping")
	}

	value := sshHostsNode(hosts)
	for i := 0; i+1 < len(root.Content); i += 2 {
		if root.Content[i].Value == "ssh_hosts" {
			root.Content[i+1] = value
			return nil
		}
	}
	root.Content = append(root.Content, scalarNode("ssh_hosts"), value)
	return nil
}

func sshHostsNode(hosts []SSHHost) *yaml.Node {
	seq := &yaml.Node{Kind: yaml.SequenceNode}
	for _, host := range hosts {
		seq.Content = append(seq.Content, sshHostNode(host))
	}
	return seq
}

func sshHostNode(host SSHHost) *yaml.Node {
	fields := []*yaml.Node{
		scalarNode("name"), scalarNode(host.Name),
		scalarNode("host"), scalarNode(host.Host),
	}
	if host.User != "" {
		fields = append(fields, scalarNode("user"), scalarNode(host.User))
	}
	if host.Port > 0 {
		fields = append(fields, scalarNode("port"), intNode(host.Port))
	}
	if host.Key != "" {
		fields = append(fields, scalarNode("key"), scalarNode(host.Key))
	}
	if len(host.Options) > 0 {
		options := &yaml.Node{Kind: yaml.SequenceNode}
		for _, opt := range host.Options {
			options.Content = append(options.Content, scalarNode(opt))
		}
		fields = append(fields, scalarNode("options"), options)
	}
	return &yaml.Node{Kind: yaml.MappingNode, Content: fields}
}

func scalarNode(value string) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value}
}

func intNode(value int) *yaml.Node {
	return &yaml.Node{Kind: yaml.ScalarNode, Tag: "!!int", Value: strconv.Itoa(value)}
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
