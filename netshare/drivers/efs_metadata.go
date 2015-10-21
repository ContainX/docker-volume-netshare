package drivers

import (
	"encoding/json"
	"net/http"
)

const (
	MetaDataURL = "http://169.254.169.254/latest/dynamic/instance-identity/document"
)

type metaData struct {
	AvailZone string `json:"availabilityZone,omitempty"`
	Region    string `json:"region,omitempty"`
}

func fetchAWSMetaData() (*metaData, error) {
	r, err := http.Get(MetaDataURL)
	if err != nil {
		return nil, err
	}
	defer r.Body.Close()

	md := &metaData{}
	json.NewDecoder(r.Body).Decode(md)
	return md, nil
}
