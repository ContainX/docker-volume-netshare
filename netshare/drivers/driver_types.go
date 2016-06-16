package drivers

type DriverType int

const (
	CIFS DriverType = iota
	NFS
	EFS
	CEPH
)

var driverTypes = []string{
	"cifs",
	"nfs",
	"efs",
	"ceph",
}

func (dt DriverType) String() string {
	return driverTypes[dt]
}
