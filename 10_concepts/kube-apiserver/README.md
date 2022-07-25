# kube-apiserver

## 处理请求

### Authentication

- CA证书
- HTTP Token: 用token来表明user身份，`kube-apiserver`通过私钥来识别用户的合法性
- HTTP Base：把`UserName:Password`用BASE64编码后放入Authorization Header中发送给`kube-apiserver`

### Authorization

API server收到一个request后，会根据其中数据创建access policy object，然后将这个object与access policy逐条匹配，如果有至少一条匹配，则鉴权通过。

#### WebHook

k8s调用外部的access control service来进行用户授权。

#### ABAC

通过如subject user、group，resource/object apiGroup、namespace、resource等现有的attribute来鉴权。

#### RBAC

- Role：一个NS中一组permission/rule的集合
- ClusterRole：整个k8s集群的一组permission/rule的集合
- RoleBinding：把一个role绑定到一个user/group/serviceAccount，roleBinding也可使用clusterRole，把一个clusterRole运用在一个NS内。
- ClusterRoleBinding：把一个clusterRole绑定到一个user

### Admission Control

当任何一个API对象被提交给APIServer之后，总有一些“初始化”性质的工作需要在它们被k8s正式处理之前进行。比如，自动为所有Pod加上某些标签（Labels）。而这个“初始化”操作的实现，借助的是Admission Control功能。它其实是k8s里一组被称为Admission Controller的代码，可以选择性地被编译进APIServer中，在API对象创建之后会被立刻调用到。k8s提供了一种“热插拔”式的Admission机制，它就是Dynamic Admission Control，也叫作：Initializer。

 Initializer也是一个controller，实时查看用户给APIServer的请求，遇到实际状态与期望值不同时，更新用户API对象。更新用户的API对象的时候，使用PATCH API来完成merge工作。而这种PATCH API，正是声明式API最主要的能力。Initializer会再创建一个新的对象，然后通过TwoWayMergePatch和PATCH API把两个API对象merge，完成类似注入的操作。

发送个`kube-apiserver`的任何一个request都需要通过买个admission controller的检查，如果不通过则`kube-apiserver`拒绝此调用请求。



An admission controller is a piece of code that intercepts requests to the Kubernetes API server prior to persistence of the object, but after the request is authenticated and authorized. 

#### 类型

Admission controllers may be "validating", "mutating":

- Mutating controllers may modify related objects to the requests they admit; 
- validating controllers may not.

The admission control process proceeds in two phases. In the first phase, mutating admission controllers are run. In the second phase, validating admission controllers are run. 



### ServiceAccount

Service account是一种给pod里的进程而不是给用户的account，它为pod李的进程提供必要的身份证明。
Pod访问`kube-apiserver`时是以service方式访问kubernetes这个service。



## List-Watch

Etcd 存储集群的数据信息，而 Apiserver 作为统一入口，任何对数据的操作都必须经过 Apiserver。客户端（如kubelet、scheduler、controller-manager）通过 list-watch 监听Apiserver 中的资源（如 pod、rs、rc 等）的 create、update和 delete 事件，并针对事件类型调用相应的事件处理函数。

list-watch 有 list 和 watch 两部分组成。list 就是调用资源的list API 罗列所有资源，它基于 HTTP 短链接实现。watch 则是调用资源的 watch  API 监听资源变更事件，基于 HTTP 长链接实现。以 pod 资源为例，它的 list 和 watch API 分别为：

- List API：返回值为 PodList，即一组 pod

```http
GET /api/v1/pods
```

- Watch API：往往带上 watch=true，表示采用 HTTP 长连接持续监听 pod 相关事件。每当有新事件，返回一个 WatchEvent 。

```http
GET /api/v1/watch/pods
```

K8s 的 informer 模块封装了 list-watch API，用户只需要指定资源，编写事件处理函数 AddFunc、UpdateFunc 和DeleteFunc 等。如下图所示，informer 首先通过 list API 罗列资源，然后调用 watch  API 监听资源的变更事件，并将结果放入到一个 FIFO 队列，队列的另一头有协程从中取出事件，并调用对应的注册函数处理事件。Informer 还维护了一个只读的 Map Store 缓存，主要为了提升查询的效率，降低 Aiserver 的负载。

![理解K8S的设计精髓之list-watch](figures/f9eab21464ec485aab29fc83bbcddea9.png)

### Watch 的实现

Watch 是如何通过 HTTP 长链接接收 Apiserver 发来的资源变更事件呢？秘诀就是 Chunked Transfer Encoding（分块传输编码），它首次出现在HTTP/1.1 。

当客户端调用 watch API 时，Apiserver 在 response 的 HTTP  Header 中设置 Transfer-Encoding 的值为 chunked，表示采用分块传输编码。客户端收到该信息后，便和服务端该链接，并等待下一个数据块，即资源的事件信息。例如：

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

## K8S Proxy API

kube-apiserver把收到的REST request转发到某个node的kubelet的REST端口上，通过k8s proxy API获得的数据来自node而非etcd。

- Authentication：

  - 最严格的HTTPS证书认证，基于CA根证书签名的双向数字证书 认证方式
  - HTTP Token认证：通过一个Token来识别合法用户
  - Http Base认证：通过用户名+密码的方式认证
- Authorization：API Server授权，包括AlwayDeny、AlwaAllow、ABAC、RBAC、WebHook
- Admission Control：k8s AC体系中的最后一道关卡，官方标准的Adminssion Control就有10个，在启动kube-apiserver时指定

### 操作

- `kubectl proxy --port=8080`: create a local proxy for the local `kubelet` `API server`
- `curl 127.0.0.1:8080/api`
- `curl 127.0.0.1:8080/api/v1`
- `curl 127.0.0.1:8080/api/v1/pods`
