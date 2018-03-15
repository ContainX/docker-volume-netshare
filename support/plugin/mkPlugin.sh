#!/bin/bash -e
if [ ! "$1" ]
then
    echo "miss $1 for plugin name"
    exit
else
    PLUGIN_NAME=$1
fi

for sharetype in nfs ceph efs cifs
do
    chmod +x ./docker-volume-netshare
    TMPDIR=/tmp/docker-volume-netshare
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < Dockerfile.tmpl > Dockerfile
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < config.json.tmpl > config.json
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < netshare.sh.tmpl > netshare.sh
    rm -rf $TMPDIR
    docker build -t netshare .
    id=$(docker create netshare true)
    mkdir -p $TMPDIR/rootfs
    cp ./config.json $TMPDIR/
    docker export "$id" | sudo tar -x -C $TMPDIR/rootfs
    docker rm -vf "$id"
    docker rmi netshare 
    docker plugin create $PLUGIN_NAME-${sharetype} $TMPDIR
done
