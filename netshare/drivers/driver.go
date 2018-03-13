package drivers

import (
	"sync"

	log "github.com/sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
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

func (v volumeDriver) Create(r *volume.CreateRequest) error {
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
		return err
	}

	v.mountm.Create(resName, dest, r.Options)
	return nil
}

func (v volumeDriver) Remove(r *volume.RemoveRequest) error {

	resolvedName, _ := resolveName(r.Name)

	log.Debugf("Entering Remove: name: %s, resolved-name: %s", r.Name, resolvedName)
	v.m.Lock()
	defer v.m.Unlock()

	if err := v.mountm.Delete(resolvedName); err != nil {
		return err
	}
	return nil
}

func (v volumeDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	resolvedName, _ := resolveName(r.Name)

	log.Debugf("Host path for %s (%s) is at %s", r.Name, resolvedName, mountpoint(v.root, resolvedName))
	return &volume.PathResponse{Mountpoint: mountpoint(v.root, resolvedName)}, nil
}

func (v volumeDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	log.Debugf("Entering Get: %v", r)
	v.m.Lock()
	defer v.m.Unlock()
	resolvedName, _ := resolveName(r.Name)

	hostdir := mountpoint(v.root, resolvedName)

	if v.mountm.HasMount(resolvedName) {
		log.Debugf("Get: mount found for %s, host directory: %s", resolvedName, hostdir)
		return &volume.GetResponse{Volume: &volume.Volume{Name: resolvedName, Mountpoint: hostdir}}, nil
	}
	return nil, nil
}

func (v volumeDriver) List() (*volume.ListResponse, error) {
	log.Debugf("Entering List")
	return &volume.ListResponse{Volumes: v.mountm.GetVolumes(v.root)}, nil
}

func (v volumeDriver) Capabilities() *volume.CapabilitiesResponse {
	// log.Debugf("Entering Capabilities: %v", r)
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
