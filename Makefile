#ARCH
GOOS ?= $(shell go env GOOS)
GOARCH ?= $(shell go env GOARCH)
build:
	CGO_ENABLED=0 GOOS=$(GOOS) GOARCH=$(GOARCH) go build -o gvm_$(GOOS)_$(GOARCH) -tags 'netgo osusergo' -ldflags '-extldflags "-static"'