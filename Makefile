.SILENT: setup clean
.PHONY: setup clean

build:
	make clean && make setup && make compile

compile:
	GOOS="linux" GOARCH="amd64" go build -o "bin/traefik-healthcheck-linux-amd64" traefik-healthcheck.go
	GOOS="linux" GOARCH="386" go build -o "bin/traefik-healthcheck-linux-386" traefik-healthcheck.go
	GOOS="darwin" GOARCH="amd64" go build -o "bin/traefik-healthcheck-darwin-amd64" traefik-healthcheck.go
	GOOS="darwin" GOARCH="386" go build -o "bin/traefik-healthcheck-dawrin-386" traefik-healthcheck.go

setup:
	mkdir bin

clean:
	rm -rf bin
