## Setup GCP VM Instance

注册 Google Cloud Platform试用账号

接下来新建一个 VPC，在选择的区域创建网络即可，用于连接虚拟机第二个网络接口。

然后开始创建虚拟机实例，配置时，注意选择

- 添加第二个网络接口，选择新建的 VPC 网络接口
- SSH 密钥填写自己的公钥，或者新生成一个公钥

其他选项较为通用，不用详细说。

# 配置环境

## 安装 Go 1.17

访问官网，得到下载包地址，执行命令下载安装包到`/usr/local`，并且解压到`/usr/local`

```sh 
$ sudo wget https://go.dev/dl/go1.17.6.linux-amd64.tar.gz -O /usr/local/go1.17.6.linux-amd64.tar.gz
$ sudo tar zxvf  /usr/local/go1.17.6.linux-amd64.tar.gz -C /usr/local
```

将`go` 命令加入PATH中，修改文件`$HOME/.bashrc`，添加如下内容到最后一行

```
export PATH=$PATH:/usr/local/go/bin
```

应用修改：

```sh
$ source $HOME/.bashrc
```

验证配置是否成功：

```sh
$ go version
go version go1.17.6 linux/amd64
```

后续使用时，发现通过 sudo 执行 go 命令时，会提示找不到命令，原因是 sudo 的时候环境变量跟当前用户环境变量不同，无法找到 go 的可执行程序。一个简单的方法就是吧 go 链接到所有用户都能访问的地方 `sudo ln -s /usr/local/go/bin/go /usr/local/bin/`，或者修改`/etc/profile` ，使环境变量的配置对所有用户生效。

## 测试 HTTP 服务

要测试我们的 GCP instance 是否能够正常跑服务，需要做两件事，1，配置防火墙策略，允许外部网络访问 instance 的服务端口，在这里，需要允许8000端口。2. 运行测试程序，在 8000 端口上启动 http test service。

首先，GCP 上添加防火墙策略，指定 tcp port 8000，网络标记指定为 instance 的网络标记。比如，在启动 instance 时，可以为该 instance 指定网络标记为`cloud-native-instance`，在防火墙策略中，可以将目标标记设置为`cloud-native-instance`。这样能够限定该防火墙策略应用于指定 instance，避免策略允许的范围过大。

配置好策略后，在本地 clone 一个 http测试程序 https://github.com/Drinkey/go-httpserver-demo，然后使用 `go run httpserver.go`启动一个临时的 http service，程序会在`0.0.0.0:8000`端口服务，通过公网 IP 访问该服务

```sh
curl http://<your_public_ip>:8000
welcome
```

如果显示正常，说明网络层面和我们的 instance 都在正常工作。

## 安装 Docker

按照 Docker 官网教程安装

https://docs.docker.com/engine/install/ubuntu/

安装完成后，我们在本地 80 端口通过 docker 启动上一步运行的 go http server 程序

```sh
$ sudo docker run -d -p 80:8000 --name httpserver drinkey/httpserver:latest
$ sudo docker ps
CONTAINER ID   IMAGE                       COMMAND             CREATED         STATUS         PORTS                                   NAMES
327e8b3779c1   drinkey/httpserver:latest   "/app/httpserver"   3 minutes ago   Up 3 minutes   0.0.0.0:80->8000/tcp, :::80->8000/tcp   httpserver
```

通过本地访问：

```sh
$ curl localhost/healthz
ok
```

通过外网访问：

```sh
$ curl http://<public_ip_address>/healthz
ok
```

如果都没有问题，说明防火墙策略配置正确，Docker 安装正确。可以开始 Kubernetes 的安装



## 安装 Kubernetes Cluster

安装之前，可以给机器做一个镜像，以免安装 K8s出现异常，导致需要重新配置。GCP VM instance 会默认关闭了 swap 分区，所以不要对 swap 分区做额外调整。

首先，确认内核模块`br_netfilter`被正确加载，如果没有，可以通过 `sudo modprobe br_netfilter` 命令来明确加载该内核模块

```sh
cloud-native-instance-1:~$ lsmod | grep br_netfilter
br_netfilter           28672  0
bridge                266240  1 br_netfilter
cloud-native-instance-1:~$
```

接下来，sysctl中配置`net.bridge.bridge-nf-call-iptables `设置成 1

```bash
cat <<EOF | sudo tee /etc/modules-load.d/k8s.conf
br_netfilter
EOF

cat <<EOF | sudo tee /etc/sysctl.d/k8s.conf
net.bridge.bridge-nf-call-ip6tables = 1
net.bridge.bridge-nf-call-iptables = 1
EOF
sudo sysctl --system
```

安装依赖包

```shell
sudo apt-get update
sudo apt-get install -y apt-transport-https ca-certificates curl
```

下载 Google 的 public signing key

```shell
sudo curl -fsSLo /usr/share/keyrings/kubernetes-archive-keyring.gpg https://packages.cloud.google.com/apt/doc/apt-key.gpg
```

添加 Kubernetes apt 源

```shell
echo "deb [signed-by=/usr/share/keyrings/kubernetes-archive-keyring.gpg] https://apt.kubernetes.io/ kubernetes-xenial main" | sudo tee /etc/apt/sources.list.d/kubernetes.list
```

安装 kubeadm，kubectl，kubelet，并且固定三个工具的版本。

```shell
sudo apt-get update
sudo apt-get install -y kubelet kubeadm kubectl
sudo apt-mark hold kubelet kubeadm kubectl
```

初始化 cluster，参考官方文档https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/

```sh
cloud-native-instance-1:~$ sudo kubeadm init --apiserver-advertise-address=192.168.123.2
[init] Using Kubernetes version: v1.23.1
[preflight] Running pre-flight checks
error execution phase preflight: [preflight] Some fatal errors occurred:
	[ERROR FileContent--proc-sys-net-ipv4-ip_forward]: /proc/sys/net/ipv4/ip_forward contents are not set to 1
[preflight] If you know what you are doing, you can make a check non-fatal with `--ignore-preflight-errors=...`
To see the stack trace of this error execute with --v=5 or higher
```

提示 ip_forward 没有开启，开启 ip_forward，直接修改会失败，使用 sysctl 来修改即可

```sh
cloud-native-instance-1:~$ sudo echo 1 > /proc/sys/net/ipv4/ip_forward
-bash: /proc/sys/net/ipv4/ip_forward: Permission denied
cloud-native-instance-1:~$ sudo sysctl -w net.ipv4.ip_forward=1
net.ipv4.ip_forward = 1
cloud-native-instance-1:~$ cat /proc/sys/net/ipv4/ip_forward
1
```

重新建立 cluster,指定 kubernetes 版本，指定 apiserver 地址为我们一开始配置的静态 IP 地址。

```sh
cloud-native-instance-1:~$ sudo kubeadm init --kubernetes-version v1.23.1 --apiserver-advertise-address=192.168.123.2
[init] Using Kubernetes version: v1.23.1
[preflight] Running pre-flight checks
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
```

执行顺利的话，会出现下面的

```sh
cloud-native-instance-1:~$ sudo kubeadm init --kubernetes-version v1.23.1 --apiserver-advertise-address=192.168.123.2
[init] Using Kubernetes version: v1.23.1
[preflight] Running pre-flight checks
[preflight] Pulling images required for setting up a Kubernetes cluster
[preflight] This might take a minute or two, depending on the speed of your internet connection
[preflight] You can also perform this action in beforehand using 'kubeadm config images pull'
[certs] Using certificateDir folder "/etc/kubernetes/pki"
[certs] Generating "ca" certificate and key
[certs] Generating "apiserver" certificate and key
[certs] apiserver serving cert is signed for DNS names [cloud-native-instance-1 kubernetes kubernetes.default kubernetes.default.svc kubernetes.default.svc.cluster.local] and IPs [10.96.0.1 192.168.123.2]
[certs] Generating "apiserver-kubelet-client" certificate and key
[certs] Generating "front-proxy-ca" certificate and key
[certs] Generating "front-proxy-client" certificate and key
[certs] Generating "etcd/ca" certificate and key
[certs] Generating "etcd/server" certificate and key
[certs] etcd/server serving cert is signed for DNS names [cloud-native-instance-1 localhost] and IPs [192.168.123.2 127.0.0.1 ::1]
[certs] Generating "etcd/peer" certificate and key
[certs] etcd/peer serving cert is signed for DNS names [cloud-native-instance-1 localhost] and IPs [192.168.123.2 127.0.0.1 ::1]
[certs] Generating "etcd/healthcheck-client" certificate and key
[certs] Generating "apiserver-etcd-client" certificate and key
[certs] Generating "sa" key and public key
[kubeconfig] Using kubeconfig folder "/etc/kubernetes"
[kubeconfig] Writing "admin.conf" kubeconfig file
[kubeconfig] Writing "kubelet.conf" kubeconfig file
[kubeconfig] Writing "controller-manager.conf" kubeconfig file
[kubeconfig] Writing "scheduler.conf" kubeconfig file
[kubelet-start] Writing kubelet environment file with flags to file "/var/lib/kubelet/kubeadm-flags.env"
[kubelet-start] Writing kubelet configuration to file "/var/lib/kubelet/config.yaml"
[kubelet-start] Starting the kubelet
[control-plane] Using manifest folder "/etc/kubernetes/manifests"
[control-plane] Creating static Pod manifest for "kube-apiserver"
[control-plane] Creating static Pod manifest for "kube-controller-manager"
[control-plane] Creating static Pod manifest for "kube-scheduler"
[etcd] Creating static Pod manifest for local etcd in "/etc/kubernetes/manifests"
[wait-control-plane] Waiting for the kubelet to boot up the control plane as static Pods from directory "/etc/kubernetes/manifests". This can take up to 4m0s
[apiclient] All control plane components are healthy after 10.502624 seconds
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config-1.23" in namespace kube-system with the configuration for the kubelets in the cluster
NOTE: The "kubelet-config-1.23" naming of the kubelet ConfigMap is deprecated. Once the UnversionedKubeletConfigMap feature gate graduates to Beta the default name will become just "kubelet-config". Kubeadm upgrade will handle this transition transparently.
[upload-certs] Skipping phase. Please see --upload-certs
[mark-control-plane] Marking the node cloud-native-instance-1 as control-plane by adding the labels: [node-role.kubernetes.io/master(deprecated) node-role.kubernetes.io/control-plane node.kubernetes.io/exclude-from-external-load-balancers]
[mark-control-plane] Marking the node cloud-native-instance-1 as control-plane by adding the taints [node-role.kubernetes.io/master:NoSchedule]
[bootstrap-token] Using token: hxn2vl.7h2ynx61zht0p4js
[bootstrap-token] Configuring bootstrap tokens, cluster-info ConfigMap, RBAC Roles
[bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to get nodes
[bootstrap-token] configured RBAC rules to allow Node Bootstrap tokens to post CSRs in order for nodes to get long term certificate credentials
[bootstrap-token] configured RBAC rules to allow the csrapprover controller automatically approve CSRs from a Node Bootstrap Token
[bootstrap-token] configured RBAC rules to allow certificate rotation for all node client certificates in the cluster
[bootstrap-token] Creating the "cluster-info" ConfigMap in the "kube-public" namespace
[kubelet-finalize] Updating "/etc/kubernetes/kubelet.conf" to point to a rotatable kubelet client certificate and key
[addons] Applied essential addon: CoreDNS
[addons] Applied essential addon: kube-proxy

Your Kubernetes control-plane has initialized successfully!

To start using your cluster, you need to run the following as a regular user:

  mkdir -p $HOME/.kube
  sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
  sudo chown $(id -u):$(id -g) $HOME/.kube/config

Alternatively, if you are the root user, you can run:

  export KUBECONFIG=/etc/kubernetes/admin.conf

You should now deploy a pod network to the cluster.
Run "kubectl apply -f [podnetwork].yaml" with one of the options listed at:
  https://kubernetes.io/docs/concepts/cluster-administration/addons/

Then you can join any number of worker nodes by running the following on each as root:

kubeadm join 192.168.123.2:6443 --token <token> \
	--discovery-token-ca-cert-hash sha256:<sha>
```

根据提示，将生成的配置文件写入相应路径，对于普通用户和 root 用户，有不同的操作指导。

默认情况下处于安全考虑， Cluster 不会将 pod 调度到 master 节点上，要建立起单节点的 Cluster，需要执行如下命令，允许 scheduler 调度 pod 到 master 节点上，这种模式一般应用于搭建开发环境。

如果你有更多的节点需要加入 cluster，那么在 worker 节点执行下面的命令

```sh
cloud-native-instance-1:~$ kubectl taint nodes --all node-role.kubernetes.io/master-
node/cloud-native-instance-1 untainted
```

# 探索

## namespace

## cgroups



