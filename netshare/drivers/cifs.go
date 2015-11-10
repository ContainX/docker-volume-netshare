package drivers

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/calavera/dkvolume"
	"os"
	"sync"
)

const (
	UsernameOpt = "username"
	PasswordOpt = "password"
	DomainOpt   = "domain"
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
	log.Debugf("Create: %s, %v", r.Name, r.Options)
	dest := mountpoint(s.root, r.Name)
	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	s.mountm.Create(dest, r.Name, r.Options)
	return dkvolume.Response{}
}

func (s cifsDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Debugf("Removing volume %s", r.Name)
	return dkvolume.Response{}
}

func (s cifsDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Debugf("Path for %s is at %s", r.Name, mountpoint(s.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(s.root, r.Name)}
}

func (s cifsDriver) Mount(r dkvolume.Request) dkvolume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)
	log.Infof("Mount: %s, %v", r.Name, r.Options)

	if s.mountm.HasMount(dest) && s.mountm.Count(dest) > 0 {
		log.Infof("Using existing CIFS volume mount: %s", dest)
		s.mountm.Increment(dest)
		return dkvolume.Response{Mountpoint: dest}
	}

	log.Infof("Mounting CIFS volume %s on %s", source, dest)

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
			log.Infof("Skipping unmount for %s - in use by other containers", dest)
			s.mountm.Decrement(dest)
			return dkvolume.Response{}
		}
		s.mountm.Decrement(dest)
	}

	log.Infof("Unmounting volume %s from %s", source, dest)

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
	var user = s.user
	var pass = s.pass
	var domain = s.domain

	if s.mountm.HasOptions(dest) {
		mopts := s.mountm.GetOptions(dest)
		if v, found := mopts[UsernameOpt]; found {
			user = v
		}
		if v, found := mopts[PasswordOpt]; found {
			pass = v
		}
		if v, found := mopts[DomainOpt]; found {
			domain = v
		}
	}

	if user != "" {
		opts.WriteString(fmt.Sprintf("username=%s,", user))
		if pass != "" {
			opts.WriteString(fmt.Sprintf("password=%s,", pass))
		}
	} else {
		opts.WriteString("guest,")
	}

	if domain != "" {
		opts.WriteString(fmt.Sprintf("domain=%s,", domain))
	}
	opts.WriteString("rw ")

	opts.WriteString(fmt.Sprintf("%s %s", source, dest))
	cmd := fmt.Sprintf("mount -t cifs %s", opts.String())
	log.Debugf("Executing: %s\n", cmd)
	return run(cmd)
}
