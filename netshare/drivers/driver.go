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

	resName, resOpts := resolveName(r.Name)
	if resOpts != nil {
		// Check to make sure there aren't options, otherwise override
		if len(r.Options) == 0 {
			r.Options = resOpts
		}
	}
	log.Debugf("Create volume -> name: %s, %v", resName, r.Options)

	dest := mountpoint(v.root, resName)
	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}

	v.mountm.Create(resName, dest, r.Options)
	return volume.Response{}
}

func (v volumeDriver) Remove(r volume.Request) volume.Response {

	resolvedName, _ := resolveName(r.Name)

	log.Debugf("Entering Remove: name: %s, resolved-name: %s, options %v", r.Name, resolvedName, r.Options)
	v.m.Lock()
	defer v.m.Unlock()

	if err := v.mountm.Delete(resolvedName); err != nil {
		return volume.Response{Err: err.Error()}
	}
	return volume.Response{}
}

func (v volumeDriver) Path(r volume.Request) volume.Response {
	resolvedName, _ := resolveName(r.Name)

	log.Debugf("Host path for %s (%s) is at %s", r.Name, resolvedName, mountpoint(v.root, resolvedName))
	return volume.Response{Mountpoint: mountpoint(v.root, resolvedName)}
}

func (v volumeDriver) Get(r volume.Request) volume.Response {
	log.Debugf("Entering Get: %v", r)
	v.m.Lock()
	defer v.m.Unlock()
	resolvedName, _ := resolveName(r.Name)

	hostdir := mountpoint(v.root, resolvedName)

	if v.mountm.HasMount(resolvedName) {
		log.Debugf("Get: mount found for %s, host directory: %s", resolvedName, hostdir)
		return volume.Response{Volume: &volume.Volume{Name: resolvedName, Mountpoint: hostdir}}
	}
	return volume.Response{}
}

func (v volumeDriver) List(r volume.Request) volume.Response {
	log.Debugf("Entering List: %v", r)
	return volume.Response{Volumes: v.mountm.GetVolumes(v.root)}
}

func (v volumeDriver) Capabilities(r volume.Request) volume.Response {
	log.Debugf("Entering Capabilities: %v", r)
	return volume.Response{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
