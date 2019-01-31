ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

chimney: *.go
	go build .

build:
	docker run --rm -v $(ROOT_DIR):/src -v /var/run/docker.sock:/var/run/docker.sock centurylink/golang-builder ccnmtl/chimney

push: build
	docker push ccnmtl/chimney
