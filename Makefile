VERSION = 0.2

GO_FMT = gofmt -s -w -l .
GO_XC = goxc -os="linux freebsd openbsd netbsd"

GOXC_FILE = .goxc.local.json

all: deps

compile: goxc

goxc:
	$(shell echo '{\n "ArtifactsDest": "build",\n "ConfigVersion": "0.9",' > $(GOXC_FILE))
	$(shell echo ' "PackageVersion": "$(VERSION)",\n "TaskSettings": {' >> $(GOXC_FILE))
	$(shell echo '  "bintray": {\n   "user": "gondor",\n   "apikey": "$(BINTRAY_APIKEY)",\n   "package": "docker-volume-netshare",' >> $(GOXC_FILE))
	$(shell echo '   "repository": "docker",\n   "subject": "pacesys"' >> $(GOXC_FILE))
	$(shell echo '  }\n }\n}' >> $(GOXC_FILE))
	$(GO_XC) 

deps:
	go get

format: 
	$(GO_FMT) 

bintray:
	$(GO_XC) bintray
