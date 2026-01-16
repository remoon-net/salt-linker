VERSION := $(shell git describe --tags --always --dirty)

build:
	CGO_ENABLED=0 go build  -ldflags="-X 'main.Version=${VERSION}' -s -w" -o salt-linker .
docker: build frontend
	docker build . -t shynome/salt-linker:${VERSION}
push: docker
	docker push shynome/salt-linker:${VERSION}
frontend:
	cd ../salt.remoon.net && npm run build && \
	cd - && \
	rm -rf ./pb_public/ && cp -r ../salt.remoon.net/build/ ./pb_public/
