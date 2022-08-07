# client-go

通常不会直接向 kube-apiserver 发请求，而是通过 client-go 提供的编程接口。client-go 提供了缓存功能，避免反复从 kube-apiserver 获取数据。k8s 的主要 Go 编程接口赖在 `k8s.io/client-go` 这个库，它是一个典型的web客户端库，可以用于调用对应的 k8s 集群对应的API，实现常用的 REST 动作。

## 客户端

### kubeconfig

kubeconfig 用于管理访问 kube-apiserver 的配置信息，默认情况下放置在 `$HOME/.kube/config` 路径下，主要包括

- cluster：k8s 集群信息，如 kube-apiserver 地址、集群证书等。
- users：用户身份演进的凭据，如 client-certificate、client-key、token 及 username/password。
- contexts：集群用户信息及 namespace，用于将请求发送到指定的集群。

### RESTClient

RESTClient 是最基础的客户端，它对 HTTP Request 进行了封装，实现了 RESTful 风格的 API。后续的ClientSet、DynamicClient、DiscoveryClient 都是基于 RESTClient 的。

#### Lab

- 启动一个 pod：`kubectl run test --image=nginx --image-pull-policy=IfNotPresent`

- [restClient](02_restclient.go)：首先通过 kubeconfig 配置信息实例化 RESTClient 对象，并通过各种方法不全参数。通过 Do() 函数执行请求，Into() 函数将请求结果解析到对应类型中。

### ClientSet

ClientSet 在 RESTClient 的基础上封装了对 Resource 和 Version 的管理方法（不需要再在RESTClient 中配置 API、Group、Version 等与 resource 相关的信息）。每个 resource 可以理解为一个客户端，而 ClientSet 则是多个客户端的集合，可以让用户同时访问多个 resource。印版情况下，对 k8s 的二次开发使用 ClientSet。

但 ClientSet 只能处理 k8s 的内置资源。

#### Lab

- [clientSet](05_clientset.go): get the test pod info

- [clientSet](07_clientset.go): get the running pod number in all the namespaces

- [clientSet](09_clientset.go): get various info about pods and nodes

### DynamicClient

DynamicClient 与 ClientSet 最大的区别在于它能够处理 k8s 中的所有 resource，包括 CRD。

DynamicClient 的处理过程将 Resource 转换成 unstructure 结构，再进行处理。处理完后，再将 unstructure 转换成 k8s 的结构体，整个过程类似于 Go 的 interface{} 断言转换过程。

### DiscoveryClient

DiscoveryClient 用于发现 kube-apiserver 所支持的 group、version 和 resource。

`kubectl api-versions` 与 `kubectl api-resources` 命令通过 DiscoveryClient 实现。但它也是基于 RESTClient 的基础上封装的。

## informer

Informer 会观察 kube-apiserver 中的某一种资源，并在发现资源发生变化时出发某些动作。

<img src="figures/image-20220723204802609.png" alt="image-20220723204802609" style="zoom:50%;" />

以上Informer为SharedIndexInformer类型，实现了sharedIndexInformer共享机制。

### Reflector

Reflector 负责监控（watch）对应的资源，其中包含ListerWatcher、store(DeltaFIFO)、lastSyncResourceVersion、resyncPeriod等信息。当资源发生变化时，会触发相应 resource（object）的变更事件，并将该 resource object 及对其的操作类型（统称为 Delta）放入本地缓存 DeltaFIFO 中。

这是远端（kube-apiserver）和本地（DeltaFIFO、Indexer、Listener）之间数据同步逻辑的核心，通过 ListAndWatch 方法来实现。

#### ListAndWatch

Reflector 主要就是 ListAndWatch 函数，负责获取资源列表（list）和监控（watch）制定的 k8s 资源。

Etcd 存储集群的数据信息，而 kube-apiserver 作为统一入口，任何对数据的操作都必须经过 kube-apiserver。客户端（如kubelet、scheduler、controller-manager）通过 list-watch 监听kube-apiserver 中的资源（如 pod、rs、rc 等）的 create、update和 delete 事件，并针对事件类型调用相应的事件处理函数。

list-watch 有 list 和 watch 两部分组成。list 就是调用资源的 list API 罗列所有资源，它基于 HTTP 短链接实现。watch 则是调用资源的 watch  API 监听资源变更事件，基于 HTTP 长链接实现。以 pod 资源为例，它的 list 和 watch API 分别为：

- List API：返回值为 PodList，即一组 pod

```http
GET /api/v1/pods
```

- Watch API：往往带上 watch=true，表示采用 HTTP 长连接持续监听 pod 相关事件。每当有新事件，返回一个 WatchEvent 。

```http
GET /api/v1/watch/pods
```

K8s 的 informer 模块封装了 list-watch API，用户只需要指定资源，编写事件处理函数 AddFunc、UpdateFunc 和DeleteFunc 等。如下图所示，informer 首先通过 list API 罗列资源，然后调用 watch  API 监听资源的变更事件，并将结果放入到一个 FIFO 队列，队列的另一头有协程从中取出事件，并调用对应的注册函数处理事件。Informer 还维护了一个只读的 Map Store 缓存，主要为了提升查询的效率，降低 kube-apiserver 的负载。

![理解K8S的设计精髓之list-watch](figures/f9eab21464ec485aab29fc83bbcddea9.png)

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

### DeltaFIFO

Delta：resource object 及对其的操作类型（如 Added、Updated、Deleted）。

ObjKey：基于 resource object 计算的资源对象的 UUID。

DeltaFIFO 是用于存储 Reflector 获得的待处理 resource object及其操作类型的本地缓存。简单来说，它是一个生产者消费者队列，拥有FIFO的特性，操作的资源对象为 Delta。每一个 Delta 包含一个资源对象和其操作类型。

DeltaFIFO 由一个 FIFO 和 Delta 的 Map 组成。

<img src="figures/image-20220807162248349.png" alt="image-20220807162248349" style="zoom:50%;" />

### Indexer

Indexer 是本地**最全的**数据存储，提供数据存储和数据索引功能。Indexer 与 ETCD 中的数据保持完全一致，这样 client-go 可以很方便的从 Indexer 中读取相应 resource object 数据，而无需从远程的 Etcd 中读取，以减轻 kube-apiser 的压力。

其通过 DeltaFIFO 中最新的Delta不停的更新自身信息，同时需要在本地（DeltaFIFO、Indexer、Listener）之间执行同步，以上两个更新和同步的步骤都由 Reflector 的 ListAndWatch 来触发。同时在本地 crach，需要进行 replace 时，也需要查看到 Indexer 中当前存储的所有key。

### Processor

Processor（ResourceEventHandler）消费 DeltaFIFO 中排队的Delta，同时更新给 Indexer，并通过 distribute 方法派发给对应的 Listener（也就是 EventHandler）集合。通过 Informer 的`AddEventHandler` 可以向Informer 注册新的 Listener，这些 Listener 共享同一个Informer。也就是说一个 Informer 可以拥有多个 Listener，是一对多的关系。 当 HandleDeltas 处理 DeltaFIFO 中的 Delta 时，会将这些更新事件派发给注册的 Listener。

### workqueue

回调函数处理得到的obj-key需要放入其中，待worker来消费，支持延迟、限速、去重、并发、标记、通知、有序。shareProcessor 的 Listener通过回调函数接收到对应的event之后，需要将对应的obj-key放入workqueue中，从而方便多个worker去消费。workqueue内部主要有queue、dirty、processing三个结构，其中queue为slice类型保证了有序性，dirty与processing为hashmap，提供去重属性。使用workqueue的优势：

- 并发：支持多生产者、多消费者
- 去重：由dirty保证一段时间内的一个元素只会被处理一次
- 有序：FIFO特性，保证处理顺序，由queue来提供
- 标记：标示以恶搞元素是否正在被处理，由processing来提供
- 延迟：支持延迟队列，延迟将元素放入队列
- 限速：支持限速队列，对放入的元素进行速率限制
- 通知：ShutDown告知该workqueue不再接收新的元素

### sharedInformer

对于同一个资源，会存在多个 Listener 去监听它的变化，如果每一个 Listener 都来实例化一个对应的 Informer 实例，那么会存在非常多冗余的 List、watch 操作，导致 kube-apiserver 压力山大。因此一个良好的设计思路为：Singleton 模式，同一类资源Informer 共享一个 Reflector，这就是 K8s 中 SharedInformer 的机制。

### 自定义代码

Informer可以非常方便的动态获取各种资源的实时变化，开发者只需要在对应的informer上调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`，就可以处理资源的`Added`、`Deleted`、`Updated`动态变化。这样，整个开发流程就变得非常简单，开发者只需要注重回调的逻辑处理，而不用关心具体事件的生成和派发。

总体来说，需要自定义的代码只有：

1. 调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`
2. 实现worker逻辑从workqueue中消费obj-key即可。



### Lab

- [Informer 不带 WorkQueue](15_informer.go)：首先建立 clientSet 与 kube-apiserver 进行交互。实例化 SharedInformer 对象，并通过此获得 Pod 资源的 Informer 对象。为 Pod Informer 添加 Pod 资源的回调方法 AddFunc、UpdateFunc、DeleteFunc。在正常情况下，回调方法会将 resource object 推送到 WorkQueue 中，但在本示例中，为了简便直接打印。

- [Informer with WorkQueue](18_informer-workqueue.go): 

