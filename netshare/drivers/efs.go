package drivers

import (
	"fmt"
	"github.com/calavera/dkvolume"
	"log"
	"os"
	"strings"
	"sync"
)

const (
	EfsTemplateURI = "%s.%s.efs.%s.amazonaws.com:/"
)

type efsDriver struct {
	root      string
	availzone string
	resolve   bool
	region    string
	mountm    *mountManager
	m         *sync.Mutex
}

func NewEFSDriver(root, az string, resolve bool) efsDriver {

	d := efsDriver{
		root:    root,
		resolve: resolve,
		mountm:  NewVolumeManager(),
		m:       &sync.Mutex{},
	}
	md, err := fetchAWSMetaData()
	if err != nil {
		log.Printf("Error resolving AWS metadata: %s\n", err.Error())
		os.Exit(1)
	}
	d.region = md.Region
	if az == "" {
		d.availzone = md.AvailZone
	}
	return d
}

func (e efsDriver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (e efsDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Printf("Removing volume %s\n", r.Name)
	return dkvolume.Response{}
}

func (e efsDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Printf("Path for %s is at %s\n", r.Name, mountpoint(e.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(e.root, r.Name)}
}

func (e efsDriver) Mount(r dkvolume.Request) dkvolume.Response {
	e.m.Lock()
	defer e.m.Unlock()
	dest := mountpoint(e.root, r.Name)
	source := e.fixSource(r.Name)

	if e.mountm.HasMount(dest) && e.mountm.Count(dest) > 0 {
		log.Printf("Using existing EFS volume mount: %s\n", dest)
		e.mountm.Increment(dest)
		return dkvolume.Response{Mountpoint: dest}
	}

	log.Printf("Mounting EFS volume %s on %s\n", source, dest)

	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := mountVolume(source, dest, 4); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	e.mountm.Add(dest, r.Name)
	return dkvolume.Response{Mountpoint: dest}
}

func (e efsDriver) Unmount(r dkvolume.Request) dkvolume.Response {
	e.m.Lock()
	defer e.m.Unlock()
	dest := mountpoint(e.root, r.Name)
	source := e.fixSource(r.Name)

	if e.mountm.HasMount(dest) {
		if e.mountm.Count(dest) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers\n", dest)
			e.mountm.Decrement(dest)
			return dkvolume.Response{}
		}
		e.mountm.Decrement(dest)
	}

	log.Printf("Unmounting volume %s from %s\n", source, dest)

	if err := run(fmt.Sprintf("umount %s", dest)); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := os.RemoveAll(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	return dkvolume.Response{}
}

func (e efsDriver) fixSource(name string) string {
	v := strings.Split(name, "/")
	if e.resolve {
		return fmt.Sprintf(EfsTemplateURI, e.availzone, v[0], e.region)
	}
	v[0] = v[0] + ":"
	return strings.Join(v, "/")
}
