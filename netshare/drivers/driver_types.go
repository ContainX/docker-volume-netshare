package drivers

type DriverType int

const (
	CIFS DriverType = iota
	NFS
	EFS
)

var driverTypes = []string{
	"cifs",
	"nfs",
	"efs",
}

func (dt DriverType) String() string {
	return driverTypes[dt]
}
