ROOT_DIR:=$(shell dirname $(realpath $(lastword $(MAKEFILE_LIST))))

build:
	docker run --rm -v $(ROOT_DIR):/src -v /var/run/docker.sock:/var/run/docker.sock centurylink/golang-builder thraxil/chimney

push: build
	docker push thraxil/chimney
