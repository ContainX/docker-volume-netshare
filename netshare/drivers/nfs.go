package drivers

import (
	"fmt"
	"github.com/calavera/dkvolume"
	"log"
	"os"
	"strings"
	"sync"
)

type nfsDriver struct {
	root    string
	version int
	m       *sync.Mutex
}

func NewNfsDriver(root string, version int) nfsDriver {
	d := nfsDriver{
		root:    root,
		version: version,
		m:       &sync.Mutex{},
	}
	return d
}

func (n nfsDriver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (n nfsDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Printf("Removing volume %s\n", r.Name)
	return dkvolume.Response{}
}

func (n nfsDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Printf("Path for %s is at %s\n", r.Name, mountpoint(n.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(n.root, r.Name)}
}

func (n nfsDriver) Mount(r dkvolume.Request) dkvolume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	log.Printf("Mounting NFS volume %s on %s, %v\n", source, dest, r.Options)

	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := n.mountVolume(source, dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	return dkvolume.Response{Mountpoint: dest}
}

func (n nfsDriver) Unmount(r dkvolume.Request) dkvolume.Response {
	n.m.Lock()
	defer n.m.Unlock()
	dest := mountpoint(n.root, r.Name)
	source := n.fixSource(r.Name)

	log.Printf("Unmounting volume %s from %s\n", source, dest)

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

func (n nfsDriver) mountVolume(source, dest string) error {
	var cmd string
	switch n.version {
	case 3:
		cmd = fmt.Sprintf("mount -o port=2049,nolock,proto=tcp %s %s", source, dest)
	default:
		cmd = fmt.Sprintf("mount -t nfs4 %s %s", source, dest)
	}
	return run(cmd)
}
