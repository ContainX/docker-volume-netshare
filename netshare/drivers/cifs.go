package drivers

import (
	"bytes"
	"fmt"
	"path/filepath"
	"strings"

	log "github.com/Sirupsen/logrus"
	"github.com/dickeyxxx/netrc"
	"github.com/docker/go-plugins-helpers/volume"
)

// Constants defining driver paremeters
const (
	UsernameOpt = "username"
	PasswordOpt = "password"
	DomainOpt   = "domain"
	SecurityOpt = "security"
	FileModeOpt = "fileMode"
	DirModeOpt  = "dirMode"
	CifsOpts    = "cifsopts"
)

// CifsDriver driver structure
type CifsDriver struct {
	volumeDriver
	creds    *CifsCreds
	netrc    *netrc.Netrc
	cifsopts map[string]string
}

// CifsCreds contains Options for cifs-mount
type CifsCreds struct {
	user     string
	pass     string
	domain   string
	security string
	fileMode string
	dirMode  string
}

func (creds *CifsCreds) String() string {
	return fmt.Sprintf("creds: { user=%s,pass=****,domain=%s,security=%s, fileMode=%s, dirMode=%s}", creds.user, creds.domain, creds.security, creds.fileMode, creds.dirMode)
}

// NewCifsCredentials setting the credentials
func NewCifsCredentials(user, pass, domain, security, fileMode, dirMode string) *CifsCreds {
	return &CifsCreds{user: user, pass: pass, domain: domain, security: security, fileMode: fileMode, dirMode: dirMode}
}

// NewCIFSDriver creating the cifs driver
func NewCIFSDriver(root string, creds *CifsCreds, netrc, cifsopts string) CifsDriver {
	d := CifsDriver{
		volumeDriver: newVolumeDriver(root),
		creds:        creds,
		netrc:        parseNetRC(netrc),
		cifsopts:     map[string]string{},
	}
	if len(cifsopts) > 0 {
		d.cifsopts[CifsOpts] = cifsopts
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

// Mount do the mounting
func (c CifsDriver) Mount(r volume.MountRequest) volume.Response {
	c.m.Lock()
	defer c.m.Unlock()
	hostdir := mountpoint(c.root, r.Name)
	source := c.fixSource(r.Name)
	host := c.parseHost(r.Name)

	resolvedName, resOpts := resolveName(r.Name)

	log.Infof("Mount: %s, ID: %s", r.Name, r.ID)

	// Support adhoc mounts (outside of docker volume create)
	// need to adjust source for ShareOpt
	if resOpts != nil {
		if share, found := resOpts[ShareOpt]; found {
			source = c.fixSource(share)
		}
	}

	if c.mountm.HasMount(r.Name) && c.mountm.Count(r.Name) > 0 {
		log.Infof("Using existing CIFS volume mount: %s", hostdir)
		c.mountm.Increment(r.Name)
		if err := run(fmt.Sprintf("mountpoint -q %s", hostdir)); err != nil {
			log.Infof("Existing CIFS volume not mounted, force remount.")
		} else {
			return volume.Response{Mountpoint: hostdir}
		}
	}

	log.Infof("Mounting CIFS volume %s on %s", source, hostdir)

	if err := createDest(hostdir); err != nil {
		return volume.Response{Err: err.Error()}
	}

	if err := c.mountVolume(r.Name, source, hostdir, c.getCreds(host)); err != nil {
		return volume.Response{Err: err.Error()}
	}
	c.mountm.Add(r.Name, hostdir)

	if c.mountm.GetOption(resolvedName, ShareOpt) != "" && c.mountm.GetOptionAsBool(resolvedName, CreateOpt) {
		log.Infof("Mount: Share and Create options enabled - using %s as sub-dir mount", resolvedName)
		datavol := filepath.Join(hostdir, resolvedName)
		if err := createDest(filepath.Join(hostdir, resolvedName)); err != nil {
			return volume.Response{Err: err.Error()}
		}
		hostdir = datavol
	}
	return volume.Response{Mountpoint: hostdir}
}

// Unmount do the unmounting
func (c CifsDriver) Unmount(r volume.UnmountRequest) volume.Response {
	c.m.Lock()
	defer c.m.Unlock()
	hostdir := mountpoint(c.root, r.Name)
	source := c.fixSource(r.Name)

	if c.mountm.HasMount(r.Name) {
		if c.mountm.Count(r.Name) > 1 {
			log.Infof("Skipping unmount for %s - in use by other containers", r.Name)
			c.mountm.Decrement(r.Name)
			return volume.Response{}
		}
		c.mountm.Decrement(r.Name)
	}

	log.Infof("Unmounting volume %s from %s", source, hostdir)

	if err := run(fmt.Sprintf("umount %s", hostdir)); err != nil {
		return volume.Response{Err: err.Error()}
	}

	c.mountm.DeleteIfNotManaged(r.Name)

	// ToDo:
	// This is a bad idea.
	// When there is a dangling mount you will lose all your data in the mounted folder
	// I didn't understand why you delete all the data ...?

	// if err := os.RemoveAll(hostdir); err != nil {
	//  	return volume.Response{Err: err.Error()}
	// }

	return volume.Response{}
}

func (c CifsDriver) fixSource(name string) string {
	if c.mountm.HasOption(name, ShareOpt) {
		return "//" + c.mountm.GetOption(name, ShareOpt)
	}
	return "//" + name
}

func (c CifsDriver) parseHost(name string) string {
	n := name
	if c.mountm.HasOption(name, ShareOpt) {
		n = c.mountm.GetOption(name, ShareOpt)
	}

	if strings.ContainsAny(n, "/") {
		s := strings.Split(n, "/")
		return s[0]
	}
	return n
}

func (c CifsDriver) mountVolume(name, source, dest string, creds *CifsCreds) error {
	var opts bytes.Buffer
	var user = creds.user
	var pass = creds.pass
	var domain = creds.domain
	var security = creds.security
	var fileMode = creds.fileMode
	var dirMode = creds.dirMode

	options := merge(c.mountm.GetOptions(name), c.cifsopts)
	if val, ok := options[CifsOpts]; ok {
		opts.WriteString(val + ",")
	}

	if c.mountm.HasOptions(name) {
		mopts := c.mountm.GetOptions(name)
		if v, found := mopts[UsernameOpt]; found {
			user = v
		}
		if v, found := mopts[PasswordOpt]; found {
			pass = v
		}
		if v, found := mopts[DomainOpt]; found {
			domain = v
		}
		if v, found := mopts[SecurityOpt]; found {
			security = v
		}
		if v, found := mopts[FileModeOpt]; found {
			fileMode = v
		}
		if v, found := mopts[DirModeOpt]; found {
			dirMode = v
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

	if security != "" {
		opts.WriteString(fmt.Sprintf("sec=%s,", security))
	}

	if fileMode != "" {
		opts.WriteString(fmt.Sprintf("file_mode=%s,", fileMode))
	}

	if dirMode != "" {
		opts.WriteString(fmt.Sprintf("dir_mode=%s,", dirMode))
	}

	opts.WriteString("rw ")

	opts.WriteString(fmt.Sprintf("%s %s", source, dest))
	cmd := fmt.Sprintf("mount -t cifs -o %s", opts.String())
	log.Debugf("Executing: %s\n", strings.Replace(cmd, "password="+pass, "password=****", 1))
	return run(cmd)
}

func (c CifsDriver) getCreds(host string) *CifsCreds {
	log.Debugf("GetCreds: host=%s, netrc=%v", host, c.netrc)
	if c.netrc != nil {
		m := c.netrc.Machine(host)
		if m != nil {
			return &CifsCreds{
				user:     m.Get("username"),
				pass:     m.Get("password"),
				domain:   m.Get("domain"),
				security: m.Get("security"),
				fileMode: m.Get("fileMode"),
				dirMode:  m.Get("dirMode"),
			}
		}
	}
	return c.creds
}
