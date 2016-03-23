package drivers

import (
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"strings"
	"sync"
)

const (
	EfsTemplateURI = "%s.%s.efs.%s.amazonaws.com"
)

type efsDriver struct {
	root      string
	availzone string
	resolve   bool
	region    string
	resolver  *Resolver
	mountm    *mountManager
	m         *sync.Mutex
	dnscache  map[string]string
}

func NewEFSDriver(root, az, nameserver string, resolve bool) efsDriver {

	d := efsDriver{
		root:     root,
		resolve:  resolve,
		mountm:   NewVolumeManager(),
		m:        &sync.Mutex{},
		dnscache: map[string]string{},
	}

	if resolve {
		d.resolver = NewResolver(nameserver)
	}
	md, err := fetchAWSMetaData()
	if err != nil {
		log.Fatalf("Error resolving AWS metadata: %s", err.Error())
		os.Exit(1)
	}
	d.region = md.Region
	if az == "" {
		d.availzone = md.AvailZone
	}
	return d
}

func (e efsDriver) Create(r volume.Request) volume.Response {
	log.Debugf("Create: %s, %v", r.Name, r.Options)
	dest := mountpoint(e.root, r.Name)
	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}
	e.mountm.Create(dest, r.Name, r.Options)
	return volume.Response{}
}

func (e efsDriver) Remove(r volume.Request) volume.Response {
	log.Debugf("Removing volume %s", r.Name)
	return volume.Response{}
}

func (e efsDriver) Path(r volume.Request) volume.Response {
	log.Debugf("Path for %s is at %s", r.Name, mountpoint(e.root, r.Name))
	return volume.Response{Mountpoint: mountpoint(e.root, r.Name)}
}

func (s efsDriver) Get(r volume.Request) volume.Response {
	log.Debugf("Get for %s is at %s", r.Name, mountpoint(s.root, r.Name))
	return volume.Response{ Volume: &volume.Volume{Name: r.Name, Mountpoint: mountpoint(s.root, r.Name)}}
}

func (s efsDriver) List(r volume.Request) volume.Response {
	log.Debugf("List Volumes")
	return volume.Response{ Volumes: s.mountm.GetVolumes(s.root) }
}

func (e efsDriver) Mount(r volume.Request) volume.Response {
	e.m.Lock()
	defer e.m.Unlock()
	dest := mountpoint(e.root, r.Name)
	source := e.fixSource(r.Name)

	if e.mountm.HasMount(dest) && e.mountm.Count(dest) > 0 {
		log.Infof("Using existing EFS volume mount: %s", dest)
		e.mountm.Increment(dest)
		return volume.Response{Mountpoint: dest}
	}

	log.Infof("Mounting EFS volume %s on %s", source, dest)

	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := e.mountVolume(source, dest); err != nil {
		return volume.Response{Err: err.Error()}
	}
	e.mountm.Add(dest, r.Name)
	return volume.Response{Mountpoint: dest}
}

func (e efsDriver) Unmount(r volume.Request) volume.Response {
	e.m.Lock()
	defer e.m.Unlock()
	dest := mountpoint(e.root, r.Name)
	source := e.fixSource(r.Name)

	if e.mountm.HasMount(dest) {
		if e.mountm.Count(dest) > 1 {
			log.Infof("Skipping unmount for %s - in use by other containers", dest)
			e.mountm.Decrement(dest)
			return volume.Response{}
		}
		e.mountm.Decrement(dest)
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

func (e efsDriver) fixSource(name string) string {
	v := strings.Split(name, "/")
	if e.resolve {
		uri := fmt.Sprintf(EfsTemplateURI, e.availzone, v[0], e.region)
		if i, ok := e.dnscache[uri]; ok {
			return mountSuffix(i)
		}

		log.Debugf("Attempting to resolve: %s", uri)
		if ip, err := e.resolver.Lookup(uri); err == nil {
			log.Debugf("Resolved Addresses: %s", ip)
			e.dnscache[uri] = ip
			return mountSuffix(ip)
		} else {
			log.Errorf("Error during resolve: %s", err.Error())
			return mountSuffix(uri)
		}
	}
	v[0] = v[0] + ":"
	return strings.Join(v, "/")
}

func mountSuffix(uri string) string {
	return uri + ":/"
}

func (e efsDriver) mountVolume(source, dest string) error {
	cmd := fmt.Sprintf("mount -t nfs4 %s %s", source, dest)
	log.Debugf("exec: %s\n", cmd)
	return run(cmd)
}