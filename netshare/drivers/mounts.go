package drivers

import (
	"context"
	"errors"
	"strings"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

const (
	ShareOpt  = "share"
	CreateOpt = "create"
)

type mount struct {
	name        string
	hostdir     string
	connections int
	opts        map[string]string
	managed     bool
}

type MountManager struct {
	mounts map[string]*mount
}

func NewVolumeManager() *MountManager {
	return &MountManager{
		mounts: map[string]*mount{},
	}
}

func (m *MountManager) HasMount(name string) bool {
	_, found := m.mounts[name]
	return found
}

func (m *MountManager) HasOptions(name string) bool {
	c, found := m.mounts[name]
	if found {
		return c.opts != nil && len(c.opts) > 0
	}
	return false
}

func (m *MountManager) HasOption(name, key string) bool {
	if m.HasOptions(name) {
		if _, ok := m.mounts[name].opts[key]; ok {
			return ok
		}
	}
	return false
}

func (m *MountManager) GetOptions(name string) map[string]string {
	if m.HasOptions(name) {
		c, _ := m.mounts[name]
		return c.opts
	}
	return map[string]string{}
}

func (m *MountManager) GetOption(name, key string) string {
	if m.HasOption(name, key) {
		v, _ := m.mounts[name].opts[key]
		return v
	}
	return ""
}

func (m *MountManager) GetOptionAsBool(name, key string) bool {
	rv := strings.ToLower(m.GetOption(name, key))
	if rv == "yes" || rv == "true" {
		return true
	}
	return false
}

func (m *MountManager) IsActiveMount(name string) bool {
	c, found := m.mounts[name]
	return found && c.connections > 0
}

func (m *MountManager) Count(name string) int {
	c, found := m.mounts[name]
	if found {
		return c.connections
	}
	return 0
}

func (m *MountManager) Add(name, hostdir string) {
	_, found := m.mounts[name]
	if found {
		m.Increment(name)
	} else {
		m.mounts[name] = &mount{name: name, hostdir: hostdir, managed: false, connections: 1}
	}
}

func (m *MountManager) Create(name, hostdir string, opts map[string]string) *mount {
	c, found := m.mounts[name]
	if found && c.connections > 0 {
		c.opts = opts
		return c
	} else {
		mnt := &mount{name: name, hostdir: hostdir, managed: true, opts: opts, connections: 0}
		m.mounts[name] = mnt
		return mnt
	}
}

func (m *MountManager) Delete(name string) error {
	// Check if any stopped containers are having references with volume.
	refCount := checkReferences(name)
	log.Debugf("Reference count %d", refCount)
	if m.HasMount(name) {
		if m.Count(name) < 1 && refCount < 1 {
			log.Debugf("Delete volume: %s, connections: %d", name, m.Count(name))
			delete(m.mounts, name)
			return nil
		}
		return errors.New("Volume is currently in use")
	}
	return nil
}

func (m *MountManager) DeleteIfNotManaged(name string) error {
	if m.HasMount(name) && !m.IsActiveMount(name) && !m.mounts[name].managed {
		log.Infof("Removing un-managed volume")
		return m.Delete(name)
	}
	return nil
}

func (m *MountManager) Increment(name string) int {
	log.Infof("Incrementing for %s", name)
	c, found := m.mounts[name]
	log.Infof("Previous connections state : %d", c.connections)
	if found {
		c.connections++
		log.Infof("Current connections state : %d", c.connections)
		return c.connections
	}
	return 0
}

func (m *MountManager) Decrement(name string) int {
	log.Infof("Decrementing for %s", name)
	c, found := m.mounts[name]
	log.Infof("Previous connections state : %d", c.connections)
	if found && c.connections > 0 {
		c.connections--
		log.Infof("Current connections state :  %d", c.connections)
	}
	return 0
}

func (m *MountManager) GetVolumes(rootPath string) []*volume.Volume {

	volumes := []*volume.Volume{}

	for _, mount := range m.mounts {
		volumes = append(volumes, &volume.Volume{Name: mount.name, Mountpoint: mount.hostdir})
	}
	return volumes
}

func (m *MountManager) AddMount(name string, hostdir string, connections int) {
	m.mounts[name] = &mount{name: name, hostdir: hostdir, managed: true, connections: connections}
}

//Checking volume references with started and stopped containers as well.
func checkReferences(volumeName string) int {

	cli, err := client.NewEnvClient()
	if err != nil {
		log.Error(err)
	}

	var counter = 0
	ContainerListResponse, err := cli.ContainerList(context.Background(), types.ContainerListOptions{All: true}) // All : true will return the stopped containers as well.
	if err != nil {
		log.Fatal(err, ". Use -a flag to setup the DOCKER_API_VERSION. Run 'docker-volume-netshare --help' for usage.")
	}

	for _, container := range ContainerListResponse {
		if len(container.Mounts) == 0 {
			continue
		}
		for _, mounts := range container.Mounts {
			if !(mounts.Name == volumeName) {
				continue
			}
			counter++
		}
	}
	return counter
}
