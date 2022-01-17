FROM golang:1.17.5-alpine3.15 AS build

WORKDIR /app
COPY . .

ENV GOPROXY="https://mirrors.aliyun.com/goproxy/"
ENV CGO_ENABLED=0
RUN go get -d -v ./... && go install -v ./... && go build  -v -o /app/build/httpserver

FROM scratch
COPY --from=build /app/build/httpserver /app/httpserver
EXPOSE 8000

ENV VERSION=v1.1
ENTRYPOINT [ "/app/httpserver" ]