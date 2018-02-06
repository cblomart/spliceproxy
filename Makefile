GOVERSION=$(shell go version)
GOOS=$(word 1,$(subst /, ,$(lastword $(GOVERSION))))
GOARCH=$(word 2,$(subst /, ,$(lastword $(GOVERSION))))
RELEASE_DIR=releases
SRC_FILES=$(wildcard *.go)
MUSL_BUILD_FLAGS=-ldflags '-linkmode external -s -w -extldflags "-static"' -a
BUILD_FLAGS=-ldflags -s -a
MUSL_CC=musl-gcc
MUSL_CCGLAGS="-static"
PROGRAM=spliceproxy

deps:
	go get -u -t ./...

build-windows-amd64:
	@$(MAKE) build GOOS=windows GOARCH=amd64 SUFFIX=.exe

dist-windows-amd64:
	@$(MAKE) dist GOOS=windows GOARCH=amd64 SUFFIX=.exe

build-linux-amd64:
	CC=$(MUSL_CC) CCGLAGS=$(MUSL_CCGLAGS) go build $(MUSL_BUILD_FLAGS) -o $(RELEASE_DIR)/linux/amd64/$(PROGRAM) .
	upx -qq --best $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)
	cp vsphere-graphite-example.json $(RELEASE_DIR)/linux/amd64/config.sample.yaml
	cp -Rf systemd $(RELEASE_DIR)/linux/amd64/

dist-linux-amd64:
	@$(MAKE) dist GOOS=linux GOARCH=amd64

build-darwin-amd64:
	@$(MAKE) build GOOS=darwin GOARCH=amd64

dist-darwin-amd64:
	@$(MAKE) dist GOOS=darwin GOARCH=amd64
    
build-linux-arm:
	@$(MAKE) build GOOS=linux GOARCH=arm GOARM=5

dist-linux-arm:
	@$(MAKE) dist GOOS=linux GOARCH=arm GOARM=5

docker: $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)
	cp $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/* docker/
	mkdir -p docker/etc
	cp config.sample.yaml docker/etc/config.yaml
	docker build -f docker/Dockerfile -t cblomart/$(PREFIX)$(PROGRAM) docker

docker-linux-amd64:
	@$(MAKE) docker GOOS=linux GOARCH=amd64

docker-linux-arm:
	@$(MAKE) docker GOOS=linux GOARCH=arm PREFIX=rpi-

docker-darwin-amd64: ;

docker-windows-amd64: ;

checks:
	go get honnef.co/go/tools/cmd/gosimple
	go get github.com/golang/lint/golint
	go get github.com/gordonklaus/ineffassign
	go get github.com/GoASTScanner/gas/cmd/gas/...
	gosimple ./...
	gofmt -s -d .
	go vet ./...
	golint ./...
	ineffassign ./
	gas ./...
	go tool vet ./..

$(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)$(SUFFIX): $(SRC_FILES)
	go build $(BUILD_FLAGS) -o $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)$(SUFFIX) .
	upx -qq --best $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)/$(SUFFIX)
	cp config.sample.yaml $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/config.sample.yaml
	cp -Rf systemd $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/

$(RELEASE_DIR)/$(PROGRAM)_$(GOOS)_$(GOARCH).tgz: $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)$(SUFFIX)
	cd $(RELEASE_DIR)/$(GOOS)/$(GOARCH); tar czf /tmp/$(PROGRAM)_$(GOOS)_$(GOARCH).tgz ./$(PROGRAM)$(SUFFIX) ./confg.sample.yaml ./systemd

dist: $(RELEASE_DIR)/$(PROGRAM)_$(GOOS)_$(GOARCH).tgz

build: $(RELEASE_DIR)/$(GOOS)/$(GOARCH)/$(PROGRAM)$(SUFFIX)

clean:
	rm -rf $(RELEASE_DIR)
	
all:
	@$(MAKE) dist-windows-amd64 
	@$(MAKE) dist-linux-amd64
	@$(MAKE) dist-darwin-amd64
	@$(MAKE) dist-linux-arm
