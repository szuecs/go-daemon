.PHONY: clean clean check build.docker scm-source

BINARY_BASE   ?= go-daemon
IMAGE_NAME    ?= $(BINARY_BASE)
VERSION       ?= $(shell git describe --tags --always --dirty)
IMAGE         ?= pierone.stups.zalan.do/teapot/$(IMAGE_NAME)
TAG           ?= $(VERSION)
DOCKERFILE    ?= Dockerfile
GITHEAD       = $(shell git rev-parse --short HEAD)
GITURL        = $(shell git config --get remote.origin.url)
GITSTATUS     = $(shell git status --porcelain || echo "no changes")
SOURCES       = $(shell find . -name '*.go')
BUILD_FLAGS   ?= -v
LDFLAGS       ?= -X main.Version=$(VERSION) -X main.Buildstamp=$(shell date -u '+%Y-%m-%d_%I:%M:%S%p') -X main.Githash=$(shell git rev-parse HEAD)

default: build.local

clean:
	rm -rf build
	rm -f test/bench.*
	rm -f test/prof.*
	find . -name '*.test' -delete

config:
	@test -d ~/.config/$(BINARY_BASE) || mkdir -p ~/.config/$(BINARY_BASE)
	@test -e ~/.config/$(BINARY_BASE)/config.yaml || cp config.yaml.sample ~/.config/$(BINARY_BASE)/config.yaml
	@echo "modify ~/.config/$(BINARY_BASE)/config.yaml as you need"

check:
	golint ./... | egrep -v '^vendor/'
	go vet -v ./... 2>&1 | egrep -v '^(vendor/|exit status 1)'

build.local: build/$(BINARY_BASE)
build.linux: build/linux/$(BINARY_BASE)
build.osx: build/osx/$(BINARY_BASE)

build/$(BINARY_BASE): $(SOURCES)
	go build -o build/"$(BINARY_BASE)" "$(BUILD_FLAGS)" -ldflags "$(LDFLAGS)" -tags zalandoValidation .

build/linux/$(BINARY_BASE): $(SOURCES)
	GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/linux/"$(BINARY_BASE)" -ldflags "$(LDFLAGS)" -tags zalandoValidation .

build/osx/$(BINARY_BASE): $(SOURCES)
	GOOS=darwin GOARCH=amd64 CGO_ENABLED=0 go build "$(BUILD_FLAGS)" -o build/osx/"$(BINARY_BASE)" -ldflags "$(LDFLAGS)" -tags zalandoValidation .

$(DOCKERFILE).upstream: $(DOCKERFILE)
	sed "s@UPSTREAM@$(shell $(shell head -1 $(DOCKERFILE) | sed -E 's@FROM (.*)/(.*)/(.*):.*@pierone tags \2 \3 --url \1@') | awk '{print $$3}' | tail -1)@" $(DOCKERFILE) > $(DOCKERFILE).upstream

build.docker: $(DOCKERFILE).upstream scm-source.json build.linux
	docker build --rm -t "$(IMAGE):$(TAG)" -f $(DOCKERFILE).upstream .

build.push: build.docker
	docker push "$(IMAGE):$(TAG)"

scm-source.json: .git
	scm-source
