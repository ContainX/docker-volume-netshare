package drivers

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

const (
	NfsCacheOptions   = "nfsopts"
	DefaultNfsCacheV3 = "port=2049,nolock,proto=tcp"
)

type nfsCacheDriver struct {
	cacheDriver
	version int
	nfsopts map[string]string
	lazyUmount bool
}

func NewNFSCacheDriver(root string, version int, nfsopts string, mounts *MountManager, path string, state string, lazyUmount bool) nfsCacheDriver {
	d := nfsCacheDriver{
		cacheDriver: newCacheDriver(root, mounts, path, state),
		version:     version,
		nfsopts:     map[string]string{},
		lazyUmount:  lazyUmount,
	}

	if len(nfsopts) > 0 {
		d.nfsopts[NfsCacheOptions] = nfsopts
	}
	return d
}

func (n nfsCacheDriver) Mount(r *volume.MountRequest) (*volume.MountResponse, error) {
	log.Debugf("Entering Mount: %v", r)

	n.m.Lock()
	defer n.m.Unlock()

	if _, err := strconv.Atoi(r.Name); err == nil {
		if path, ok := n.cacheDriver.dirs[r.Name]["Path"]; ok {
			log.Printf("Mount volume: %s with mountpoint: %s", r.Name, path)
			return &volume.MountResponse{Mountpoint: path}, nil
		}
	} else {
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

		if n.mountm.HasMount(resolvedName) {
			log.Infof("Using existing NFS volume mount: %s", hostdir)
			n.mountm.Increment(resolvedName)
			regexp1 := "(^|\\s)\\K"
			regexp2 := "(?=\\s|$)"
			if err := run(fmt.Sprintf("grep -c -P '%s%s%s' /proc/mounts", regexp1, hostdir, regexp2)); err != nil {
				log.Infof("Existing NFS volume not mounted, force remount.")
				// maintain count
				if n.mountm.Count(resolvedName) > 0 {
					n.mountm.Decrement(resolvedName)
				}
			} else {
				//n.mountm.Increment(resolvedName)
				return &volume.MountResponse{Mountpoint: hostdir}, nil
			}
		}

		log.Infof("Mounting NFS volume %s on %s", source, hostdir)

		if err := createDest(hostdir); err != nil {
			if n.mountm.Count(resolvedName) > 0 {
				n.mountm.Decrement(resolvedName)
			}
			return nil, err
		}

		if n.mountm.HasMount(resolvedName) == false {
			n.mountm.Create(resolvedName, hostdir, resOpts)
		}

		n.mountm.Add(resolvedName, hostdir)

		if err := n.mountVolume(resolvedName, source, hostdir, n.version); err != nil {
			n.mountm.Decrement(resolvedName)
			return nil, err
		}

		if n.mountm.GetOption(resolvedName, ShareOpt) != "" && n.mountm.GetOptionAsBool(resolvedName, CreateOpt) {
			log.Infof("Mount: Share and Create options enabled - using %s as sub-dir mount", resolvedName)
			datavol := filepath.Join(hostdir, resolvedName)
			if err := createDest(filepath.Join(hostdir, resolvedName)); err != nil {
				n.mountm.Decrement(resolvedName)
				return nil, err
			}
			hostdir = datavol
		}

		return &volume.MountResponse{Mountpoint: hostdir}, nil
	}
	return &volume.MountResponse{Mountpoint: ""}, fmt.Errorf("Can't find volume: %s", r.Name)
}

func (n nfsCacheDriver) Unmount(r *volume.UnmountRequest) error {
	log.Debugf("Entering Unmount: %v", r)

	n.m.Lock()
	defer n.m.Unlock()

	if _, err := strconv.Atoi(r.Name); err != nil {
		resolvedName, _ := resolveName(r.Name)

		hostdir := mountpoint(n.root, resolvedName)

		if n.mountm.HasMount(resolvedName) {
			if n.mountm.Count(resolvedName) > 1 {
				log.Printf("Skipping unmount for %s - in use by other containers", resolvedName)
				n.mountm.Decrement(resolvedName)
				return nil
			}
			n.mountm.Decrement(resolvedName)
		}

		lazy := ""
		if n.lazyUmount {
			lazy = "-l"
			log.Infof("Unmounting volume name %s from %s with -l", resolvedName, hostdir)
		} else {
			log.Infof("Unmounting volume name %s from %s", resolvedName, hostdir)
		}

		if err := run(fmt.Sprintf("umount %s %s", lazy, hostdir)); err != nil {
			log.Errorf("Error unmounting volume from host: %s", err.Error())
			return err
		}

		n.mountm.DeleteIfNotManaged(resolvedName)

		// Check if directory is empty. This command will return "err" if empty
		if err := run(fmt.Sprintf("ls -1 %s | grep .", hostdir)); err == nil {
			log.Warnf("Directory %s not empty after unmount. Skipping RemoveAll call.", hostdir)
		} else {
			if err := os.RemoveAll(hostdir); err != nil {
				return err
			}
		}
	}

	return nil
}

func (n nfsCacheDriver) fixSource(name string) string {
	if n.mountm.HasOption(name, ShareOpt) {
		return addShareColon(n.mountm.GetOption(name, ShareOpt))
	}
	return addShareColon(name)
}

func (n nfsCacheDriver) mountVolume(name, source, dest string, version int) error {
	var cmd string

	options := merge(n.mountm.GetOptions(name), n.nfsopts)
	opts := ""
	if val, ok := options[NfsCacheOptions]; ok {
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
			opts = DefaultNfsCacheV3
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
