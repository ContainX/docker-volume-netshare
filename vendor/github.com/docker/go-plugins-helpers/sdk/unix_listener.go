// +build linux freebsd

package sdk

import (
	"net"
	"os"
	"path/filepath"

	"github.com/docker/go-connections/sockets"
)

const (
	pluginSockDir = "/run/docker/plugins"
)

func newUnixListener(pluginName string, group string) (net.Listener, string, error) {
	path, err := fullSocketAddress(pluginName)
	if err != nil {
		return nil, "", err
	}
	listener, err := sockets.NewUnixSocket(path, group)
	if err != nil {
		return nil, "", err
	}
	return listener, path, nil
}

func fullSocketAddress(address string) (string, error) {
	if err := os.MkdirAll(pluginSockDir, 0755); err != nil {
		return "", err
	}
	if filepath.IsAbs(address) {
		return address, nil
	}
	return filepath.Join(pluginSockDir, address+".sock"), nil
}
