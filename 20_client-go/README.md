# client-go

通常不会直接向 kube-apiserver 发请求，而是通过 client-go 提供的编程接口。client-go 提供了缓存功能，避免反复从 kube-apiserver 获取数据。k8s 的主要 Go 编程接口赖在 `k8s.io/client-go` 这个库，它是一个典型的 web 客户端库，可以用于调用对应的 k8s 集群对应的 API，实现常用的 REST 动作。

## 客户端

client-go 支持 4 种客户端与 kube-apiserver 交互。

<img src="figures/image-20220904135525211.png" alt="image-20220904135525211" style="zoom:50%;" />

### kubeconfig

kubeconfig 用于管理访问 kube-apiserver 的配置信息，默认情况下放置在 `$HOME/.kube/config` 路径下，主要包括

- cluster：k8s 集群信息，如 kube-apiserver 地址、集群证书等。
- users：用户身份演进的凭据，如 client-certificate、client-key、token 及 username/password。
- contexts：集群用户信息及 namespace，用于将请求发送到指定的集群。

### RESTClient

RESTClient 是最基础的客户端，它对 HTTP Request 进行了封装，实现了 RESTful 风格的 API，后续的ClientSet、DynamicClient、DiscoveryClient 都是基于 RESTClient 的。

### ClientSet

ClientSet 在 RESTClient 的基础上封装了对 Resource 和 Version 的管理方法（不需要再在RESTClient 中配置 API、Group、Version 等与 resource 相关的信息）。每个 resource 可以理解为一个客户端，而 ClientSet 则是多个客户端的集合，可以让用户同时访问多个 resource。一般情况下，对 k8s 的二次开发使用 ClientSet，但 ClientSet 只能处理 k8s 的内置资源。

### DynamicClient

DynamicClient 与 ClientSet 最大的区别在于它能够处理 k8s 中的所有 resource，包括 CRD。DynamicClient 的处理过程将 Resource 转换成 unstructure 结构，再进行处理。处理完后，再将 unstructure 转换成 k8s 的结构体，整个过程类似于 Go 的 interface{} 断言转换过程。

DynamicClient 的输入和输出都是 `*unstructred.Unstructured` 对象，它的数据结构与 json.Unmarshall 的反序列化后的输出一样。

### DiscoveryClient

DiscoveryClient 用于发现 kube-apiserver 所支持的 group、version 和 resource。`kubectl api-versions` 与 `kubectl api-resources` 命令通过 DiscoveryClient 实现，但它也是基于 RESTClient 的基础上封装的。

## Informer

Informer 会观察 kube-apiserver 中的某一种资源，并在发现资源发生变化时出发某些动作。

一般使用的 Informer 为 SharedIndexInformer 类型，实现了sharedIndexInformer 共享机制。对于同一个资源，会存在多个 Listener 去监听它的变化，如果每一个 Listener 都来实例化一个对应的 Informer 实例，那么会存在非常多冗余的 List、watch 操作，导致 kube-apiserver 压力大。因此一个良好的设计思路为：Singleton 模式，同一类资源 Informer 共享一个 Reflector，这就是 K8s 中 SharedInformer 的机制。

![image-20220904145805905](figures/image-20220904145805905.png)

### Reflector

Reflector 负责监控（watch）对应的资源，其中包含 ListerWatcher、store(DeltaFIFO)、lastSyncResourceVersion、resyncPeriod 等信息。当资源发生变化时，会触发相应 resource object 的变更事件，并将该 resource object 及对其的操作类型（统称为 Delta）放入本地缓存 DeltaFIFO 中。

这是远端（kube-apiserver）和本地（DeltaFIFO、Indexer、Listener）之间数据同步逻辑的核心，是通过 ListAndWatch 方法来实现。

#### ListAndWatch

Reflector 主要就是 ListAndWatch 函数，负责获取资源列表（list）和监控（watch）指定的 k8s 资源。

Etcd 存储集群的数据信息，而 kube-apiserver 作为统一入口，任何对数据的操作都必须经过 kube-apiserver。客户端（如kubelet、scheduler、controller-manager）通过 list-watch 监听kube-apiserver 中的资源（如 pod、rs、rc 等）的 create、update和 delete 事件，并针对事件类型调用相应的 handler 事件处理函数。

list-watch 有 list 和 watch 两部分组成。list 就是调用资源的 list API 罗列所有资源，它基于 HTTP 短链接实现。watch 则是调用资源的 watch  API 监听资源变更事件，基于 HTTP 长链接实现。以 pod 资源为例，它的 list 和 watch API 分别为：

- List API：返回值为 PodList，即一组 pod

```http
GET /api/v1/pods
```

- Watch API：往往带上 watch=true，表示采用 HTTP 长连接持续监听 pod 相关事件。每当有新事件，返回一个 WatchEvent 。

```http
GET /api/v1/watch/pods
```

K8s 的 informer 模块封装了 list-watch API，用户只需要指定资源，编写事件处理函数 AddFunc、UpdateFunc 和 DeleteFunc 等。Informer 首先通过 list API 罗列资源，然后调用 watch  API 监听资源的变更事件，并将结果放入到一个 FIFO 队列。队列的另一头有 Processor 从中取出事件，并调用对应的注册 Handler 函数处理事件。Informer 还维护了一个只读的 Map Store 缓存，主要为了提升查询的效率，降低 kube-apiserver 的负载。
##### Watch实现

Watch 是如何通过 HTTP 长链接接收 kube-apiserver 发来的资源变更事件呢？秘诀就是 Chunked Transfer Encoding（分块传输编码），它首次出现在HTTP/1.1 。

当客户端调用 watch API 时，kube-apiserver 在 response 的 HTTP  Header 中设置 Transfer-Encoding 的值为 chunked，表示采用分块传输编码。客户端收到该信息后，便和服务端建立连接，并等待下一个数据块，即资源的事件信息。例如：

```shell
$ curl -i http://{kube-api-server-ip}:8080/api/v1/watch/pods?watch=yes

HTTP/1.1 200 OK
Content-Type: application/json
Transfer-Encoding: chunked
Date: Thu, 02 Jan 2019 20:22:59 GMT
Transfer-Encoding: chunked

{"type":"ADDED", "object":{"kind":"Pod","apiVersion":"v1",...}}
{"type":"ADDED", "object":{"kind":"Pod","apiVersion":"v1",...}}
{"type":"MODIFIED", "object":{"kind":"Pod","apiVersion":"v1",...}}
...
```

List-Watch 基于 HTTP 协议，是 K8s 重要的异步消息通知机制。它通过 list 获取全量数据，通过 watch  API 监听增量数据，保证消息可靠性、实时性、性能和顺序性。而消息的实时性、可靠性和顺序性又是实现声明式设计的良好前提。

当 Reflector 监听到 Added、Updated、Deleted 事件时，将会自动将对应的 resource object 更新到本地缓存 DeltaFIFO 中。

### DeltaFIFO

Delta：resource object 及对其的操作类型（如 Added、Updated、Deleted）。

ObjKey：基于 resource object，通过 DeltaFIFO.KeyOf() 函数计算资源对象的 UUID。

DeltaFIFO 是用于存储 Reflector 获得的待处理 resource object 及其操作类型的本地缓存。简单来说，它是一个生产者消费者队列，拥有 FIFO 的特性，操作的资源对象为 Delta。每一个 Delta 包含一个资源对象和其操作类型。

DeltaFIFO 由一个 FIFO 和 Delta 的 Map 组成，其中 map 会保存对 resource object 的操作类型。

![image-20220904141010968](figures/image-20220904141010968.png)
- 生产者：DeltaFIFO 的生产者是 Reflector 调用的 DeltaFIFO 的 Add 方法。
- 消费者：DeltaFIFO 的消费者是 Processor 调用的 DeltaFIFO 的 Pop 方法。

### Processor

Processor（HandlerDelta）是一个针对不同 resource 的 handler 回调函数的路由分发器。它消费 DeltaFIFO 中排队的 Delta，并通过 distribute() 函数将 Delta 分发至不同的 handler。通过 Informer 的`AddEventHandler()` 可以向 Processor 注册新的 Handler。 当 Processor/HandleDeltas 处理 DeltaFIFO 中的 Delta 时，会将这些更新事件派发给注册的 Handler。

Informer 可以非常方便的动态获取各种资源的实时变化，开发者只需要在对应的 Informer 的 Processor 中调用 `AddEventHandler` 添加相应的逻辑处理 `AddFunc`、 `DeleteFunc`、 `UpdateFun`，就可以处理该 resource 的`Added`、`Deleted`、`Updated`动态变化。这样，整个开发流程就变得非常简单，开发者只需要注重回调的逻辑处理，而不用关心具体事件的生成和派发。

在大部分 Controller 中，Handler 的操作逻辑包括：更新给 Indexer、将 resource object 推送到 WorkQueue，从而等待其他 worker 来做下一步处理。

## Controller

在 k8s 中，controller 实现了一个控制循环，它通过 kube-apiserver 观测集群中的共享状态，进行必要的变更，尝试把资源对应的当前状态期望的目标状态。controller 负责执行例行性任务来保证集群尽可能接近其期望状态。典型情况下控制器读取 .spec 字段，运行一些逻辑，然后修改 .status 字段。

controller 可以对 k8s 的核心资源（如 pod、deployment）等进场操作，但也可以观察并操作用户自定义资源。k8s 自身提供了大量的 controller，并由 controller manager 统一管理。

### Indexer

Indexer 可以理解为 Etcd 的本地缓存，它是 client-go 用来存储 resource object 并自带 index 的本地存储，提供数据存储和数据索引功能。DeltaFIFO 通过 Processor 消费出来的 resource object 会存储在 Indexer。 Indexer 与 Etcd 中的数据保持完全一致，这样 client-go 可以很方便的从 Indexer 中读取相应 resource object 数据，而无需从远程的 Etcd 中读取，以减轻 kube-apiserver 的压力。

其通过 DeltaFIFO 中最新的 Delta 不停的更新自身信息，同时需要在本地（DeltaFIFO、Indexer、Listener）之间执行同步，以上两个更新和同步的步骤都由 Reflector 的 ListAndWatch 来触发。同时在本地 crash，需要进行 replace 时，也需要查看到 Indexer 中当前存储的所有 key。

ThreadSafeMap 是一个在内存中实现并发安全的 map，在每次增删改查操作时都会加锁，以保证数据的一致性。Indexer 在它之上做了封装，在每次增删改查 ThreadSafeMap 数据时，都会自动更新索引。

### WorkQueue

Processor 中注册的 Handler 通过回调函数接收到对应的 event 之后，需要将对应的 ObjKey 放入 WorkQueue 中，从而方便多个 worker 去消费。WorkQueue 内部主要有 queue、dirty、processing 三个结构，其中 queue 为 slice 类型保证了有序性， dirty 与 processing 为 hashmap，提供去重属性。使用 workqueue 的优势包括：

- 并发：支持多生产者、多消费者
- 去重：由dirty保证一段时间内的一个元素只会被处理一次
- 有序：FIFO特性，保证处理顺序，由queue来提供
- 标记：标示以恶搞元素是否正在被处理，由processing来提供
- 延迟：支持延迟队列，延迟将元素放入队列
- 限速：支持限速队列，对放入的元素进行速率限制
- 通知：ShutDown告知该workqueue不再接收新的元素

#### 延迟队列

基于 WorkQueue 增加了 AddAfter 方法，用于延迟一段时间后再将元素插入 WorkQueue 队列中。

#### 限速队列

限速队列利用延迟队列的特性，延迟某个元素的插入时间，从而达到限速的目的。它提供 4 种限速接口算法 RateLimiter：

- 令牌桶算法：
- 排队指数算法：
- 计数器算法：
- 混合模式：

### 控制循环

- 读取资源的状态：通常采用事件驱动模式
- 改变资源的状态：
- 通过 kube-apiserver 更新资源的状态：
- 循环执行以上3步：

### 自定义代码

总体来说，需要自定义的代码只有：

1. 调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`
2. 实现 worker 逻辑从 workqueue 中消费 ObjKey 即可。

## Lab

### 客户端

- 启动一个 pod：`kubectl run test --image=nginx --image-pull-policy=IfNotPresent`

- [restClient](02_restclient.go)：首先通过 kubeconfig 配置信息实例化 RESTClient 对象，并通过各种方法不全参数。通过 Do() 函数执行请求，Into() 函数将请求结果解析到对应类型中。
- [clientSet](05_clientset.go): get the test pod info
- [clientSet](07_clientset.go): get the running pod number in all the namespaces
- [clientSet](09_clientset.go): get various info about pods and nodes
- [dynamicClient](11_dynamicclient.go)：
- [discoveryClient](13_discoveryclient.go)：

### Informer

- [Informer](21_informer.go)：首先建立 clientSet 与 kube-apiserver 进行交互。实例化 SharedInformer 对象，并通过此获得 Pod 资源的 Informer 对象。为 Pod Informer 添加 Pod 资源的回调方法 AddFunc、UpdateFunc、DeleteFunc。在正常情况下，回调方法会将 resource object 推送到 WorkQueue 中，但在本示例中，为了简便直接打印。
- [Informer with WorkQueue](26_informer-workqueue.go): 

### Controller

- [Ingress Manager Controller](80_app/README.md)
