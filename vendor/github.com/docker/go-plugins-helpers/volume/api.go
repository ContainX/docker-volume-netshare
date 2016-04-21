package volume

import (
	"net/http"

	"github.com/docker/go-plugins-helpers/sdk"
)

const (
	// DefaultDockerRootDirectory is the default directory where volumes will be created.
	DefaultDockerRootDirectory = "/var/lib/docker-volumes"

	manifest        = `{"Implements": ["VolumeDriver"]}`
	createPath      = "/VolumeDriver.Create"
	getPath         = "/VolumeDriver.Get"
	listPath        = "/VolumeDriver.List"
	removePath      = "/VolumeDriver.Remove"
	hostVirtualPath = "/VolumeDriver.Path"
	mountPath       = "/VolumeDriver.Mount"
	unmountPath     = "/VolumeDriver.Unmount"
)

// Request is the structure that docker's requests are deserialized to.
type Request struct {
	Name    string
	Options map[string]string `json:"Opts,omitempty"`
}

// Response is the strucutre that the plugin's responses are serialized to.
type Response struct {
	Mountpoint string
	Err        string
	Volumes    []*Volume
	Volume     *Volume
}

// Volume represents a volume object for use with `Get` and `List` requests
type Volume struct {
	Name       string
	Mountpoint string
}

// Driver represent the interface a driver must fulfill.
type Driver interface {
	Create(Request) Response
	List(Request) Response
	Get(Request) Response
	Remove(Request) Response
	Path(Request) Response
	Mount(Request) Response
	Unmount(Request) Response
}

// Handler forwards requests and responses between the docker daemon and the plugin.
type Handler struct {
	driver Driver
	sdk.Handler
}

type actionHandler func(Request) Response

// NewHandler initializes the request handler with a driver implementation.
func NewHandler(driver Driver) *Handler {
	h := &Handler{driver, sdk.NewHandler(manifest)}
	h.initMux()
	return h
}

func (h *Handler) initMux() {
	h.handle(createPath, func(req Request) Response {
		return h.driver.Create(req)
	})

	h.handle(getPath, func(req Request) Response {
		return h.driver.Get(req)
	})

	h.handle(listPath, func(req Request) Response {
		return h.driver.List(req)
	})

	h.handle(removePath, func(req Request) Response {
		return h.driver.Remove(req)
	})

	h.handle(hostVirtualPath, func(req Request) Response {
		return h.driver.Path(req)
	})

	h.handle(mountPath, func(req Request) Response {
		return h.driver.Mount(req)
	})

	h.handle(unmountPath, func(req Request) Response {
		return h.driver.Unmount(req)
	})
}

func (h *Handler) handle(name string, actionCall actionHandler) {
	h.HandleFunc(name, func(w http.ResponseWriter, r *http.Request) {
		var req Request
		if err := sdk.DecodeRequest(w, r, &req); err != nil {
			return
		}

		res := actionCall(req)

		sdk.EncodeResponse(w, res, res.Err)
	})
}
