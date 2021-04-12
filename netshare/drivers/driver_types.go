package drivers

type DriverType int

const (
	CIFS DriverType = iota
	NFS
	NFSCACHE
	EFS
	CEPH
)

var driverTypes = []string{
	"cifs",
	"nfs",
	"nfscache",
	"efs",
	"ceph",
}

func (dt DriverType) String() string {
	return driverTypes[dt]
}
