package sdk

import (
	"io/ioutil"
	"net"
	"os"
	"path/filepath"

	"github.com/docker/go-connections/sockets"
)

const (
	pluginSpecDir = "/etc/docker/plugins"
)

func newTCPListener(address string, pluginName string) (net.Listener, string, error) {
	listener, err := sockets.NewTCPSocket(address, nil)
	if err != nil {
		return nil, "", err
	}
	spec, err := writeSpec(pluginName, listener.Addr().String())
	if err != nil {
		return nil, "", err
	}
	return listener, spec, nil
}

func writeSpec(name string, address string) (string, error) {
	if err := os.MkdirAll(pluginSpecDir, 0755); err != nil {
		return "", err
	}
	spec := filepath.Join(pluginSpecDir, name+".spec")
	url := "tcp://" + address
	if err := ioutil.WriteFile(spec, []byte(url), 0644); err != nil {
		return "", err
	}
	return spec, nil
}
