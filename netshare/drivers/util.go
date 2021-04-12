package drivers

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	ShareSplitIndentifer = "#"
)

func createDest(dest string) error {
	fi, err := os.Lstat(dest)

	if os.IsNotExist(err) {
		if err := os.MkdirAll(dest, 0755); err != nil {
			return err
		}
	} else if err != nil {
		return err
	}

	if fi != nil && !fi.IsDir() {
		return fmt.Errorf("%v already exist and it's not a directory", dest)
	}
	return nil
}

// Used to support on the fly volume creation using docker run. If = is in the name we split
// and elem[1] is the volume name
func resolveName(name string) (string, map[string]string) {
	if strings.Contains(name, ShareSplitIndentifer) {
		sharevol := strings.Split(name, ShareSplitIndentifer)
		opts := map[string]string{}
		opts[ShareOpt] = sharevol[0]
		opts[CreateOpt] = "true"
		return sharevol[1], opts
	}
	return name, nil
}

func shareDefinedWithVolume(name string) bool {
	return strings.Contains(name, ShareSplitIndentifer)
}

func addShareColon(share string) string {
	if strings.Contains(share, ":") {
		return share
	}
	source := strings.Split(share, "/")
	source[0] = source[0] + ":"
	return strings.Join(source, "/")
}

func mountpoint(elem ...string) string {
	return filepath.Join(elem...)
}

func run(cmd string) error {
	if out, err := exec.Command("sh", "-c", cmd).CombinedOutput(); err != nil {
		log.Println(string(out))
		return err
	}
	return nil
}

func merge(src, src2 map[string]string) map[string]string {
	if len(src) == 0 && len(src2) == 0 {
		return EmptyMap
	}

	dst := map[string]string{}
	for k, v := range src2 {
		dst[k] = v
	}
	for k, v := range src {
		dst[k] = v
	}
	return dst
}

func checkFile(file string) error {
	_, err := os.Stat(file)
	if os.IsNotExist(err) {
		_, err := os.Create(file)
		if err != nil {
			return err
		}
	}
	return nil
}

func readState(f string) (map[string]map[string]string, error) {
	checkFile(f)
	var jsonData map[string]map[string]string
	jsonFile, _ := ioutil.ReadFile(f)
	err := json.Unmarshal([]byte(jsonFile), &jsonData)
	res := map[string]map[string]string{}
	// Remove volumes with non existent paths
	for k, v := range jsonData {
		if _, err := os.Stat(v["Path"]); !os.IsNotExist(err) {
			res[k] = map[string]string{
				"Path": v["Path"],
			}
		}
	}
	return res, err
}

func writeState(f string, d map[string]map[string]string) error {
	jsonData, _ := json.Marshal(d)
	err := ioutil.WriteFile(f, jsonData, 0644)
	return err
}
