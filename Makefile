GO_FMT = gofmt -s -w -l .
GO_XC = goxc -os="linux freebsd openbsd netbsd"

GOXC_FILE = .goxc.local.json

all: deps

compile: goxc

goxc:
	$(shell echo '{\n "ConfigVersion": "0.9",' > $(GOXC_FILE))
	$(shell echo ' "TaskSettings": {' >> $(GOXC_FILE))
	$(shell echo '  "bintray": {\n   "apikey": "$(BINTRAY_APIKEY)"' >> $(GOXC_FILE))
	$(shell echo '  }\n } \n}' >> $(GOXC_FILE))
	$(GO_XC) 

deps:
	go get

format: 
	$(GO_FMT) 

bintray:
	$(GO_XC) bintray
