package drivers

import (
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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

func mountpoint(root, name string) string {
	return filepath.Join(root, name)
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