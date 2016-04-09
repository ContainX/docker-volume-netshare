package drivers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"strings"
)

const (
	NfsOptions   = "nfsopts"
	DefaultNfsV3 = "port=2049,nolock,proto=tcp"
)

type nfsDriver struct {
	volumeDriver
	version int
	nfsopts map[string]string
}

var (
	EmptyMap = map[string]string{}
)

func NewNFSDriver(root string, version int, nfsopts string) nfsDriver {
	d := nfsDriver{
		volumeDriver: newVolumeDriver(root),
		version:      version,
		nfsopts:      map[string]string{},
	}

	if len(nfsopts) > 0 {
		d.nfsopts[NfsOptions] = nfsopts
	}
	return d
}

func (n nfsDriver) Mount(r volume.Request) volume.Response {
	log.Debugf("Entering Mount: %v", r)
	n.m.Lock()
	defer n.m.Unlock()
	hostdir := mountpoint(n.root, r.Name)
	source := n.fixSource(r)

	if n.mountm.HasMount(r.Name) && n.mountm.Count(r.Name) > 0 {
		log.Infof("Using existing NFS volume mount: %s", hostdir)
		n.mountm.Increment(r.Name)
		return volume.Response{Mountpoint: hostdir}
	}

	log.Infof("Mounting NFS volume %s on %s", source, hostdir)

	if err := createDest(hostdir); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := n.mountVolume(source, hostdir, n.version); err != nil {
		return volume.Response{Err: err.Error()}
	}
	n.mountm.Add(r.Name, hostdir)
	return volume.Response{Mountpoint: hostdir}
}

func (n nfsDriver) Unmount(r volume.Request) volume.Response {
	log.Debugf("Entering Unmount: %v", r)

	n.m.Lock()
	defer n.m.Unlock()
	hostdir := mountpoint(n.root, r.Name)

	if n.mountm.HasMount(r.Name) {
		if n.mountm.Count(r.Name) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers", r.Name)
			n.mountm.Decrement(r.Name)
			return volume.Response{}
		}
		n.mountm.Decrement(r.Name)
	}

	log.Infof("Unmounting volume name %s from %s", r.Name, hostdir)

	if err := run(fmt.Sprintf("umount %s", hostdir)); err != nil {
		return volume.Response{Err: err.Error()}
	}

	n.mountm.DeleteIfNotManaged(r.Name)

	if err := os.RemoveAll(hostdir); err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (n nfsDriver) fixSource(r volume.Request) string {
	if n.mountm.HasOption(r.Name, ShareOpt) {
		return n.mountm.GetOption(r.Name, ShareOpt)
	}
	source := strings.Split(r.Name, "/")
	source[0] = source[0] + ":"
	return strings.Join(source, "/")
}

func (n nfsDriver) mountVolume(source, dest string, version int) error {
	var cmd string

	options := n.mountOptions(n.mountm.GetOptions(dest))
	opts := ""
	if val, ok := options[NfsOptions]; ok {
		opts = val
	}

	mountCmd := "mount"

	if log.GetLevel() == log.DebugLevel {
		mountCmd = mountCmd + " -v"
	}

	switch version {
	case 3:
		log.Debugf("Mounting with NFSv3 - src: %s, dest: %s", source, dest)
		if len(opts) < 1 {
			opts = DefaultNfsV3
		}
		cmd = fmt.Sprintf("%s -o %s %s %s", mountCmd, opts, source, dest)
	default:
		log.Debugf("Mounting with NFSv4 - src: %s, dest: %s", source, dest)
		if len(opts) > 0 {
			cmd = fmt.Sprintf("%s -t nfs4 -o %s %s %s", mountCmd, opts, source, dest)
		} else {
			cmd = fmt.Sprintf("%s -t nfs4 %s %s", mountCmd, source, dest)
		}
	}
	log.Debugf("exec: %s\n", cmd)
	return run(cmd)
}

func (n nfsDriver) mountOptions(src map[string]string) map[string]string {
	if len(n.nfsopts) == 0 && len(src) == 0 {
		return EmptyMap
	}

	dst := map[string]string{}
	for k, v := range n.nfsopts {
		dst[k] = v
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}
