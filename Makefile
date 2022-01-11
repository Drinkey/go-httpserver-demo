CUR_DIR:=$(shell pwd)
TEST_RESULT:="FAIL"

local-build:
	go build -v -o httpserver

unittest:
	go test -v ./...

docker-build:
	docker build -t httpserver-build:latest -f build.dockerfile .

docker-image:
	docker build -t drinkey/httpserver:latest .

build: docker-build
	echo $(CUR_DIR)
	docker run --rm -v $(CUR_DIR):/app/ -w /app httpserver-build:latest go build -v -o build/httpserver

run: build docker-image
	docker run -d -p 80:8000 --name httpserver drinkey/httpserver:latest

stop:
	docker rm -f httpserver
	docker ps -a

test: unittest run
	curl http://localhost/healthz

push: run test
	docker login
	docker push drinkey/httpserver:latest

release: push stop
	@echo "Push success, service stopped"
