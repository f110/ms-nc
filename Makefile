VERSION = $(shell git rev-parse HEAD)
LDFALGS     = -ldflags "-X \"main.Version=${VERSION}\""

GOOS    = $(shell go env GOOS)
GOARCH  = $(shell go env GOARCH)
GOBUILD = go build ${LDFALGS} -v

build:
	${GOBUILD} ms-nc.go sender.go receiver.go
