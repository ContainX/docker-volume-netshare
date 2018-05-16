# Docker NFS, AWS EFS & Samba/CIFS Volume Plugin

[![Build Status](https://travis-ci.org/ContainX/docker-volume-netshare.svg)](https://travis-ci.org/ContainX/docker-volume-netshare) [![release](http://github-release-version.herokuapp.com/github/ContainX/docker-volume-netshare/release.svg?style=flat)](https://github.com/ContainX/docker-volume-netshare/releases/latest)

Mount NFS v3/4, AWS EFS or CIFS inside your docker containers.  This is a docker plugin which enables these volume types to be directly mounted within a container.

### Documentation [http://netshare.containx.io](http://netshare.containx.io)
### Chat/Slack [https://containx.slack.com](https://containx.slack.com)

## NFS Prerequisites on Linux

NFS needs to be installed on Linux systems in order to properly mount NFS mounts.  

- For Ubuntu/Debian: `sudo apt-get install -y nfs-common`
- For RHEL/CentOS: `sudo yum install -y nfs-utils`

It is recommend to try mounting an NFS volume to eliminate any configuration issues prior to running the plugin:
```
sudo mount -t nfs4 1.1.1.1:/mountpoint /target/mount
```

## Installation

#### From Source

```
$ go get github.com/ContainX/docker-volume-netshare
$ go build
```

#### From Binaries

Binaries are available through GitHub releases.  You can download the appropriate binary, package and version from the [Releases](https://github.com/ContainX/docker-volume-netshare/releases) page

#### On Ubuntu / Debian

The method below will install the sysvinit and /etc/default options that can be overwritten during service start.

1. Install the Package

```
  $ wget https://github.com/ContainX/docker-volume-netshare/releases/download/v0.34/docker-volume-netshare_0.34_amd64.deb
  $ sudo dpkg -i docker-volume-netshare_0.34_amd64.deb
```

2. Modify the startup options in `/etc/default/docker-volume-netshare`
3. Start the service `service docker-volume-netshare start`


## Usage

### Launching in NFS mode

**1. Run the plugin - can be added to systemd or run in the background**

```
  $ sudo docker-volume-netshare nfs
```
If you are not using the latest stable version of docker engine please specify the version with flag.
For example:
```
  $ sudo docker-volume-netshare nfs -a 1.35
```


**2. Launch a container**

```
  $ docker run -i -t --volume-driver=nfs -v nfshost/path:/mount ubuntu /bin/bash
```

### Launching in EFS mode

**1. Run the plugin - can be added to systemd or run in the background**

```
  // With File System ID resolution to AZ / Region URI
  $ sudo docker-volume-netshare efs
  // For VPCs without AWS DNS - using IP for Mount
  $ sudo docker-volume-netshare efs --noresolve
```

**2. Launch a container**

```
  // Launching a container using the EFS File System ID
  $ docker run -i -t --volume-driver=efs -v fs-2324532:/mount ubuntu /bin/bash
  // Launching a container using the IP Address of the EFS mount point (--noresolve flag in plugin)
  $ docker run -i -t --volume-driver=efs -v 10.2.3.1:/mount ubuntu /bin/bash
```

### Launching in Samba/CIFS mode

#### Docker Version < 1.9.0

**1. Run the plugin - can be added to systemd or run in the background**

```
  $ sudo docker-volume-netshare cifs --username user --password pass --domain domain --security security
```

**2. Launch a container**

```
  // In CIFS the "//" is omitted and handled by netshare
  $ docker run -it --volume-driver=cifs -v cifshost/share:/mount ubuntu /bin/bash
```

##### .NetRC support

.NetRC is fully support eliminating users and passwords to be specified in step 1.  To use .netrc do the following steps:

**1. Create a /root/.netrc file (since netshare needs to be run as a root user).  Add the host and credential mappings.**  

See example:

```
  //.netrc
  machine some_hostname
       username  jeremy
       password  somepass
       domain    optional
       security  optional
       fileMode  optional
       dirMode   optional
```

**2. Run the plugin**

```
  $ sudo docker-volume-netshare cifs
```

**3. Launch a container**

```
  // In CIFS the "//" is omitted and handled by netshare
  $ docker run -it --volume-driver=cifs -v cifshost/share:/mount ubuntu /bin/bash
```

#### Docker Version 1.9.0+

Docker 1.9.0 now has support for volume management.  This allows you to user `docker volume create` to define a volume by name so
options and other info can be eliminated when running a container.

**1. Run the plugin - can be added to systemd or run in the background**

```
  $ sudo docker-volume-netshare cifs
```

**2. Create a Volume**

This will create a new volume via the Docker daemon which will call `Create` in netshare passing in the corresponding user, pass and domain info.

```
  $ docker volume create -d cifs --name cifshost/share --opt username=user --opt password=pass --opt domain=domain --opt security=security --opt fileMode=0777 --opt dirMode=0777
```

**3. Launch a container**

```
  // cifs/share matches the volume as defined in Step #2 using docker volume create
  $ docker run -it -v cifshost/share:/mount ubuntu /bin/bash
```

#### Security Option
Some CIFS servers may require a specific security mode to connect. The ``security`` option defines the ``sec`` option that is passed to ``mount.cifs``. [More information about available ``sec`` options](https://www.samba.org/samba/docs/man/manpages-3/mount.cifs.8.html).
e.g.: Apple Time Capsule's require the security mode ``ntlm``.

## License

This software is licensed under the Apache 2 license, quoted below.

Copyright 2016 ContainX / Jeremy Unruh

Licensed under the Apache License, Version 2.0 (the "License"); you may not use this file except in compliance with the License. You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0
Unless required by applicable law or agreed to in writing, software distributed under the License is distributed on an "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied. See the License for the specific language governing permissions and limitations under the License.
