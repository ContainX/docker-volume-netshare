package drivers

import (
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"sync"
)

type volumeDriver struct {
	root   string
	mountm *mountManager
	m      *sync.Mutex
}

func newVolumeDriver(root string) volumeDriver {
	return volumeDriver{
		root:   root,
		mountm: NewVolumeManager(),
		m:      &sync.Mutex{},
	}
}

func (v volumeDriver) Create(r volume.Request) volume.Response {
	log.Debugf("Entering Create: name: %s, options %v", r.Name, r.Options)

	v.m.Lock()
	defer v.m.Unlock()

	log.Debugf("Create volume -> name: %s, %v", r.Name, r.Options)

	// TODO - check for share option
	// TODO - refactor to use name instead of key

	dest := mountpoint(v.root, r.Name)
	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}
	v.mountm.Create(r.Name, dest, r.Options)
	return volume.Response{}
}

func (v volumeDriver) Remove(r volume.Request) volume.Response {
	log.Debugf("Entering Remove: name: %s, options %v", r.Name, r.Options)
	v.m.Lock()
	defer v.m.Unlock()

	if err := v.mountm.Delete(r.Name); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (v volumeDriver) Path(r volume.Request) volume.Response {
	log.Debugf("Host path for %s is at %s", r.Name, mountpoint(v.root, r.Name))
	return volume.Response{Mountpoint: mountpoint(v.root, r.Name)}
}

func (v volumeDriver) Get(r volume.Request) volume.Response {
	log.Debugf("Entering Get: %v", r)
	v.m.Lock()
	defer v.m.Unlock()
	hostdir := mountpoint(v.root, r.Name)

	if v.mountm.HasMount(r.Name) {
		log.Debugf("Get: mount found for %s, host directory: %s", r.Name, hostdir)
		return volume.Response{Volume: &volume.Volume{Name: r.Name, Mountpoint: hostdir}}
	}
	return volume.Response{}
}

func (v volumeDriver) List(r volume.Request) volume.Response {
	log.Debugf("Entering List: %v", r)
	return volume.Response{Volumes: v.mountm.GetVolumes(v.root)}
}
