package drivers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/calavera/dkvolume"
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

func (n nfsDriver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (n nfsDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Debugf("Removing volume %s\n", r.Name)
	return dkvolume.Response{}
}

func (n nfsDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Debugf("Path for %s is at %s\n", r.Name, mountpoint(n.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(n.root, r.Name)}
}

func (n nfsDriver) Mount(r dkvolume.Request) dkvolume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	if n.mountm.HasMount(dest) && n.mountm.Count(dest) > 0 {
		log.Infof("Using existing NFS volume mount: %s\n", dest)
		n.mountm.Increment(dest)
		return dkvolume.Response{Mountpoint: dest}
	}

	log.Infof("Mounting NFS volume %s on %s\n", source, dest)

	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := mountVolume(source, dest, n.version); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	n.mountm.Add(dest, r.Name)
	return dkvolume.Response{Mountpoint: dest}
}

func (n nfsDriver) Unmount(r dkvolume.Request) dkvolume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	if n.mountm.HasMount(dest) {
		if n.mountm.Count(dest) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers\n", dest)
			n.mountm.Decrement(dest)
			return dkvolume.Response{}
		}
		n.mountm.Decrement(dest)
	}

	log.Infof("Unmounting volume %s from %s\n", source, dest)

	if err := run(fmt.Sprintf("umount %s", dest)); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := os.RemoveAll(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	return dkvolume.Response{}
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
