# client-go

k8s的主要Go编程接口赖在 `k8s.io/client-go` 这个库，它是一个典型的web客户端库，可以用于调用对应的k8s集群对应的API，实现常用的 REST 动作。

## 基础概念

### API版本



### 词汇

- API组：
- version：在YAML中的apiVersion其实就是“API组+version”
- kind：k8s中的资源是某种kind的实例。根据kind的不同，资源中具体字段也会有所不同，不过他们都用基本相同的结构。不同的kind被划分到不同的API组中，并有着不同的版本号。
- GVK-GroupVersionKind：
- resource-资源：
- GVR-GroupVersionResource：资源也有分组和版本号，如deployments对应 /apis/apps/v1/namespaces/ns1/deployments

#### 映射关系

- scheme：Golang类型与GVK间的映射关系
- RESTMapping：GVK与GVR之前的映射关系

<img src="figures/image-20220725092000368.png" alt="image-20220725092000368" style="zoom:50%;" />

### 数据结构

Go中k8s的对象都实现了 runtime.Object接口，包含GetObjectKind() 和 DeepCopyObject() 两个方法。

- TypeMeat：
  - apiVersion：
  - kind：
- ObjectMeta：也就是metadata项
- Spec：用户期望的状态
- Status：当前的状态



## 通用库

### apimachinery

`k8s.io/apimachinery` 包含了用于实现类似 k8s API的通用代码，它并不仅限于容器管理，还可以用于任何业务领域的API接口开发。它包含了很多通用的API类型，如ObjectMeta、TypeMeta、GetOptions、ListOptions等。



## 客户端

### RestClient



### ClientSet

ClientSet可以让用户同时访问多个API组和资源



## informer

Informer会观察某一种资源，并在发现资源发生变化时出发某些动作。

### 架构

<img src="figures/image-20220723204802609.png" alt="image-20220723204802609" style="zoom:50%;" />

以上Informer为SharedIndexInformer类型，实现了sharedIndexInformer共享机制。

1. Reflector：这是远端（APiServer）和本地（DeltaFIFO、Indexer、Listener）之间数据同步逻辑的核心，通过ListAndWatch方法来实现。Reflector负责监控对应的资源，其中包含ListerWatcher、store(DeltaFIFO)、lastSyncResourceVersion、resyncPeriod等信息， 当资源发生变化时，会触发相应obj的变更事件，并将该obj的delta放入DeltaFIFO中。
2. DeltaFIFO：存储Reflector获得的待处理obj(确切说是Delta)的地方，存储本地**最新的**数据，提供数据Add、Delete、Update方法，以及执行relist的Replace方法。简单来说，它是一个生产者消费者队列，拥有FIFO的特性，操作的资源对象为Delta。每一个Delta包含一个操作类型和操作对象。
3. Indexer(Local Store)：本地**最全的**数据存储，提供数据存储和数据索引功能。其通过DeltaFIFO中最新的Delta不停的更新自身信息，同时需要在本地(DeltaFIFO、Indexer、Listener)之间执行同步，以上两个更新和同步的步骤都由Reflector的ListAndWatch来触发。同时在本地crash，需要进行replace时，也需要查看到Indexer中当前存储的所有key。
4. shareProcessor（ResourceEventHandler）：消费DeltaFIFO中排队的Delta，同时更新给Indexer，并通过[distribute方法](https://km.woa.com/group/39344/articles/show/498233?kmref=dailymail_headline&jumpfrom=daily_mail#distribute方法)派发给对应的Listener集合。通过informer的`AddEventHandler`或`AddEventHandlerWithResyncPeriod`就可以向informer注册新的Listener，这些Listener共享同一个informer， 也就是说一个informer可以拥有多个Listener，是一对多的关系。 当HandleDeltas处理DeltaFIFO中的Delta时，会将这些更新事件派发给注册的Listener。
5. workqueue：回调函数处理得到的obj-key需要放入其中，待worker来消费，支持延迟、限速、去重、并发、标记、通知、有序。shareProcessor 的 Listener通过回调函数接收到对应的event之后，需要将对应的obj-key放入workqueue中，从而方便多个worker去消费。workqueue内部主要有queue、dirty、processing三个结构，其中queue为slice类型保证了有序性，dirty与processing为hashmap，提供去重属性。使用workqueue的优势：
   - 并发：支持多生产者、多消费者
   - 去重：由dirty保证一段时间内的一个元素只会被处理一次
   - 有序：FIFO特性，保证处理顺序，由queue来提供
   - 标记：标示以恶搞元素是否正在被处理，由processing来提供
   - 延迟：支持延迟队列，延迟将元素放入队列
   - 限速：支持限速队列，对放入的元素进行速率限制
   - 通知：ShutDown告知该workqueue不再接收新的元素

### sharedIndexInformer共享机制

对于同一个资源，会存在多个Listener去监听它的变化，如果每一个Listener都来实例化一个对应的Informer实例，那么会存在非常多冗余的List、watch操作，导致ApiServer的压力山大。因此一个良好的设计思路为：`Singleton模式`，一个资源只实例化一个Informer，后续所有的Listener都共享这一个Informer实例即可，这就是K8s中Informer的共享机制。



### 自定义代码

Informer可以非常方便的动态获取各种资源的实时变化，开发者只需要在对应的informer上调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`，就可以处理资源的`Added`、`Deleted`、`Updated`动态变化。这样，整个开发流程就变得非常简单，开发者只需要注重回调的逻辑处理，而不用关心具体事件的生成和派发。

总体来说，需要自定义的代码只有：

1. 调用`AddEventHandler`，添加相应的逻辑处理`AddFunc`、`DeleteFunc`、`UpdateFun`
2. 实现worker逻辑从workqueue中消费obj-key即可。



## Lab

- 启动一个 pod：`kubectl run test --image=nginx --image-pull-policy=IfNotPresent`

- [restClient](02_restclient.go): get the test pod info

- [clientSet](05_clientset.go): get the test pod info

- [clientSet](07_clientset.go): get the running pod number in all the namespaces

- [clientSet](09_clientset.go): get various info about pods and nodes

- [informer](15_informer.go): create a simple pod informer to watch pod resource evoluation

- [informer with WorkQueue](18_informer-workqueue.go): 

