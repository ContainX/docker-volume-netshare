package drivers

import (
	"bytes"
	"fmt"
	"github.com/calavera/dkvolume"
	"log"
	"os"
	"sync"
)

type smbDriver struct {
	root      string
	user      string
	pass      string
	workgroup string
	m         *sync.Mutex
}

func NewSambaDriver(root, user, pass, workgroup string) smbDriver {
	d := smbDriver{
		root:      root,
		user:      user,
		workgroup: workgroup,
		m:         &sync.Mutex{},
	}
	return d
}

func (s smbDriver) Create(r dkvolume.Request) dkvolume.Response {
	return dkvolume.Response{}
}

func (s smbDriver) Remove(r dkvolume.Request) dkvolume.Response {
	log.Printf("Removing volume %s\n", r.Name)
	return dkvolume.Response{}
}

func (s smbDriver) Path(r dkvolume.Request) dkvolume.Response {
	log.Printf("Path for %s is at %s\n", r.Name, mountpoint(s.root, r.Name))
	return dkvolume.Response{Mountpoint: mountpoint(s.root, r.Name)}
}

func (s smbDriver) Mount(r dkvolume.Request) dkvolume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)

	log.Printf("Mounting Samba volume %s on %s, %v\n", source, dest, r.Options)

	if err := createDest(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := s.mountVolume(source, dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}
	return dkvolume.Response{Mountpoint: dest}
}

func (s smbDriver) Unmount(r dkvolume.Request) dkvolume.Response {
	s.m.Lock()
	defer s.m.Unlock()
	dest := mountpoint(s.root, r.Name)
	source := s.fixSource(r.Name)

	log.Printf("Unmounting volume %s from %s\n", source, dest)

	if err := run(fmt.Sprintf("umount %s", dest)); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	if err := os.RemoveAll(dest); err != nil {
		return dkvolume.Response{Err: err.Error()}
	}

	return dkvolume.Response{}
}

func (s smbDriver) fixSource(name string) string {
	return "//" + name
}

func (s smbDriver) mountVolume(source, dest string) error {
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

	if s.workgroup != "" {
		opts.WriteString(fmt.Sprintf("domain=%s,", s.workgroup))
	}
	opts.WriteString("rw ")

	opts.WriteString(fmt.Sprintf("%s %s", source, dest))
	cmd := fmt.Sprintf("mount -t cifs %s", opts.String())
	log.Printf("Executing: %s\n", cmd)
	return run(cmd)
}
