package main

import (
	"github.com/gondor/docker-volume-netshare/netshare"
)

func main() {
	netshare.Execute()
	//	flag.Parse()

	//	d := newNfsDriver(*root, *version)
	//	h := dkvolume.NewHandler(d)
	//	fmt.Println(h.ServeTCP("nfs", ":8888"))
	//	fmt.Println(h.ServeUnix("", "nfs"))
}
