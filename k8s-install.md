- [准备](#准备)
  - [Setup GCP VM Instance](#setup-gcp-vm-instance)
- [配置环境](#配置环境)
  - [安装 Go 1.17](#安装-go-117)
  - [测试 HTTP 服务](#测试-http-服务)
  - [安装 Docker](#安装-docker)
  - [安装 Kubernetes Cluster](#安装-kubernetes-cluster)
    - [安装前准备](#安装前准备)
    - [建立 cluster](#建立-cluster)
    - [安装网络](#安装网络)
    - [验证安装](#验证安装)
    - [Trouble shooting](#trouble-shooting)
- [探索](#探索)
  - [启动一个 pod，暴露服务，访问服务](#启动一个-pod暴露服务访问服务)
    - [启动一个 pod](#启动一个-pod)
    - [查看 pod 状态](#查看-pod-状态)
    - [查看pod详细信息](#查看pod详细信息)
    - [暴露服务](#暴露服务)
    - [查看服务状态](#查看服务状态)
    - [删除服务和 pod](#删除服务和-pod)

# 准备

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

### 安装前准备

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

开启 ip_forward，直接修改会失败，使用 sysctl 来修改即可

```sh
cloud-native-instance-1:~$ sudo echo 1 > /proc/sys/net/ipv4/ip_forward
-bash: /proc/sys/net/ipv4/ip_forward: Permission denied
cloud-native-instance-1:~$ sudo sysctl -w net.ipv4.ip_forward=1
net.ipv4.ip_forward = 1
cloud-native-instance-1:~$ cat /proc/sys/net/ipv4/ip_forward
1
```

### 建立 cluster

初始化 cluster，参考[官方文档]([Creating a cluster with kubeadm | Kubernetes](https://kubernetes.io/docs/setup/production-environment/tools/kubeadm/create-cluster-kubeadm/))

指定 kubernetes 版本，指定 apiserver 地址为我们一开始配置的静态 IP 地址，指定`--pod-network-cidr` 为`192.168.123.0/24`，这个参数必须要指定，否则安装 Calico 时会失败。

```sh

cloud-native-instance-1:~$ sudo kubeadm init \
                                --kubernetes-version v1.23.1
                                --apiserver-advertise-address=192.168.123.2
                                --pod-network-cidr 192.168.123.0/24
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
[apiclient] All control plane components are healthy after 8.003761 seconds
[upload-config] Storing the configuration used in ConfigMap "kubeadm-config" in the "kube-system" Namespace
[kubelet] Creating a ConfigMap "kubelet-config-1.23" in namespace kube-system with the configuration for the kubelets in the cluster
NOTE: The "kubelet-config-1.23" naming of the kubelet ConfigMap is deprecated. Once the UnversionedKubeletConfigMap feature gate graduates to Beta the default name will become just "kubelet-config". Kubeadm upgrade will handle this transition transparently.
[upload-certs] Skipping phase. Please see --upload-certs
[mark-control-plane] Marking the node cloud-native-instance-1 as control-plane by adding the labels: [node-role.kubernetes.io/master(deprecated) node-role.kubernetes.io/control-plane node.kubernetes.io/exclude-from-external-load-balancers]
[mark-control-plane] Marking the node cloud-native-instance-1 as control-plane by adding the taints [node-role.kubernetes.io/master:NoSchedule]
[bootstrap-token] Using token: txvn5f.k33wqdqwlcr84qd5
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

```shell
cloud-native-instance-1:~$ mkdir -p $HOME/.kube
cloud-native-instance-1:~$ sudo cp -i /etc/kubernetes/admin.conf $HOME/.kube/config
cp: overwrite '/home/user/.kube/config'? y
cloud-native-instance-1:~$ sudo chown $(id -u):$(id -g) $HOME/.kube/config
```

默认情况下处于安全考虑， Cluster 不会将 pod 调度到 master 节点上，要建立起单节点的 Cluster，需要执行如下命令，允许 scheduler 调度 pod 到 master 节点上，这种模式一般应用于搭建开发环境。

```sh
cloud-native-instance-1:~$ kubectl taint nodes --all node-role.kubernetes.io/master-
node/cloud-native-instance-1 untainted
```

如果你有更多的节点需要加入 cluster，那么在 worker 节点执行join 的命令，这这个例子中没有执行。

```shell
cloud-native-instance-1:~$ kubeadm join 192.168.123.2:6443 --token <token> \
	--discovery-token-ca-cert-hash sha256:<sha>
```



### 安装网络

首先安装 Calico 提供的 tigera operator，然后创建 custom resources，这里创建 custom resources 的时候，需要将文件里的`calicoNetwork.ipPools.cidr`的值修改成`192.168.123.0/24`，跟我们初始化 cluster 的时候设置的`--pod-network-cidr`保持一致。

```shell
cloud-native-instance-1:~$ kubectl create -f https://docs.projectcalico.org/manifests/tigera-operator.yaml
cloud-native-instance-1:~$ wget https://docs.projectcalico.org/manifests/custom-resources.yaml
cloud-native-instance-1:~$ vi custom-resources.yaml
cloud-native-instance-1:~$ kubectl create -f custom-resources.yaml

```

执行完成后提示成功，安装完成

### 验证安装

验证 core services 运行正常

```shell

cloud-native-instance-1:~$ k get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE                         ERROR
scheduler            Healthy   ok
controller-manager   Healthy   ok
etcd-0               Healthy   {"health":"true","reason":""}
```

验证 calico namespace 运行状态和其他 namespace

```shell
cloud-native-instance-1:~$  k get ns
NAME               STATUS   AGE
calico-apiserver   Active   2m20s
calico-system      Active   3m22s
default            Active   4m46s
kube-node-lease    Active   4m47s
kube-public        Active   4m48s
kube-system        Active   4m48s
tigera-operator    Active   3m35s证
```

验证节点状态是否 ready

```shell
cloud-native-instance-1:~$ k get node
NAME                      STATUS   ROLES                  AGE     VERSION
cloud-native-instance-1   Ready    control-plane,master   4m56s   v1.23.1
```

验证 kube-system 空间下的服务是否正确启动并运行

```shell
cloud-native-instance-1:~$ ks get pods
NAME                                              READY   STATUS    RESTARTS   AGE
coredns-64897985d-gxdtl                           1/1     Running   0          11m
coredns-64897985d-tkx28                           1/1     Running   0          11m
etcd-cloud-native-instance-1                      1/1     Running   9          11m
kube-apiserver-cloud-native-instance-1            1/1     Running   9          11m
kube-controller-manager-cloud-native-instance-1   1/1     Running   0          11m
kube-proxy-hvhzs                                  1/1     Running   0          11m
kube-scheduler-cloud-native-instance-1            1/1     Running   8          11m果
```

如果上述检查都通过，没有异常，则说明安装成功



### Trouble shooting

首先验证节点是否准备就绪

```shell
cloud-native-instance-1:~$ k get cs
Warning: v1 ComponentStatus is deprecated in v1.19+
NAME                 STATUS    MESSAGE                         ERROR
controller-manager   Healthy   ok
scheduler            Healthy   ok
etcd-0               Healthy   {"health":"true","reason":""}
cloud-native-instance-1:~$ k get no
NAME                      STATUS     ROLES                  AGE     VERSION
cloud-native-instance-1   NotReady   control-plane,master   5d22h   v1.23.1
```

发现节点处于 `NotReady` 状态

```shell

cloud-native-instance-1:~$ k get pods -n kube-system
NAME                                              READY   STATUS    RESTARTS      AGE
coredns-64897985d-4cjp9                           0/1     Pending   0             5d22h
coredns-64897985d-64nhf                           0/1     Pending   0             5d22h
etcd-cloud-native-instance-1                      1/1     Running   8 (64m ago)   5d22h
kube-apiserver-cloud-native-instance-1            1/1     Running   8 (63m ago)   5d22h
kube-controller-manager-cloud-native-instance-1   1/1     Running   8 (64m ago)   5d22h
kube-proxy-kfl8t                                  1/1     Running   7 (64m ago)   5d22h
kube-scheduler-cloud-native-instance-1            1/1     Running   7 (64m ago)   5d22h
jkzhang@cloud-native-instance-1:~$
```

发现 coredns 没有启动成功

```shell
cloud-native-instance-1:~$ ks describe node
Name:               cloud-native-instance-1
Roles:              control-plane,master
Labels:             beta.kubernetes.io/arch=amd64
                    beta.kubernetes.io/os=linux
                    kubernetes.io/arch=amd64
                    kubernetes.io/hostname=cloud-native-instance-1
                    kubernetes.io/os=linux
                    node-role.kubernetes.io/control-plane=
                    node-role.kubernetes.io/master=
                    node.kubernetes.io/exclude-from-external-load-balancers=
Annotations:        kubeadm.alpha.kubernetes.io/cri-socket: /var/run/dockershim.sock
                    node.alpha.kubernetes.io/ttl: 0
                    volumes.kubernetes.io/controller-managed-attach-detach: true
CreationTimestamp:  Wed, 12 Jan 2022 03:57:32 +0000
Taints:             node.kubernetes.io/not-ready:NoSchedule
Unschedulable:      false
Lease:
  HolderIdentity:  cloud-native-instance-1
  AcquireTime:     <unset>
  RenewTime:       Tue, 18 Jan 2022 02:57:56 +0000
Conditions:
  Type             Status  LastHeartbeatTime                 LastTransitionTime                Reason                       Message
  ----             ------  -----------------                 ------------------                ------                       -------
  MemoryPressure   False   Tue, 18 Jan 2022 02:57:46 +0000   Wed, 12 Jan 2022 03:57:30 +0000   KubeletHasSufficientMemory   kubelet has sufficient memory available
  DiskPressure     False   Tue, 18 Jan 2022 02:57:46 +0000   Wed, 12 Jan 2022 03:57:30 +0000   KubeletHasNoDiskPressure     kubelet has no disk pressure
  PIDPressure      False   Tue, 18 Jan 2022 02:57:46 +0000   Wed, 12 Jan 2022 03:57:30 +0000   KubeletHasSufficientPID      kubelet has sufficient PID available
  Ready            False   Tue, 18 Jan 2022 02:57:46 +0000   Wed, 12 Jan 2022 03:57:30 +0000   KubeletNotReady              container runtime network not ready: NetworkReady=false reason:NetworkPluginNotReady message:docker: network plugin is not ready: cni config uninitialized
Addresses:
  InternalIP:  192.168.123.2
  Hostname:    cloud-native-instance-1
Capacity:
  cpu:                4
  ephemeral-storage:  52174732Ki
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             16384456Ki
  pods:               110
Allocatable:
  cpu:                4
  ephemeral-storage:  48084232932
  hugepages-1Gi:      0
  hugepages-2Mi:      0
  memory:             16282056Ki
  pods:               110
System Info:
  Machine ID:                 ac437a51197e55c81cd67c9305274acb
  System UUID:                ac437a51-197e-55c8-1cd6-7c9305274acb
  Boot ID:                    9154b6e7-045c-4818-8756-b8211314f9bc
  Kernel Version:             5.13.0-1010-gcp
  OS Image:                   Ubuntu 21.10
  Operating System:           linux
  Architecture:               amd64
  Container Runtime Version:  docker://20.10.12
  Kubelet Version:            v1.23.1
  Kube-Proxy Version:         v1.23.1
Non-terminated Pods:          (6 in total)
  Namespace                   Name                                               CPU Requests  CPU Limits  Memory Requests  Memory Limits  Age
  ---------                   ----                                               ------------  ----------  ---------------  -------------  ---
  kube-system                 etcd-cloud-native-instance-1                       100m (2%)     0 (0%)      100Mi (0%)       0 (0%)         5d23h
  kube-system                 kube-apiserver-cloud-native-instance-1             250m (6%)     0 (0%)      0 (0%)           0 (0%)         5d23h
  kube-system                 kube-controller-manager-cloud-native-instance-1    200m (5%)     0 (0%)      0 (0%)           0 (0%)         5d23h
  kube-system                 kube-proxy-kfl8t                                   0 (0%)        0 (0%)      0 (0%)           0 (0%)         5d23h
  kube-system                 kube-scheduler-cloud-native-instance-1             100m (2%)     0 (0%)      0 (0%)           0 (0%)         5d23h
  tigera-operator             tigera-operator-768d489967-fhxzf                   0 (0%)        0 (0%)      0 (0%)           0 (0%)         110m
Allocated resources:
  (Total limits may be over 100 percent, i.e., overcommitted.)
  Resource           Requests    Limits
  --------           --------    ------
  cpu                650m (16%)  0 (0%)
  memory             100Mi (0%)  0 (0%)
  ephemeral-storage  0 (0%)      0 (0%)
  hugepages-1Gi      0 (0%)      0 (0%)
  hugepages-2Mi      0 (0%)      0 (0%)
Events:
  Type    Reason                   Age   From     Message
  ----    ------                   ----  ----     -------
  Normal  Starting                 51m   kubelet  Starting kubelet.
  Normal  NodeHasSufficientMemory  51m   kubelet  Node cloud-native-instance-1 status is now: NodeHasSufficientMemory
  Normal  NodeHasNoDiskPressure    51m   kubelet  Node cloud-native-instance-1 status is now: NodeHasNoDiskPressure
  Normal  NodeHasSufficientPID     51m   kubelet  Node cloud-native-instance-1 status is now: NodeHasSufficientPID
  Normal  NodeAllocatableEnforced  51m   kubelet  Updated Node Allocatable limit across pods
```

从 conditions 里面发现错误，说 `container runtime network not ready`，由此推断是网络相关问题，刚才做的网络相关的操作是安装 calico，所以回头看看相关文档。发现 calico custom resources 的 cidr配置需要跟 kubeadm init 的时候指定的`--pod-network-cidr`一致，但是在刚才运行 init 命令时，我没有指定这个参数。

解决方法是重新执行 init。首先执行 `kubeadm reset` 来恢复到 init 之前的状态，然后加上参数重新执行init即可。



# 探索

## 启动一个 pod，暴露服务，访问服务

### 启动一个 pod

```shell
cloud-native-instance-1:~$ k run --image=drinkey/httpserver http
pod/http created
```

### 查看 pod 状态

```shell
cloud-native-instance-1:~$ k get po
NAME   READY   STATUS    RESTARTS   AGE
http   1/1     Running   0          6s
```

### 查看pod详细信息

```sh
cloud-native-instance-1:~$ k describe po
Name:         http
Namespace:    default
Priority:     0
Node:         cloud-native-instance-1/192.168.123.2
Start Time:   Tue, 18 Jan 2022 06:53:18 +0000
Labels:       run=http
Annotations:  cni.projectcalico.org/containerID: b011da1b866b4fbb43b6266e3189a9d169c3583b5cc8e049815de1f94d4b5ae7
              cni.projectcalico.org/podIP: 192.168.123.203/32
              cni.projectcalico.org/podIPs: 192.168.123.203/32
Status:       Running
IP:           192.168.123.203
IPs:
  IP:  192.168.123.203
Containers:
  http:
    Container ID:   docker://9c0a3bd6273533a5948baaea728ea05c52ea0a8d055c2d3cf55942e0f2a276e1
    Image:          drinkey/httpserver
    Image ID:       docker-pullable://drinkey/httpserver@sha256:ad76459e7bae785b76623fceba93e3b5d0d6dbd880d9e100e08fce918fb56daa
    Port:           <none>
    Host Port:      <none>
    State:          Running
      Started:      Tue, 18 Jan 2022 06:53:21 +0000
    Ready:          True
    Restart Count:  0
    Environment:    <none>
    Mounts:
      /var/run/secrets/kubernetes.io/serviceaccount from kube-api-access-q84cp (ro)
Conditions:
  Type              Status
  Initialized       True
  Ready             True
  ContainersReady   True
  PodScheduled      True
Volumes:
  kube-api-access-q84cp:
    Type:                    Projected (a volume that contains injected data from multiple sources)
    TokenExpirationSeconds:  3607
    ConfigMapName:           kube-root-ca.crt
    ConfigMapOptional:       <nil>
    DownwardAPI:             true
QoS Class:                   BestEffort
Node-Selectors:              <none>
Tolerations:                 node.kubernetes.io/not-ready:NoExecute op=Exists for 300s
                             node.kubernetes.io/unreachable:NoExecute op=Exists for 300s
Events:
  Type    Reason     Age   From               Message
  ----    ------     ----  ----               -------
  Normal  Scheduled  13s   default-scheduler  Successfully assigned default/http to cloud-native-instance-1
  Normal  Pulling    12s   kubelet            Pulling image "drinkey/httpserver"
  Normal  Pulled     10s   kubelet            Successfully pulled image "drinkey/httpserver" in 1.994025865s
  Normal  Created    10s   kubelet            Created container http
  Normal  Started    10s   kubelet            Started container http

```

### 暴露服务

```shell
cloud-native-instance-1:~$ k expose pod http --selector run=http --port=80 --target-port=8000 --type=NodePort
service/http exposed
```

### 查看服务状态

```shell
cloud-native-instance-1:~$ k get svc
NAME         TYPE        CLUSTER-IP      EXTERNAL-IP   PORT(S)        AGE
http         NodePort    10.102.41.216   <none>        80:31945/TCP   4s
kubernetes   ClusterIP   10.96.0.1       <none>        443/TCP        4h5m
cloud-native-instance-1:~$ k describe svc http
Name:                     http
Namespace:                default
Labels:                   run=http
Annotations:              <none>
Selector:                 run=http
Type:                     NodePort
IP Family Policy:         SingleStack
IP Families:              IPv4
IP:                       10.102.41.216
IPs:                      10.102.41.216
Port:                     <unset>  80/TCP
TargetPort:               8000/TCP
NodePort:                 <unset>  31945/TCP
Endpoints:                192.168.123.203:8000
Session Affinity:         None
External Traffic Policy:  Cluster
Events:                   <none>
cloud-native-instance-1:~$
cloud-native-instance-1:~$ curl 10.102.41.216
welcome
```

### 删除服务和 pod

```shell
cloud-native-instance-1:~$ k delete svc http
service "http" deleted
cloud-native-instance-1:~$ k delete pod http
pod "http" deleted
cloud-native-instance-1:~$ k get pod
No resources found in default namespace.
cloud-native-instance-1:~$ k get svc
NAME         TYPE        CLUSTER-IP   EXTERNAL-IP   PORT(S)   AGE
kubernetes   ClusterIP   10.96.0.1    <none>        443/TCP   4h9m
cloud-native-instance-1:~$
```


