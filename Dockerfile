FROM alpine:latest

WORKDIR /app

ADD build/httpserver /app/httpserver

EXPOSE 8000

ENV VERSION=v1.1

CMD ["/app/httpserver"]
