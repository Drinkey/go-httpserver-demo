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

## Enter Docker Namespace

```sh
cloud-native-instance-1:~$ pid=`sudo docker inspect httpserver -f "{{.State.Pid}}"`
cloud-native-instance-1:~$ echo $pid
59298
cloud-native-instance-1:~$ sudo nsenter -t $pid -n ip addr show
1: lo: <LOOPBACK,UP,LOWER_UP> mtu 65536 qdisc noqueue state UNKNOWN group default qlen 1000
    link/loopback 00:00:00:00:00:00 brd 00:00:00:00:00:00
    inet 127.0.0.1/8 scope host lo
       valid_lft forever preferred_lft forever
5: eth0@if6: <BROADCAST,MULTICAST,UP,LOWER_UP> mtu 1500 qdisc noqueue state UP group default
    link/ether 02:42:ac:11:00:02 brd ff:ff:ff:ff:ff:ff link-netnsid 0
    inet 172.17.0.2/16 brd 172.17.255.255 scope global eth0
       valid_lft forever preferred_lft forever
cloud-native-instance-1:~$ sudo nsenter -t $pid -n ip ro show
default via 172.17.0.1 dev eth0
172.17.0.0/16 dev eth0 proto kernel scope link src 172.17.0.2
cloud-native-instance-1:~$
```

## Kubernetes

[Kubernetes installation on GCP](k8s-install)