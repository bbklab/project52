.PHONY: default all prepare build clean

PWD := $(shell pwd)
BUNDLES := $(shell pwd)/bundles
IMG := "golang:1.8.1"

default: build

prepare:
	mkdir -p $(BUNDLES)

build: prepare build-master build-agent

build-master:
	docker run --rm -v $(PWD):/tmp/mole:ro -v $(BUNDLES):/product:rw -w /tmp/mole $(IMG) go build -o /product/master ./examples/master

build-agent:
	docker run --rm -v $(PWD):/tmp/mole:ro -v $(BUNDLES):/product:rw -w /tmp/mole $(IMG) go build -o /product/agent ./examples/agent

run-master: build-master
	env MOLE_ROLE=master MOLE_LISTEN=:9999 $(BUNDLES)/master

run-agent: build-agent
	env MOLE_ROLE=agent MOLE_MASTER_ENDPOINT=http://0.0.0.0:9999 MOLE_BACKEND_ENDPOINT=unix:///var/run/docker.sock $(BUNDLES)/agent

clean:
	rm -rfv $(BUNDLES)
