.PHONY: build build-web install clean test

build:
	go build -o gococo ./cmd/gococo/

build-web:
	cd web && npm install && npm run build

install: build
	go install ./cmd/gococo/

clean:
	rm -f gococo
	rm -rf web/dist web/node_modules

test:
	go test ./...
