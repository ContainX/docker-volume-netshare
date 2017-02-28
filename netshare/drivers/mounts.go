package drivers

import (
	"errors"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"strings"
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

type mountManager struct {
	mounts map[string]*mount
}

func NewVolumeManager() *mountManager {
	return &mountManager{
		mounts: map[string]*mount{},
	}
}

func (m *mountManager) HasMount(name string) bool {
	_, found := m.mounts[name]
	return found
}

func (m *mountManager) HasOptions(name string) bool {
	c, found := m.mounts[name]
	if found {
		return c.opts != nil && len(c.opts) > 0
	}
	return false
}

func (m *mountManager) HasOption(name, key string) bool {
	if m.HasOptions(name) {
		if _, ok := m.mounts[name].opts[key]; ok {
			return ok
		}
	}
	return false
}

func (m *mountManager) GetOptions(name string) map[string]string {
	if m.HasOptions(name) {
		c, _ := m.mounts[name]
		return c.opts
	}
	return map[string]string{}
}

func (m *mountManager) GetOption(name, key string) string {
	if m.HasOption(name, key) {
		v, _ := m.mounts[name].opts[key]
		return v
	}
	return ""
}

func (m *mountManager) GetOptionAsBool(name, key string) bool {
	rv := strings.ToLower(m.GetOption(name, key))
	if rv == "yes" || rv == "true" {
		return true
	}
	return false
}

func (m *mountManager) IsActiveMount(name string) bool {
	c, found := m.mounts[name]
	return found && c.connections > 0
}

func (m *mountManager) Count(name string) int {
	c, found := m.mounts[name]
	if found {
		return c.connections
	}
	return 0
}

func (m *mountManager) Add(name, hostdir string) {
	_, found := m.mounts[name]
	if found {
		m.Increment(name)
	} else {
		m.mounts[name] = &mount{name: name, hostdir: hostdir, managed: false, connections: 1}
	}
}

func (m *mountManager) Create(name, hostdir string, opts map[string]string) *mount {
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

func (m *mountManager) Delete(name string) error {
	log.Debugf("Delete volume: %s, connections: %d", name, m.Count(name))
	if m.HasMount(name) {
		if m.Count(name) < 1 {
			delete(m.mounts, name)
			return nil
		}
		return errors.New("Volume is currently in use")
	}
	return nil
}

func (m *mountManager) DeleteIfNotManaged(name string) error {
	if m.HasMount(name) && !m.IsActiveMount(name) && !m.mounts[name].managed {
		log.Infof("Removing un-managed volume")
		return m.Delete(name)
	}
	return nil
}

func (m *mountManager) Increment(name string) int {
	c, found := m.mounts[name]
	if found {
		c.connections++
		return c.connections
	}
	return 0
}

func (m *mountManager) Decrement(name string) int {
	c, found := m.mounts[name]
	if found && c.connections > 0 {
		c.connections--
	}
	return 0
}

func (m *mountManager) GetVolumes(rootPath string) []*volume.Volume {

	volumes := []*volume.Volume{}

	for _, mount := range m.mounts {
		volumes = append(volumes, &volume.Volume{Name: mount.name, Mountpoint: mount.hostdir})
	}
	return volumes
}
