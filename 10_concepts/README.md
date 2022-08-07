# k8s Core Concept

## 简介

### 特性

k8s用于管理分布式、容器化应用，它提供了零停机时间部署、自动回滚、缩放和自愈等功能。K8s提供了一个抽象层，使其可以在物理或VM环境中部署容器应用，提供以容器为中心的基础架构。其设计理念是为了支撑**横向扩展**，即调整应用的副本数以提高可用性。k8s的具体特点如下：

- 环境无依赖：同一个应用支持公有云、私有云、混合云、多云部署。
- 面向切片：通过插件化，使所用功能都以插件部署形式动态加载，尤其针对复杂度较高的应用。
- 声明式：平台自身通过自动化方式达到预期状态。

## 架构

k8s遵从C/S架构，集群分为master和node 2部分，master作为服务端，node作为客户端。

<img src="figures/image-20220804190808652.png" alt="image-20220804190808652" style="zoom:50%;" />

### Master

Master可以实现高可用，默认情况下1个节点也能完成所有工作。它首先负责管理所有node，负责调度pod在哪些节点上运行，并且负责控制集群运行过程中的所有状态。所有控制命令都由master接收并处理，其核心组件包括：

- etcd：保存了整个集群的状态。
- [kube-apiserver](kube-apiserver/README.md)：集群的REST接口，是集群控制的入口。提供了资源操作的唯一入口，并提供认证、授权、访问控制、API 注册和发现等机制。kube-apiserver负责将k8s的GVR“资源组/资源版本/资源”以REST的形式对外暴露并提供服务。k8s集群中的所有组件都通过kube-apiserver操作资源对象。kube-apiserver也是集群中唯一与Etcd集群交互的核心组件，k8s将所有数据存储至Etcd集群中前缀为/registry的目录下。
- [kube-controller-manager](kube-controller-mgr/README.md)：集群所有资源对象的自动化控制中心，负责维护集群的状态，比如故障检测、自动扩展、滚动更新等。kube-controller-manager的目的是确保k8s的实际状态收敛到所需状态，它会及时发现并执行自动化修复流程，确保集群始终处于预期的工作状态。kube-controller-manager提供了一些默认的controller，每个controller通过kube-apiserver的接口实时监控整个集群的每个资源对象的状态。当发生各种故障而导致集群状态发生变化时，会尝试将系统状态恢复到期望状态。
- [kube-scheduler](kube-scheduler/README.md)：集群pod资源对象的调度服务，负责资源的调度，按照预定的调度策略将 Pod 调度到相应的机器上。kube-scheduler负责在k8s集群中为一个pod资源对象找到合适的节点并在该节点上运行。scheduler每次只调度一个pod。

### Node

Node是k8s集群的工作节点，负责管理本node上的所有容器，监控并上报所有pod的运行状态。node节点的工作由master进行分配，其核心组件包括：

- [kubelet](kubelet/README.md)：负责维持容器的生命周期，包括容器的创建、删除、启停等任务，与master进行通信。同时也负责 runtime（CRI）、Volume（CSI）和网络（CNI）的管理。kubelet用于管理节点，运行在每个k8s的node节点上。kubelet接收、处理、上报kube-apiserver下发的任务。kubelet启动时会先向kube-apiserver注册自身节点的信息。后续当kube-apiserver下发如创建pod等信息，kubelet负责在节点上的pod资源对象的管理，如pod资源对象的创建、修改、监控、删除、驱逐等。同时，kubelet会定期监控所在节点的资源使用情况并上报给kube-apiserver，这些数据可以帮助kube-scheduler为pod资源对象预选节点。kubelet也会对所在节点的容器和镜像做清理工作，保证节点上的镜像不会暂满磁盘空间、删除容器从而释放相关资源。
- Container runtime：它接收kubelet的指令，负责镜像管理以及 Pod 和容器的真正运行（CRI），默认的容器运行时为 Docker。
- [kube-proxy](kube-proxy/README.md)：负责k8s中服务的通讯及负载均衡，如为 Service 提供 cluster 内部的服务发现和负载均衡。kube-proxy作为node上的网络代理，它监控kube-apiserver的服务和端点资源变化，通过iptables/IPVS等配置负载均衡，为一组pod提供统一的流量转发和负载均衡功能。kube-proxy对某个IP:Port的请求，负责将其转发给专用网络上的相应服务。

### Add-ons组件

除了核心组件，还有一些推荐的 Add-ons：

- [kube-dns](kube-dns/README.md)：负责为整个集群提供 DNS 服务
- Ingress Controller：为服务提供外网入口
- Heapster：提供资源监控
- Dashboard：提供 GUI
- Federation：提供跨可用区的集群
- Fluentd-elasticsearch：提供集群日志采集、存储与查询

![image-20200806173918737](figures/image-20200806173918737.png)

### 客户端

- kubectl：kubectl是k8s的CLI，用户可以通过kubectl以命令交互的方式对kube-apiserver进行操作，通讯协议使用HTTP/JSON。kubectl发送相应的HTTP请求，请求由kube-apiserver接收、处理并将结果反馈给kubectl。kubectl接收到相应并展示结果。
- client-go：client-go是从k8s的代码中独立抽离出来的包，并作为官方提供的Go的SDK乏味作用。在大部分基于k8s做二次开发的程序中，建议通过client-go来实现与kube-apiserver的交互过程。因为client-go在k8s系统上做了大量优化，k8s的核心组件（如kube-scheduler、kube-controller-manager等）都通过client-go与kube-apiserver进行交互。

## 核心数据结构

### Group 资源组

k8s定义了许多group，这些group按不同的功能将resource进行划分，但也支持一个resource属于不同的group，例如`apis/apps/v1/deployments`。

有些资源是没有group的，被称为core group，例如`api/v1/pods`。

资源组的主要功能包括：

- 将资源划分group后，允许以group为单元进行启用/禁用。
- 每个group有自己的version，方便以group为单元进行迭代升级。

#### 数据结构

- Name：group的名字。
- Version：group下所支持的版本。
- PreferredVersion：推荐使用的version。

### Version 资源版本

每个group可以拥有不同的version，在YAML中的Version其实就是“group+version”。k8s的version分为了Alpha、Beta、Stable，依次逐步成熟，在默认情况下Alpha的功能在生产环境会被禁用。

#### 数据结构

- Versions：所支持的所有版本。

### Kind 资源种类

描述 resource的种类，与resource同一级别。k8s中的资源是某种kind的实例化。根据kind的不同，资源中具体字段也会有所不同，不过他们都用基本相同的结构。不同的kind被划分到不同的group中，并有着不同的version。

#### GVK（GroupVersionKind）



### Resource 资源

resource是k8s的核心概念，其整个体系都是围绕着resource构建的。k8s的本质就是resource的控制，包括注册、管理、调度并维护资源的状态。kind被实例化后会表现为一个resource object（entity）。目前k8s支持8种资源操作，分别是 create、delete、delectcollection、get、list、patch、update、watch。

#### External vs. Internal

在k8s中，用一个资源对应 External 和 Internal 2 个版本：

- External：对外暴露给用户所使用的resource object，其代码在`pkg/apis/group/version/`目录下。
- Internal：不对外暴露，仅在kube-apiserver内部使用。Internal 常用于资源版本的转换（不同的 External 资源版本通过 internal 进行中转），如将v1beta1转换为v1 的路径为 v1beta1 --> internal --> v1。其代码在 `pkg/apis/group/__internal/`目录下。

#### Schema

GVR（GroupVersionResource）：资源也有分组和版本号，具体表现形式为 `group/version/resource/subresource`，如deployments对应 /apis/apps/v1/namespaces/ns1/deployments。在k8s中，GVR被称为资源信息 schema。

#### Kind vs. Resource

Kind是对应了Go结构体名字，可以认为是一种类型。而resource是Kind的实例化。

#### 数据结构

- TypeMeat：
  - apiVersion：
  - kind：
- ObjectMeta：也就是YAML的metadata项
  - UID：
  - Name：
  - Namespace：
  - Labels：
- Spec：用户期望的状态
- Status：当前的状态

？？？

- Name：
- SingularName：resource的单数名称。
- Namespaced：是否有所属的namespace。
- Group：resource所在的group。
- Version：resource所在的version。
- Kind：resource的kind。
- Verbs：对该resource可操作的方法列表。
- ShortNames：resource的简称，如pod的简称为po。

### 映射关系

<img src="figures/image-20220725092000368.png" alt="image-20220725092000368" style="zoom:50%;" />

#### scheme 注册表

k8s有众多的资源，每一种资源就是一种资源类型，这些资源需要统一的注册、存储、查询和管理。scheme是k8s中的注册表，目前 k8s 中的所有资源类型都需要注册到 scheme 中，用于建立 **Golang 类型与 GVK 间的映射关系**。目前 k8s scheme 支持 UnversionedType 和 KnownType（也被直接称为 Type） 两种资源类型的注册。

scheme 资源注册表的数据结构主要由 4 个 map组成：

- gvkToType：
- typeToGVK：
- unversionedTypes：
- unversionedKinds：

在祖册资源类型时，会根据 Type 的类型同时添加到这 4 个 map 中。

#### RESTMapping

GVK与GVR之前的映射关系

### apimachinery

`k8s.io/apimachinery` 包含了用于实现类似 k8s API的通用代码，它并不仅限于容器管理，还可以用于任何业务领域的API接口开发。它包含了很多通用的API类型，如ObjectMeta、TypeMeta、GetOptions、ListOptions等。

#### runtime.Object

runtime.Object是k8s的通用资源类型，k8s上的所有resource object实际上都是Go的一个struct，它们都拥有runtime.Object。runtime.Object被设计为Interface，作为resource object通用部分。Go中k8s的对象都实现了 runtime.Object接口，包含 2 个方法：

- GetObjectKind()：返回GVK
- DeepCopyObject()：将数据结构克隆一份

##### Lab

- ？？？[runtime.object操作]()：实例化 pod 资源，再将 pod 资源转换为 runtime.object 资源，在将 runtime.object 资源转换回 pod 资源，最终通过 reflect 来验证转换是否等价。

#### interface.go

Serializer 包含序列化和反序列化操作。序列化将数据结构转换为字符串，而反序列化将字符串转换为数据结构，这样可以轻松地维护并存储、传输数据结构。Codec 包含编码器和解码器，它比 serializer 更为通用，指将一种数据结构转换为特定的格式的过程。所以，可以将 serializer 理解为一种特殊的 codec。

k8s 的 codec 包含 3 种 serializer：jsonSerializer、yamlSerializer、protobufSerializer。



## 代码

### Layout



### Option设置

- New Options：创建options
- Add Flags：将命令行flag添加到options结构体中
- Init logs：初始化日志
- Complete Options：填充默认参数到options
- Validate Options：验证options中所有参数

<img src="figures/image-20220804190826143.png" alt="image-20220804190826143" style="zoom:50%;" />

#### kube-apiserver Option示例

<img src="figures/image-20220804190837112.png" alt="image-20220804190837112" style="zoom:50%;" />

### ？？？代码生成器



## 构建

编译Go代码生成二进制文件

### 本地构建（推荐）

```shell
make all
```



### 容器环境构建



### Bazel环境构建

