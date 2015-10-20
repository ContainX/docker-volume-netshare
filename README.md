# Docker NFS, NFS4, Samba/CIFS Volume Plugin

[![Build Status](https://travis-ci.org/gondor/docker-volume-netshare.svg)](https://travis-ci.org/gondor/docker-volume-netshare)

Mount NFS v3,4 or CIFS inside your docker containers.  This is a docker plugin which enables these volume types to be directly mounted within a container.

## Installation

#### From Source

```
$ go get github.com/gondor/docker-volume-netshare
$ go build
```

#### From Binaries

* Architecture i386 [ [linux](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_linux_386.tar.gz?direct) / [netbsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_netbsd_386.zip?direct) / [freebsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_freebsd_386.zip?direct) / [openbsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_openbsd_386.zip?direct) ]
 * Architecture amd64 [ [linux](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_linux_amd64.tar.gz?direct) / [netbsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_netbsd_amd64.zip?direct) / [freebsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_freebsd_amd64.zip?direct) / [openbsd](https://dl.bintray.com//content/pacesys/docker/docker-volume-netshare_0.1_openbsd_amd64.zip?direct) ]

## Usage

#### Launching in NFS mode

*1. Run the plugin - can be added to systemd or run in the background*

```
  $ sudo docker-volume-netshare nfs
```

*2. Launch a container*

```
  $ docker run -i -t --volume-driver=nfs -v nfshost/path:/mount ubuntu /bin/bash
```

#### Launching in Samba/CIFS mode

*1. Run the plugin - can be added to systemd or run in the background*

```
  $ sudo docker-volume-netshare samba --username smbuser --password smbpass --workgroup workgroup
```

*2. Launch a container*

```
  $ docker run -i -t --volume-driver=smb -v nfshost/path:/mount ubuntu /bin/bash
```

## License

This software is licensed under the Apache 2 license, quoted below.

Copyright 2015 Jeremy Unruh

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
