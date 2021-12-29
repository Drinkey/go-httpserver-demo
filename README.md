# go-httpserver-demo
Demo for learning go httpserver

## Requirement

- 接收客户端 request，并将 request 中带的 header 写入 response header
- 读取当前系统的环境变量中的 VERSION 配置，并写入 response header
- Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
- 当访问 localhost/healthz 时，应返回 200

## Command

Run the following command will build the httpserver binary and build the docker image to run the server, then launch a container to run the service on `0.0.0.0:80`

```sh
$ make run
```