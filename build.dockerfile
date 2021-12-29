FROM golang:1.17.5-alpine3.15

WORKDIR /app
COPY . .

ENV GOPROXY="https://mirrors.aliyun.com/goproxy/"

RUN go get -d -v ./... && go install -v ./...
