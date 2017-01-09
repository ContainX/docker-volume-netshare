FROM golang:1.6

## dkv-netshare is BASE image used by CIFS, NFS tafs
##

RUN mkdir -p /go/src/github.com/ContainX/docker-volume-netshare
COPY . /go/src/github.com/ContainX/docker-volume-netshare
WORKDIR /go/src/github.com/ContainX/docker-volume-netshare
#RUN go-wrapper download && go-wrapper install && go build -o docker-volume-netshare && cp docker-volume-netshare /bin
