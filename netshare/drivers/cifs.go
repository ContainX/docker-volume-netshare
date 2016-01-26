package drivers

import (
	"bytes"
	"fmt"
	log "github.com/Sirupsen/logrus"
	"github.com/dickeyxxx/netrc"
	"github.com/docker/go-plugins-helpers/volume"
	"os"
	"path/filepath"
	"strings"
	"sync"
)

const (
	UsernameOpt = "username"
	PasswordOpt = "password"
	DomainOpt   = "domain"
)

type cifsDriver struct {
	root   string
	creds  *cifsCreds
	netrc  *netrc.Netrc
	mountm *mountManager
	m      *sync.Mutex
}

type cifsCreds struct {
	user   string
	pass   string
	domain string
}

func NewCIFSDriver(root, user, pass, domain, netrc_path string) cifsDriver {
	d := cifsDriver{
		root:   root,
		creds:  &cifsCreds{user: user, pass: pass, domain: domain},
		netrc:  parseNetRC(netrc_path),
		mountm: NewVolumeManager(),
		m:      &sync.Mutex{},
	}
	return d
}

func parseNetRC(path string) *netrc.Netrc {
	if n, err := netrc.Parse(filepath.Join(path, ".netrc")); err == nil {
		return n
	} else {
		log.Warnf("Error: %s", err.Error())
	}
	return nil
}

func (s cifsDriver) Create(r volume.Request) volume.Response {
	log.Debugf("Create: %s, %v", r.Name, r.Options)
	dest := mountpoint(s.root, r.Name)
	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}
	s.mountm.Create(dest, r.Name, r.Options)
	return volume.Response{}
}

func (s cifsDriver) Remove(r volume.Request) volume.Response {
	log.Debugf("Removing volume %s", r.Name)
	return volume.Response{}
}

func (s cifsDriver) Path(r volume.Request) volume.Response {
	log.Debugf("Path for %s is at %s", r.Name, mountpoint(s.root, r.Name))
	return volume.Response{Mountpoint: mountpoint(s.root, r.Name)}
}

func (s cifsDriver) Mount(r volume.Request) volume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)
	host := parseHost(r.Name)
	log.Infof("Mount: %s, %v", r.Name, r.Options)

	if s.mountm.HasMount(dest) && s.mountm.Count(dest) > 0 {
		log.Infof("Using existing CIFS volume mount: %s", dest)
		s.mountm.Increment(dest)
		return volume.Response{Mountpoint: dest}
	}

	log.Infof("Mounting CIFS volume %s on %s", source, dest)

	if err := createDest(dest); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := s.mountVolume(source, dest, s.getCreds(host)); err != nil {
		return volume.Response{Err: err.Error()}
	}
	s.mountm.Add(dest, r.Name)
	return volume.Response{Mountpoint: dest}
}

func (s cifsDriver) Unmount(r volume.Request) volume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)

	if s.mountm.HasMount(dest) {
		if s.mountm.Count(dest) > 1 {
			log.Infof("Skipping unmount for %s - in use by other containers", dest)
			s.mountm.Decrement(dest)
			return volume.Response{}
		}
		s.mountm.Decrement(dest)
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

func (s cifsDriver) fixSource(name string) string {
	return "//" + name
}

func parseHost(name string) string {
	if strings.ContainsAny(name, "/") {
		s := strings.Split(name, "/")
		return s[0]
	}
	return name
}

func (s cifsDriver) mountVolume(source, dest string, creds *cifsCreds) error {
	var opts bytes.Buffer

	opts.WriteString("-o ")
	var user = creds.user
	var pass = creds.pass
	var domain = creds.domain

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
	log.Debugf("Executing: %s\n", strings.Replace(cmd, pass, "", 1))
	return run(cmd)
}

func (s cifsDriver) getCreds(host string) *cifsCreds {
	log.Debugf("GetCreds: host=%s, netrc=%v", host, s.netrc)
	if s.netrc != nil {
		m := s.netrc.Machine(host)
		if m != nil {
			return &cifsCreds{
				user:   m.Get("username"),
				pass:   m.Get("password"),
				domain: m.Get("domain"),
			}
		}
	}
	return s.creds
}
