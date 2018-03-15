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
    TMPDIR=./plugin-build-${sharetype}
    rm -rf $TMPDIR
    mkdir -p $TMPDIR
    cp ./docker-volume-netshare ${TMPDIR}/docker-volume-netshare
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < support/plugin/Dockerfile.tmpl > ${TMPDIR}/Dockerfile
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < support/plugin/netshare.sh.tmpl > ${TMPDIR}/netshare.sh
    docker build -t netshare ${TMPDIR}
    rm ${TMPDIR}/Dockerfile ${TMPDIR}/netshare.sh
    SHARE_TYPE=${sharetype} envsubst '$SHARE_TYPE' < support/plugin/config.json.tmpl > ${TMPDIR}/config.json
    id=$(docker create netshare true)
    mkdir -p $TMPDIR/rootfs
    docker export "$id" | sudo tar -x -C $TMPDIR/rootfs
    docker rm -vf "$id"
    docker rmi netshare 
    sudo docker plugin create $PLUGIN_NAME-${sharetype} $TMPDIR
done
