test:
	go test -v ./...

docker-build:
	docker build -t httpserver-build:latest -f build.dockerfile .

docker-image:
	docker build -t httpserver:latest .

build: docker-build
	docker run --rm -v "$(PWD)":/app/ -w /app httpserver-build:latest go build -v -o build/httpserver

run: build docker-image
	docker run -d -p 80:8000 --name httpserver httpserver:latest

stop:
	docker rm -f httpserver
	docker ps -a

local-build:
	go build -v -o httpserver

release: build
	tar zcvf goat-linux-amd64.tar.gz goat-linux-amd64
	export GOARCH=amd64
	export GOOS=linux
	export CGO_ENABLED=1
	go build -v -o goat-linux-amd64
	tar zcvf goat-`git tag |sort|tail -n1`-linux-amd64.tar.gz goat-linux-amd64
