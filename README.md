# go-httpserver-demo
Demo for learning go httpserver

README Table of Content
- [go-httpserver-demo](#go-httpserver-demo)
- [Module 2](#module-2)
  - [Requirement](#requirement)
  - [Useful make commands](#useful-make-commands)
  - [Graceful shutdown](#graceful-shutdown)
- [Module 3](#module-3)
  - [Requirement](#requirement-1)
  - [Enter Docker Namespace](#enter-docker-namespace)
- [Kubernetes](#kubernetes)
  - [Configure ConfigMap](#configure-configmap)
  - [Create Deployment](#create-deployment)
  - [Create service](#create-service)
  - [Viewing pod logs](#viewing-pod-logs)
  - [Probes](#probes)

# Module 2
## Requirement

- 接收客户端 request，并将 request 中带的 header 写入 response header
- 读取当前系统的环境变量中的 VERSION 配置，并写入 response header
- Server 端记录访问日志包括客户端 IP，HTTP 返回码，输出到 server 端的标准输出
- 当访问 localhost/healthz 时，应返回 200

## Useful make commands

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

## Graceful shutdown

Listen to SIGINT and SIGTERM for graceful shutdown the http server, read the 
signal from a channel, and block until signal received. Once signal received, 
start initiating a grace shutdown.

Here is the test:

Start the http server. Then start another shell to send interrupt signal to the process:
```
$ kill -2 <pid_of_server_process>
```

Then observed output at the server side:
```
$ ./go-httpserver-demo
2022/01/17 15:07:49 Starting http server
2022/01/17 15:07:49 Server started on :8000
2022/01/17 15:08:10 Got signal interrupt   #<<<<<< Comment: Received SIGINT >>>>>>
2022/01/17 15:08:10 Server properly stopped
2022/01/17 15:08:10 Running clean up...
```

Try SIGTERM
```
$ kill -15 <pid_of_server_process>
```

Then observed output at the server side:
```
$ ./go-httpserver-demo
2022/01/17 15:10:10 Starting http server
2022/01/17 15:10:10 Server started on :8000
2022/01/17 15:10:32 Got signal terminated   #<<<<<< Comment: Received SIGTERM >>>>>>
2022/01/17 15:10:32 Server properly stopped
2022/01/17 15:10:32 Running clean up...
```

# Module 3

## Requirement
- 构建本地镜像
- 编写 Dockerfile 将练习 2.2 编写的 httpserver 容器化
  - [Dockerfile](Dockerfile)
- 将镜像推送至 docker 官方镜像仓库
  - [Makefile](Makefile) `make push` or `make release` to push the image to hub.docker.com
- 通过 docker 命令本地启动 httpserver
  - [Makefile](Makefile) `make run` to start the container
- 通过 nsenter 进入容器查看 IP 配置
  - [Enter Docker Namespace and show network](#enter-docker-namespace)


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

# Kubernetes

[Kubernetes installation on GCP](k8s-install)

## Configure ConfigMap

Create configuration file configmaps
```
$ k create configmap httpserver-ini-prod --from-file=app.ini=prod_app.ini
$ k create configmap httpserver-ini-staging --from-file=app.ini=staging_app.ini
```

Create environment variable configmaps
```
$ k create -f prod_config.yaml
$ k create -f staging_config.yaml
```

## Create Deployment

Use different deployment YAML file to create different kind of deployment

To create production deployment
```
$ k create -f deploy_prod.yaml
```

## Create service
```
$ k create -f service.yaml
```

## Viewing pod logs

```
$ k logs -f httpserver-7455c7c995-dxd2s
2022/01/27 07:46:55 Got config path from env: /etc/httpserver/app.ini
2022/01/27 07:46:55 Loading configuration conent
2022/01/27 07:46:55 /etc/httpserver/app.ini = # Production app configuration file, should stores in configMap
welcome="Hello, Kubernetes"
port=80
2022/01/27 07:46:55 Starting http server
2022/01/27 07:46:55 Server started on :8000
2022/01/27 07:46:55 version=v1.0.2
2022/01/27 07:46:55 Creating startup ready flag /tmp/httpserver_ready
```
This means the server app started successfully, and it reads correct configmaps content.

## Probes

According to [official Kubernetes document regarding probes](https://kubernetes.io/docs/tasks/configure-pod-container/configure-liveness-readiness-startup-probes/), 
- liveness probe - know when to restart a container
- readiness probe - know when a container is ready to start accepting traffic
- startup probe - know when a container application has started

We defined all three types of probes in our deployment. 
- `liveness probe` uses HTTP GET to know if the service is functional or not
- `readiness probe` uses TCP socket to detect whether the service is listening on the specified port(s)
- `startup probe` uses a flag file to detect whether the service application is start correctly

Once our application starts and listening on the service port, we will create the start up flag file. It indicates the services started as expected.

Let's first kill the application process and see what happens

```
$ k get po
NAME                          READY   STATUS    RESTARTS   AGE
httpserver-7455c7c995-dxd2s   1/1     Running   0          3m8s
httpserver-7455c7c995-wsjpt   1/1     Running   0          3m8s
$ k exec httpserver-7455c7c995-dxd2s -- ps
PID   USER     TIME  COMMAND
    1 root      0:00 /app/httpserver
   23 root      0:00 ps
$ k exec httpserver-7455c7c995-dxd2s -- kill -15 1
```

Meanwhile, look at the application log
```
$ k logs -f httpserver-7455c7c995-dxd2s
2022/01/27 07:50:20 Got signal terminated
2022/01/27 07:50:20 Server properly stopped
2022/01/27 07:50:20 Running clean up...
2022/01/27 07:50:20 Deleting start flag file /tmp/httpserver_ready
```

Let's watch the pod `httpserver-7455c7c995-dxd2s` manually marked by `#<<<<<<<<>>>>>>>>`

```
$ k get po -w
NAME                          READY   STATUS    RESTARTS      AGE
httpserver-7455c7c995-dxd2s   0/1     Running   1 (14s ago)   3m54s #<<<<<<<<>>>>>>>>
httpserver-7455c7c995-wsjpt   1/1     Running   0             3m54s
httpserver-7455c7c995-dxd2s   0/1     Running   1 (15s ago)   3m55s #<<<<<<<<>>>>>>>>
httpserver-7455c7c995-dxd2s   1/1     Running   1 (15s ago)   3m55s #<<<<<<<<>>>>>>>>
^C
$ k get po
NAME                          READY   STATUS    RESTARTS      AGE
httpserver-7455c7c995-dxd2s   1/1     Running   1 (38s ago)   4m18s #<<<<<<<<>>>>>>>>
httpserver-7455c7c995-wsjpt   1/1     Running   0             4m18s
```
