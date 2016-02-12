package drivers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"strings"
	"sync"
)

type nfsDriver struct {
	root    string
	version int
	mountm  *mountManager
	m       *sync.Mutex
}

func NewNFSDriver(root string, version int) nfsDriver {
	d := nfsDriver{
		root:    root,
		version: version,
		mountm:  NewVolumeManager(),
		m:       &sync.Mutex{},
	}
	return d
}

func (n nfsDriver) Create(r volume.Request) volume.Response {
	log.Debugf("Create: %s, %v", r.Name, r.Options)
	dest := mountpoint(n.root, r.Name)
	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}
	n.mountm.Create(dest, r.Name, r.Options)
	return volume.Response{}
}

func (n nfsDriver) Remove(r volume.Request) volume.Response {
	log.Debugf("Removing volume %s", r.Name)
	return volume.Response{}
}

func (n nfsDriver) Path(r volume.Request) volume.Response {
	log.Debugf("Path for %s is at %s", r.Name, mountpoint(n.root, r.Name))
	return volume.Response{Mountpoint: mountpoint(n.root, r.Name)}
}

func (s nfsDriver) Get(r volume.Request) volume.Response {
	log.Debugf("Get for %s is at %s", r.Name, mountpoint(s.root, r.Name))
	return volume.Response{ Volume: &volume.Volume{Name: r.Name, Mountpoint: mountpoint(s.root, r.Name)}}
}

func (s nfsDriver) List(r volume.Request) volume.Response {
	log.Debugf("List Volumes")
	return volume.Response{ Volumes: s.mountm.GetVolumes(s.root) }
}

func (n nfsDriver) Mount(r volume.Request) volume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	if n.mountm.HasMount(dest) && n.mountm.Count(dest) > 0 {
		log.Infof("Using existing NFS volume mount: %s", dest)
		n.mountm.Increment(dest)
		return volume.Response{Mountpoint: dest}
	}

	log.Infof("Mounting NFS volume %s on %s", source, dest)

	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := mountVolume(source, dest, n.version); err != nil {
		return volume.Response{Err: err.Error()}
	}
	n.mountm.Add(dest, r.Name)
	return volume.Response{Mountpoint: dest}
}

func (n nfsDriver) Unmount(r volume.Request) volume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	if n.mountm.HasMount(dest) {
		if n.mountm.Count(dest) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers", dest)
			n.mountm.Decrement(dest)
			return volume.Response{}
		}
		n.mountm.Decrement(dest)
	}

	log.Infof("Unmounting volume %s from %s", source, dest)

	if err := run(fmt.Sprintf("umount %s", dest)); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := os.RemoveAll(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (n nfsDriver) fixSource(name string) string {
	source := strings.Split(name, "/")
	source[0] = source[0] + ":"
	return strings.Join(source, "/")
}

func mountVolume(source, dest string, version int) error {
	var cmd string
	switch version {
	case 3:
		cmd = fmt.Sprintf("mount -o port=2049,nolock,proto=tcp %s %s", source, dest)
	default:
		cmd = fmt.Sprintf("mount -t nfs4 %s %s", source, dest)
	}
	log.Debugf("exec: %s\n", cmd)
	return run(cmd)
}
