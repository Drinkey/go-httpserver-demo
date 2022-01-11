CUR_DIR:=$(shell pwd)

local-build:
	go build -v -o httpserver

unittest:
	go test -v ./...

docker-build:
	docker build -t httpserver-build:latest -f build.dockerfile .

build-docker-image:
	docker build -t drinkey/httpserver:latest .

build: unittest docker-build
	docker run --rm -v $(CUR_DIR):/app/ -w /app httpserver-build:latest go build -v -o build/httpserver

run: build build-docker-image
	docker run -d -p 80:8000 --name httpserver drinkey/httpserver:latest

stop:
	docker rm -f httpserver
	docker ps -a

servicetest: run
	curl http://localhost/healthz
	curl http://localhost/
	@echo "Service Test PASSED"

test: servicetest stop
	@echo "All Tests PASSED"

push:
	docker login
	docker push drinkey/httpserver:latest

release: build test push
	@echo "Push success, service stopped"
