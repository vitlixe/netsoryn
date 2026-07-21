package sshclient

import (
	"strconv"

	"github.com/vitlixe/netsoryn/internal/config"
)

func FindHost(cfg *config.Config, name string) (config.SSHHost, bool) {
	for _, host := range cfg.SSHHosts {
		if host.Name == name {
			return host, true
		}
	}
	return config.SSHHost{}, false
}

func BuildArgs(host config.SSHHost, remote []string) []string {
	args := make([]string, 0, len(host.Options)+len(remote)+6)
	args = append(args, host.Options...)
	if host.Key != "" {
		args = append(args, "-i", host.Key)
	}
	if host.Port > 0 && host.Port != 22 {
		args = append(args, "-p", strconv.Itoa(host.Port))
	}
	args = append(args, Target(host))
	args = append(args, remote...)
	return args
}

func Target(host config.SSHHost) string {
	if host.User == "" {
		return host.Host
	}
	return host.User + "@" + host.Host
}

func RemoteDumpCommand(format, sections string, pretty bool) []string {
	args := []string{"netsoryn", "dump", "--format", format}
	if sections != "" {
		args = append(args, "--sections", sections)
	}
	if pretty {
		args = append(args, "--pretty")
	}
	return args
}
