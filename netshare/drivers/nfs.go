package drivers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"path/filepath"
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

func (n nfsDriver) Mount(r volume.MountRequest) volume.Response {
	log.Debugf("Entering Mount: %v", r)
	n.m.Lock()
	defer n.m.Unlock()

	resolvedName, resOpts := resolveName(r.Name)

	hostdir := mountpoint(n.root, resolvedName)
	source := n.fixSource(resolvedName)

	// Support adhoc mounts (outside of docker volume create)
	// need to adjust source for ShareOpt
	if resOpts != nil {
		if share, found := resOpts[ShareOpt]; found {
			source = n.fixSource(share)
		}
	}

	if n.mountm.HasMount(resolvedName) && n.mountm.Count(resolvedName) > 0 {
		log.Infof("Using existing NFS volume mount: %s", hostdir)
		n.mountm.Increment(resolvedName)
		if err := run(fmt.Sprintf("grep -c %s /proc/mounts", hostdir)); err != nil {
			log.Infof("Existing NFS volume not mounted, force remount.")
		} else {
			return volume.Response{Mountpoint: hostdir}
		}
	}

	log.Infof("Mounting NFS volume %s on %s", source, hostdir)

	if err := createDest(hostdir); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if n.mountm.HasMount(resolvedName) == false {
		n.mountm.Create(resolvedName, hostdir, resOpts)
	}

	if err := n.mountVolume(resolvedName, source, hostdir, n.version); err != nil {
		return volume.Response{Err: err.Error()}
	}
	n.mountm.Add(resolvedName, hostdir)

	if n.mountm.GetOption(resolvedName, ShareOpt) != "" && n.mountm.GetOptionAsBool(resolvedName, CreateOpt) {
		log.Infof("Mount: Share and Create options enabled - using %s as sub-dir mount", resolvedName)
		datavol := filepath.Join(hostdir, resolvedName)
		if err := createDest(filepath.Join(hostdir, resolvedName)); err != nil {
			return volume.Response{Err: err.Error()}
		}
		hostdir = datavol
	}

	return volume.Response{Mountpoint: hostdir}
}

func (n nfsDriver) Unmount(r volume.UnmountRequest) volume.Response {
	log.Debugf("Entering Unmount: %v", r)

	n.m.Lock()
	defer n.m.Unlock()

	resolvedName, _ := resolveName(r.Name)

	hostdir := mountpoint(n.root, resolvedName)

	if n.mountm.HasMount(resolvedName) {
		if n.mountm.Count(resolvedName) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers", resolvedName)
			n.mountm.Decrement(resolvedName)
			return volume.Response{}
		}
		n.mountm.Decrement(resolvedName)
	}

	log.Infof("Unmounting volume name %s from %s", resolvedName, hostdir)

	if err := run(fmt.Sprintf("umount %s", hostdir)); err != nil {
		log.Errorf("Error unmounting volume from host: %s", err.Error())
		return volume.Response{Err: err.Error()}
	}

	n.mountm.DeleteIfNotManaged(resolvedName)

	if err := os.RemoveAll(hostdir); err != nil {
		return volume.Response{Err: err.Error()}
	}

	return volume.Response{}
}

func (n nfsDriver) fixSource(name string) string {
	if n.mountm.HasOption(name, ShareOpt) {
		return addShareColon(n.mountm.GetOption(name, ShareOpt))
	}
	return addShareColon(name)
}

func (n nfsDriver) mountVolume(name, source, dest string, version int) error {
	var cmd string

	options := merge(n.mountm.GetOptions(name), n.nfsopts)
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
		cmd = fmt.Sprintf("%s -t nfs -o %s %s %s", mountCmd, opts, source, dest)
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
