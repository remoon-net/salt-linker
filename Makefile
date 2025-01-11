build:
	CGO_ENABLED=0 go build  -ldflags="-X 'main.Version=$$(git describe --tags --always --dirty)' -s -w" -o salt-linker .
docker: build
	docker build . -t shynome/salt-linker:$$(git describe --tags --always --dirty)
push: docker
	docker push shynome/salt-linker:$$(git describe --tags --always --dirty)
