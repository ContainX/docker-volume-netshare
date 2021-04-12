package drivers

import (
	"fmt"
	"os"
	"os/user"
	"path/filepath"
	"strconv"
	"sync"

	"github.com/docker/go-plugins-helpers/volume"
	log "github.com/sirupsen/logrus"
)

type cacheDriver struct {
	root       string
	mountm     *MountManager
	m          *sync.Mutex
	dirs       map[string]map[string]string
	mountPoint string
	state      string
}

func newCacheDriver(root string, mounts *MountManager, path string, state string) cacheDriver {
	// Load state info
	jsonData, _ := readState(state)
	log.Printf("Json: %s", jsonData)

	return cacheDriver{
		root:       root,
		mountm:     mounts,
		m:          &sync.Mutex{},
		dirs:       jsonData,
		mountPoint: path,
		state:      state,
	}
}

func (c cacheDriver) Create(r *volume.CreateRequest) error {
	log.Debugf("Entering Create: name: %s", r.Name)

	c.m.Lock()
	defer c.m.Unlock()

	if _, err := strconv.Atoi(r.Name); err == nil {
		u, err := user.LookupId(r.Name)
		if err != nil {
			return err
		}

		volumePath := filepath.Join(c.mountPoint, u.Uid)

		// Create the volume if does not exist
		if _, ok := c.dirs[u.Uid]; !ok {
			// Create directory if does not exist and set permissions
			log.Debugf("Create ccache directory: %s/%s", c.mountPoint, u.Uid)
			if _, err := os.Stat(volumePath); os.IsNotExist(err) {
				os.Mkdir(volumePath, 0700)
				uid, _ := strconv.Atoi(u.Uid)
				gid, _ := strconv.Atoi(u.Gid)
				os.Chown(volumePath, uid, gid)
			}
			// Add the volume
			c.dirs[u.Uid] = map[string]string{
				"Path": volumePath,
			}
			writeState(c.state, c.dirs)
		}
	} else {
		log.Debugf("Options %v", r.Options)
		resName, resOpts := resolveName(r.Name)
		if resOpts != nil {
			// Check to make sure there aren't options, otherwise override
			if len(r.Options) == 0 {
				r.Options = resOpts
			}
		}
		log.Debugf("Create volume -> name: %s, %v", resName, r.Options)

		dest := mountpoint(c.root, resName)
		if err := createDest(dest); err != nil {
			return err
		}
		c.mountm.Create(resName, dest, r.Options)
	}
	return nil
}

func (c cacheDriver) Remove(r *volume.RemoveRequest) error {
	c.m.Lock()
	defer c.m.Unlock()
	if _, err := strconv.Atoi(r.Name); err == nil {
		log.Printf("Entering Remove: name: %s", r.Name)
		if path, ok := c.dirs[r.Name]["Path"]; ok {
			delete(c.dirs, r.Name)
			log.Printf("Remove ccache directory: %s", path)
			os.RemoveAll(path)
			writeState(c.state, c.dirs)
		}
	} else {
		resolvedName, _ := resolveName(r.Name)
		log.Debugf("Entering Remove: name: %s, resolved-name: %s", r.Name, resolvedName)
		if err := c.mountm.Delete(resolvedName); err != nil {
			return err
		}
	}
	return nil
}

func (c cacheDriver) Path(r *volume.PathRequest) (*volume.PathResponse, error) {
	if _, err := strconv.Atoi(r.Name); err == nil {
		log.Debugf("Get volume path for: %s", r.Name)
		if path, ok := c.dirs[r.Name]["Path"]; ok {
			return &volume.PathResponse{Mountpoint: path}, nil
		}
	} else {
		resolvedName, _ := resolveName(r.Name)
		log.Debugf("Host path for %s (%s) is at %s", r.Name, resolvedName, mountpoint(c.root, resolvedName))
		return &volume.PathResponse{Mountpoint: mountpoint(c.root, resolvedName)}, nil
	}
	return &volume.PathResponse{Mountpoint: ""}, fmt.Errorf("Can't find volume: %s", r.Name)
}

func (c cacheDriver) Get(r *volume.GetRequest) (*volume.GetResponse, error) {
	log.Debugf("Entering Get: %v", r)

	c.m.Lock()
	defer c.m.Unlock()

	if _, err := strconv.Atoi(r.Name); err == nil {
		if v, ok := c.dirs[r.Name]; ok {
			log.Debugf("Get: path found for %s", r.Name)
			return &volume.GetResponse{
				Volume: &volume.Volume{
					Name:       r.Name,
					Mountpoint: v["Path"],
				},
			}, nil
		}
	} else {
		resolvedName, _ := resolveName(r.Name)
		hostdir := mountpoint(c.root, resolvedName)
		if c.mountm.HasMount(resolvedName) {
			log.Debugf("Get: mount found for %s, host directory: %s", resolvedName, hostdir)
			return &volume.GetResponse{Volume: &volume.Volume{Name: resolvedName, Mountpoint: hostdir}}, nil
		}
	}
	return &volume.GetResponse{}, fmt.Errorf("Can't find volume: %s", r.Name)
}

func (c cacheDriver) List() (*volume.ListResponse, error) {
	log.Debugf("Entering List")
	cachevols := []*volume.Volume{}
	for k, v := range c.dirs {
		// Only if path exist list it as volume
		if _, err := os.Stat(v["Path"]); !os.IsNotExist(err) {
			cachevols = append(cachevols, &volume.Volume{
				Name:       k,
				Mountpoint: v["Path"],
			})
		}
	}
	volumes := append(cachevols, c.mountm.GetVolumes(c.root)...)
	return &volume.ListResponse{Volumes: volumes}, nil
}

func (c cacheDriver) Capabilities() *volume.CapabilitiesResponse {
	// log.Debugf("Entering Capabilities: %v", r)
	return &volume.CapabilitiesResponse{
		Capabilities: volume.Capability{
			Scope: "local",
		},
	}
}
