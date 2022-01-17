# go-httpserver-demo
Demo for learning go httpserver

## Requirement

- 接收客户端 request，并将 request 中带的 header 写入 response header
- 读取当前系统的环境变量中的 VERSION 配置，并写入 response header
- Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
- 当访问 localhost/healthz 时，应返回 200

## Useful make command

Run unittest
```
make unittest
```

Run unittest, then build docker image, launch the httpserver container and do the
test with curl.
```
make test
```

Release the docker image. This will execute unittest, build docker image, launch 
the container just build to run integration test, and then push the image to docker
registry.
```
make release
```

## Kubernetes

[Kubernetes installation on GCP](k8s-install)