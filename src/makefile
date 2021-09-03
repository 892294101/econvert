GOCMD=go
GOBUILD=${GOCMD} build

BUILD_DIR=../build/econvert
BINARY_DIR=$(BUILD_DIR)/bin

LGOOS=$(shell echo `go version`|cut -f4 -d ' '|cut -d '/' -f1)
LGOARCH=$(shell echo `go version`|cut -f4 -d ' '|cut -d '/' -f2)
LDATE=$(shell date)
VERSION=1.02
GOVERSION=$(shell go version)

all: build
build:
	${GOBUILD} -gcflags=all='-l -N' -ldflags " \
	-X 'main.Platform=$(LGOOS) $(LGOARCH)' \
	-X 'main.BuildTime=$(LDATE)' \
	-X 'main.GoVersion=$(GOVERSION)' \
	-X 'main.VERSION=$(VERSION)'" -o ${BINARY_DIR}/econvert


