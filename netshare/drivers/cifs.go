package drivers

import (
	"bytes"
	"fmt"
	"github.com/calavera/dkvolume"
	"log"
	"os"
	"sync"
)

type cifsDriver struct {
	root   string
	user   string
	pass   string
	domain string
	mountm *mountManager
	m      *sync.Mutex
}

func NewCIFSDriver(root, user, pass, domain string) cifsDriver {
	d := cifsDriver{
		root:   root,
		user:   user,
		domain: domain,
		mountm: NewVolumeManager(),
		m:      &sync.Mutex{},
	}
	return d
}

func (s cifsDriver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (s cifsDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Printf("Removing volume %s\n", r.Name)
	return dkvolume.Response{}
}

func (s cifsDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Printf("Path for %s is at %s\n", r.Name, mountpoint(s.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(s.root, r.Name)}
}

func (s cifsDriver) Mount(r dkvolume.Request) dkvolume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)

	if s.mountm.HasMount(dest) && s.mountm.Count(dest) > 0 {
		log.Printf("Using existing CIFS volume mount: %s\n", dest)
		s.mountm.Increment(dest)
		return dkvolume.Response{Mountpoint: dest}
	}

	log.Printf("Mounting CIFS volume %s on %s, %v\n", source, dest, r.Options)

	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := s.mountVolume(source, dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	s.mountm.Add(dest, r.Name)
	return dkvolume.Response{Mountpoint: dest}
}

func (s cifsDriver) Unmount(r dkvolume.Request) dkvolume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)

	if s.mountm.HasMount(dest) {
		if s.mountm.Count(dest) > 1 {
			log.Printf("Skipping unmount for %s - in use by other containers\n", dest)
			s.mountm.Decrement(dest)
			return dkvolume.Response{}
		}
		s.mountm.Decrement(dest)
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

func (s cifsDriver) fixSource(name string) string {
	return "//" + name
}

func (s cifsDriver) mountVolume(source, dest string) error {
	var opts bytes.Buffer

	opts.WriteString("-o ")

	if s.user != "" {
		opts.WriteString(fmt.Sprintf("username=%s,", s.user))
		if s.pass != "" {
			opts.WriteString(fmt.Sprintf("password=%s,", s.pass))
		}
	} else {
		opts.WriteString("guest,")
	}

	if s.domain != "" {
		opts.WriteString(fmt.Sprintf("domain=%s,", s.domain))
	}
	opts.WriteString("rw ")

	opts.WriteString(fmt.Sprintf("%s %s", source, dest))
	cmd := fmt.Sprintf("mount -t cifs %s", opts.String())
	log.Printf("Executing: %s\n", cmd)
	return run(cmd)
}
