FROM golang:1.5

## dkv-netshare is BASE image used by CIFS, NFS tafs
##

COPY . /go/src/app
WORKDIR /go/src/app
RUN go-wrapper download && go-wrapper install && go build -o docker-volume-netshare && cp docker-volume-netshare /bin
