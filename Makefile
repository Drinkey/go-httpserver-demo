CUR_DIR:=$(shell pwd)

local-build:
	go build -v -o httpserver

unittest:
	go test -v ./...

build: unittest
	docker build -t drinkey/httpserver:latest .

run: build
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
