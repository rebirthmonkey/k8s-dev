# Controller

## 简介

在 k8s 中，controller 用来实现真正的 k8s 业务逻辑。controller 可以对 k8s 的核心资源（如 pod、deployment）等进场操作，但也可以观察并操作用户自定义资源。controller 实现了一个控制循环，它通过 kube-apiserver 观测集群中的共享状态，进行必要的变更从而把资源对应的当前状态（status）向期望的目标状态（spec）转变。controller 负责执行例行性任务来保证集群尽可能接近其期望状态，典型情况下控制器读取 `.spec` 字段、执行一些逻辑、然后修改 `.status` 字段。k8s 自身提供了大量的 controller，并由 controller manager 统一管理。

### 原理

- 事件驱动：动作（create/update/delete）+资源的key（以namespace/name的形式表示）
- 申明式（通过 Reconcile Loop）：使 object 的 status 与 object 的 Spec 中定义的状态保持一致。其具体控制逻辑为：
  - 监控：1/ 监视 object 的当前 status 实际状态变化；2/ 接收 object 的新的创建/更新/删除事件。
  - 触发：对比 object 的当前 status 与期望 spec，判断是否触发后续的执行操作。因此只需要注入 EventHandler 将变化的事件放入 WorkQueue。
  - 执行：worker 执行能够调整 object 当前状态变化的操作，使之与期望状态匹配。
  - 更新：调用完成后，worker 会将 object 的状态更新为当前实际状态。
- aspect-oriented：用于解决业务的高复杂性，提升架构的可扩展性

### 架构

一般而言，controller 包含：
- Client：用于调用远程的 kube-apiserver。
- Informer（多个）：每个 Informer 监控一种资源，它从 DeltaFIFO 取出 API 对象，根据事件的类型来创建、更新或删除本地缓存。Informer 另一方面可以调用注册的 Event Handler 把 API 对象发送给对应的 controller。
  - Reflector：维护与 APIServer 的连接，使用 ListAndWatcher 方法来监听对象的变化，并把该变化事件及对应的 API 对象存入 DeltaFIFO 队列中。为了获得当前状态的详细信息，Reflector 会向 kube-apiserver 发送请求，并负责 watch 资源对象的状态变化，将相关 Event 发送到 WorkQueue 中。
  - DeltaFIFO：
  - HandlerDelta（EventHandler）：处理相关资源发生变化的 Event，将一个 API 对象的 key 存入 workQueue 中，这里存储的只是 API 对象的 key，value 会基于 key 去 Indexer 缓存中拉取。
- WorkQueue：存储事件的消息队列，用于异步接受 event。
- Indexer：用于本地缓存 Etcd 中资源的信息。它使用线程安全的数据存储来缓存 API 对象及其值，为 controller 提供数据索引功能。
- Worker：真正的业务处理逻辑，从 WorkQueue 中取出事件进行处理。它是个 Loop，会循环获取到 API 对象后则会根据 API 对象描述的期望状态与集群中的实际状态进行比对、协调，最终达到期望状态。

![image-20230304161258440](figures/image-20230304161258440.png)



<img src="figures/image-20230205131453908.png" alt="image-20230205131453908" style="zoom: 50%;" />

## Client

通常不会直接向 kube-apiserver 发请求，而是通过 client-go 提供的编程接口。k8s 的主要 Go 编程接口依赖 `k8s.io/client-go` 这个库。它是一个典型的 web 客户端库，可以用于调用对应的 k8s 集群对应的 API，实现常用的 REST 动作。client-go 负责与 kube-apiserver 通信，获取 API resource 的状态信息。同时，client-go 提供了缓存功能，避免反复从 kube-apiserver 获取数据。client-go 支持 4 种客户端与 kube-apiserver 交互。

<img src="figures/image-20220904135525211.png" alt="image-20220904135525211" style="zoom:50%;" />

### kubeconfig

kubeconfig 用于管理访问 kube-apiserver 的配置信息，默认情况下放置在 `$HOME/.kube/config` 路径下，主要包括

- cluster：k8s 集群信息，如 kube-apiserver 地址、集群证书等。
- users：用户身份演进的凭据，如 client-certificate、client-key、token 及 username/password。
- contexts：集群用户信息及 namespace，用于将请求发送到指定的集群。

### RESTClient

RESTClient 是最基础的客户端，它对 HTTP Request 进行了封装，实现了 RESTful 风格的 API，后续的 ClientSet、DynamicClient、DiscoveryClient 都是基于 RESTClient 实现的。

但 RESTClient 是最基本的，只能操作一种资源。

### ClientSet

ClientSet 在 RESTClient 的基础上封装了对 Resource 和 Version 的管理方法（不需要再在 RESTClient 中配置 API、Group、Version 等与 resource 相关的信息）。每个 resource 可以理解为一个客户端，而 ClientSet 则是多个客户端的集合，可以让用户同时访问多个 resource。一般情况下，对 k8s 的二次开发使用 ClientSet，但 ClientSet 只能处理 k8s 的内置资源。

### DynamicClient

DynamicClient 与 ClientSet 最大的区别在于它能够处理 k8s 中的所有 resource，包括 CRD。DynamicClient 的处理过程将 Resource 转换成 unstructure 结构，再进行处理。处理完后，再将 unstructure 转换成 k8s 的结构体，整个过程类似于 Go 的 interface{} 断言转换过程。

DynamicClient 的输入和输出都是 `*unstructred.Unstructured` 对象，它的数据结构与 json.Unmarshall 的反序列化后的输出一样。

### DiscoveryClient

DiscoveryClient 用于发现 kube-apiserver 所支持的 group、version 和 resource。`kubectl api-versions` 与 `kubectl api-resources` 命令通过 DiscoveryClient 实现，但它也是基于 RESTClient 的基础上封装的。

## Informer

Informer 是基于 client-go 实现的 k8s 客户端程序框架，它用于观察 kube-apiserver 中的某一种特定的资源，并在发现该资源发生变化时触发对应动作，它是对如 watch 等机制的可靠封装。

![image-20230304175657699](figures/image-20230304175657699.png)

### SharedIndexInformer

一般使用的 Informer 为 SharedIndexInformer 类型，实现了sharedIndexInformer 共享机制。对于同一个资源，会存在多个 Listener 去监听它的变化，如果每一个 Listener 都来实例化一个对应的 Informer 实例，那么会存在非常多冗余的 List、watch 操作，导致 kube-apiserver 压力大。因此一个良好的设计思路为，同一类资源 Informer 共享一个 Reflector，这就是 K8s 中 SharedInformer 的机制。在具体实现中，SharedInformer 中有个 map 数据结构，用于存放每个资源对应的 Informer。

启动流程：
- NewForConfig(config)：创建 clientset
  
- NewSharedInformerFactory()：创建共享 informer 的 factory
  
- informerFactory.Core().V1().Pods()：创建某个资源的 informer
  
- podInformer.Informer().AddEventHandler()：添加 handler
  
- informerFactory.Start()：启动 factory
  
- informerFactory.WaitForCacheSync()：等待缓存填充完毕
  

### Reflector

Reflector 基于 client-go 实现监控（ListAndWatch）对应的资源，其中包含 ListerWatcher、store(DeltaFIFO)、lastSyncResourceVersion、resyncPeriod 等信息。当资源发生变化时，会触发相应 resource object 的变更事件，并将该 resource object 及对其的操作类型（统称为 Delta）放入本地缓存 DeltaFIFO 中。

这是远端（kube-apiserver）和本地（DeltaFIFO、Indexer、Listener）之间数据同步逻辑的核心，是通过 ListAndWatch 方法来实现。

#### ListAndWatch

Reflector 主要就是 ListAndWatch() 函数，负责获取资源列表（list）和监控（watch）指定的 k8s 资源。

k8s 由 Etcd 存储集群的数据信息，而 kube-apiserver 作为统一入口，任何对数据的操作都必须经过 kube-apiserver。客户端（如 kubelet、scheduler、controller-manager）通过 list-watch 监听 kube-apiserver 中资源（如 pod、rs、rc 等）的 create、update 和 delete 等事件，并针对事件类型调用相应的 handler 事件处理函数。

list-watch 有 list 和 watch 两部分组成：list 就是调用资源的 list API 罗列所有资源，它基于 HTTP 短链接实现；而 watch 则是调用资源的 watch  API 监听资源变更事件，基于 HTTP 长链接实现。以 pod 资源为例，它的 list 和 watch API 分别为：

- List API：返回值为 PodList，即一组 pod

```http
GET /api/v1/pods
```

- Watch API：往往带上 watch=true，表示采用 HTTP 长连接持续监听 pod 相关事件。每当有新事件，返回一个 WatchEvent 。

```http
GET /api/v1/watch/pods
```

K8s 的 informer 模块封装了 list-watch API，用户只需要指定资源，编写事件处理函数 AddFunc、UpdateFunc 和 DeleteFunc 等。Informer 首先通过 list API 罗列资源，然后调用 watch  API 监听资源的变更事件，并将结果放入到一个 FIFO 队列。队列的另一头有 HandlerDelta 从中取出事件，并调用对应的注册 Handler 函数处理事件。Informer 还维护了一个只读的 Map Store 缓存，主要为了提升查询的效率，降低 kube-apiserver 的负载。
##### Watch实现

Watch 的 Event 在 APIServer 与 controller 之间通过 HTTP 流的方式传送，监听资源对象的 Event 变化（包括 Added、Modified、Deleted、Bookmark），并进行相应处理。Watch 是如何通过 HTTP 长链接接收 kube-apiserver 发来的资源变更事件呢？秘诀就是 Chunked Transfer Encoding（分块传输编码），它首次出现在HTTP/1.1 。

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

ObjKey：基于 resource object，通过 DeltaFIFO.KeyOf() 函数计算资源对象的 UUID。

DeltaFIFO 是用于存储 Reflector 获得的待处理的 resource object event 及其操作类型的本地缓存。简单来说，它是一个生产者消费者队列，拥有 FIFO 的特性，操作的资源对象为 Delta，而每一个 Delta 是一个 resource object 以及对其的操作类型（如 Added、Updated、Deleted）。

DeltaFIFO 由一个 FIFO 和 Delta 的 Map 组成，其中 map 会保存对 resource object 的操作类型：
- 生产者：DeltaFIFO 的生产者是 Reflector 调用的 DeltaFIFO 的 Add 方法。
- 消费者：DeltaFIFO 的消费者是 HandlerDelta 调用的 DeltaFIFO 的 Pop 方法。

<img src="figures/image-20220904141010968.png" alt="image-20220904141010968" style="zoom:50%;" />

存储类型：

- interface：tools/cache/store.go，其实 store 就是个 map
- cache：default的实现
- UndeltaStore：实现 Store 接口，但数据存放在 cache 中，数据变更时通过 PushFunc 发送所有状态。
- FIFO：实现 Queue 接口（包含 Store 接口）
- DeltaFIFO：对象与上次处理时发生了哪些变化，Delta 的类型包括 Added、Updated、Deleted、Replaced。

DeltaFIFO 生产的数据来自 List、Watch、Resync 等 Event，而消费的数据会发送到 WorkQueue 并缓存到 Indexer。

### HandlerDelta

HandlerDelta 是一个针对不同 resource 的 handler 回调函数的路由分发器。它消费 DeltaFIFO 中排队的 Delta，并通过 distribute() 函数将 Delta 分发至不同的 handler（回调函数）。通过 Informer 的`AddEventHandler()` 可以向 HandlerDelta 注册新的 handler 回调函数。 当 HandleDeltas 处理 DeltaFIFO 中的 Delta 时，会将这些更新事件派发给注册的 Handler。

Informer 可以非常方便的动态获取各种资源的实时变化，开发者只需要在对应的 Informer 的 HandlerDelta 中调用 `AddEventHandler` 添加相应的逻辑处理 `AddFunc`、 `DeleteFunc`、 `UpdateFun`，就可以处理该 resource 的`Added`、`Deleted`、`Updated`动态变化。这样，整个开发流程就变得非常简单，开发者只需要注重回调的逻辑处理，而不用关心具体事件的生成和派发。在大部分场景中，Handler 的操作逻辑包括：

- 更新本地缓存 Indexer：
- 将 resource object 推送到 WorkQueue：从而等待对应的 worker 来做下一步处理。

因此，对 Delta 真正的操作不在 HandlerDelta 中，它只是对 DeltaFIFO 的消费处理。真正的操作会由 Controller 中的 Worker 来实施。

## WorkQueue

HandlerDelta 中注册的 Handler 通过回调函数接收到对应的 event 之后，需要将对应的 ObjKey 放入 WorkQueue 中，从而方便并行的多个 worker 去消费，它用于保证：1/ Worker 处理速度低于 Event 产生速度；2/ 在必要时可以重试。

使用 WorkQueue 的优势包括：

- 并发：支持多生产者、多消费者
- 去重：由 dirty 保证一段时间内的一个元素只会被处理一次
- 有序：保证处理顺序，由 queue 来提供
- 标记：标示以恶搞元素是否正在被处理，由 processing来 提供
- 延迟：支持延迟队列，延迟将元素放入队列
- 限速：支持限速队列，对放入的元素进行速率限制
- 通知：ShutDown 告知该 workqueue 不再接收新的元素

WorkQueue 内部主要有 3个结构：

- queue：实际存储 item 的地方，为 slice 类型保证了有序性。
- dirty：确保去重，哪怕一个 item 被添加了多次，也只会被处理一次。
- processing：用于记录 item 是否正在被处理。

### 延迟队列

基于 WorkQueue 增加了 AddAfter() 方法，用于延迟一段时间后再将 item 插入 WorkQueue 队列中。AddAfter() 会插入一个 item，并附上一个 duration 延时。该 duration 会指定一个延时时间，如果 duration 小于等于 0，则直接会将 item 加入队列中。

### 限速队列

限速队列利用延迟队列的特性，延迟某个 item 的插入时间，从而达到限速的目的。它提供 4 种限速接口算法 RateLimiter：

- 令牌桶算法：是一个固定速率（qps）的限速器，该限速器是利用 `golang.org/x/time/rate` 库来实现。令牌桶算法内部实现了一个存放 token 的“桶”。初始时“桶”是空的，token 会以固定速率往“桶”里填充，直到将其填满为止，多余的 token 会被丢弃。每个 item 都会从令牌桶得到一个 token，只有得到 token 的 item 才允许通过，而没有得到 token 的 item 处于等待状态。令牌桶算法通过控制发放 token 来达到限速目的。
- 排队指数算法：
- 计数器算法：
- 混合模式：

## Indexer

Indexer 可以理解为 Etcd 在 Informer 的本地全量缓存，它是 Controller 用来存储 resource object 并自带 index 的本地存储，提供数据存储和数据索引功能。Controller 如果每次获取几个 object 就去访问 kube-apisever，会给 kube-apiserver 带来较大的压力。DeltaFIFO 通过 HandlerDelta 消费出来的 resource object 会存储在 Indexer。 Indexer 与 Etcd 中的数据保持完全一致，这样 Controller 可以很方便的从 Indexer 中读取相应 resource object 数据，而无需从远程的 Etcd 中读取，以减轻 kube-apiserver 的压力。

其通过 DeltaFIFO 中最新的 Delta 不停的更新自身信息，同时需要在本地（DeltaFIFO、Indexer、Listener）之间执行同步，以上两个更新和同步的步骤都由 Reflector 的 ListAndWatch 来触发。同时在本地 crash，需要进行 replace 时，也需要查看到 Indexer 中当前存储的所有 key。

### ThreadSafeMap

ThreadSafeMap 是一个在内存中实现并发安全的 map，在每次增、删、改、查操作时都会加锁，以保证数据的一致性。Informer 的 Indexer 就是在它之上做了封装，在每次增删改查 ThreadSafeMap 数据时，都会自动更新索引。

### Index 索引

Indexer 除了维护了一份本地内存缓存外，还有一个很重要的功能，便是索引功能了。索引的目的就是为了快速查找，比如需要查找某个 node 节点上的所有 pod、查找某个 namespace 下的所有 pod 等。利用到索引，可以实现快速查找。关于索引功能，则依赖于 threadSafeMap 结构体中的 indexers 与 indices 属性。

- Indexers：包含了所有索引器及其 IndexFunc，IndexFunc 为计算某个索引键下的所有对象键列表的方法。

```go
Indexers: {
  "namespace": MetaNamespaceIndexFunc,
  "nodeName": NodeNameIndexFunc,
}

func NodeNameIndexFunc(obj interface{})([]string, error){
  pod, ok := obj.(*v1.Pod)
  ...
  return []string{pod.Spec.NodeName}, nil
}
```

- Indices：包含了所有索引器及其所有的索引数据。

```go
Indices:{
  "namespace":{
    "default": ["pod1", "pod2"],
    "kube-system": ["pod3"],
  }
  "nodeName":{
    "node1": ["pod1"],
    "node2": ["pod2", "pod3"],
  }
}
```

- Index：包含了索引键以及索引键下的所有对象键的列表。

## Worker

controller 的 Run() 函数通过 runWorker() 函数持续不断地执行 processWorkItem() 函数，最终的业务逻辑会在 syncHandler() 函数中实现。

### Reconciler 控制循环

- 读取资源的状态：通常采用事件驱动模式
- 改变资源的状态：
- 通过 kube-apiserver 更新资源的状态：
- 循环执行以上3步：

### 自定义代码

总体来说，需要自定义的代码只有：

1. 添加 HandlerDelta 的回调函数：调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`。其本质是实现观察，将变化的事件放入 WorkQueue 中。
2. 实现 Worker 逻辑：从 WorkQueue 中消费 ObjKey 即可。其本质是分析、执行、更新。

## Lab

### Client

- 启动一个 pod：`kubectl run test --image=nginx --image-pull-policy=IfNotPresent`

- [restClient](02_restclient.go)：首先通过 kubeconfig 配置信息实例化 RESTClient 对象，并通过各种方法不全参数。通过 Do() 函数执行请求，Into() 函数将请求结果解析到对应类型中。
- [clientSet](05_clientset.go): get the test pod info
- [clientSet](07_clientset.go): get the running pod number in all the namespaces
- [clientSet](09_clientset.go): get various info about pods and nodes
- [dynamicClient](11_dynamicclient.go)：
- [discoveryClient](13_discoveryclient.go)：

### Informer

- [Informer](21_informer.go)：首先建立 clientSet 与 kube-apiserver 进行交互。实例化 SharedInformer 对象，并通过此获得 Pod 资源的 Informer 对象。为 Pod Informer 添加 Pod 资源的回调方法 AddFunc、UpdateFunc、DeleteFunc。在正常情况下，回调方法会将 resource object 推送到 WorkQueue 中，但在本示例中，为了简便直接打印。为了测试 Event Handler，可删除一个 pod，如 `kubectl delete pod test`。
- [Informer with WorkQueue](26_informer-workqueue.go)：把 Event 中获得的 Obj 添加到 WorkQueue 中。

### Controller

- [Ingress Manager Controller](80_ingress-mgr/README.md)

## Ref

- [图解 K8S 源码 - Deployment Controller 篇](https://developer.aliyun.com/article/774817)