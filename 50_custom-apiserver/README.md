# custom-apiserver

## 简介

基于 [k8s apiserver](https://github.com/kubernetes/apiserver)，实现一个独立运行、使用 HTTP 协议的 custom-apiserver。



### 解除依赖

custom-apiserver 首先必须解除 aaserver 对 kube-apiserver 的依赖，主要包括：

- Authentication：依赖主 kube-apiserver，是因为它需要访问 TokenReviewInterface，访问 kube-system 中的 ConfigMap。
- Authorization：依赖主 kube-apiserver，是因为它需要访问 SubjectAccessReviewInterface。
- CoreAPI：直接为 Config 提供了两个字段 ClientConfig、SharedInformerFactory。
- Admission：空 CoreAPI 会导致报错 `admission depends on a Kubernetes core API shared  informer, it cannot be  nil`。这提示不能在不依赖主 kube-apiserver 的情况下使用 Admission 控制器这一特性，需要将 Admission 也置空。

```go
o.RecommendedOptions.Authentication = nil
o.RecommendedOptions.Authorization = nil
o.RecommendedOptions.CoreAPI = nil
o.RecommendedOptions.Admission = nil
```

清空上述 4 个字段后，aaserver 还会在 PostStart 钩子中崩溃：

```go
// panic，这个SharedInformerFactory是CoreAPI选项提供的
config.GenericConfig.SharedInformerFactory.Start(context.StopCh)
// 仅仅Admission控制器使用该InformerFactory
o.SharedInformerFactory.Start(context.StopCh)
```

由于注释中给出的原因，这个 PostStart 钩子已经没有意义，删除即可正常启动服务器。

将这些字段置空，可以解除对主 kube-apiserver 的依赖。这样启动 aaserver 时就不需要提供这三个命令行选项：

--kubeconfig=/home/xxx/.kube/config
--authentication-kubeconfig=/home/xxx/.kube/config
--authorization-kubeconfig=/home/xxx/.kube/config

### HTTP 代替 HTTPS

GenericAPIServer 的 Run 方法的默认实现，是调用 		s.SecureServingInfo.Serve，因而强制使用 HTTPS：

```go
stoppedCh, err = s.SecureServingInfo.Serve(s.Handler, s.ShutdownTimeout, internalStopCh)
```


只需要将 s.Handler 传递给自己的 http.Server 即可使用 HTTP。

### 添加 OpenAPI

## 架构

### controller-manager

济源 controller-runtime 重写一个 controller-manager



### HTTP Server

k8s 风格的 apiserver 本质上就是一个 http.Handler，它的核心类型是 APIServerHandler，该类型：

1. 将标准的面向资源的 RESTful API 委托给 go-rest 的 Container 处理
2. 将其它 HTTP API 委托给 NonGoRestfulMux 处理，这其中就包括 healthcheck 这样的 API

k8s 风格的 API 资源具有规范化、冗长的特点，修改资源时发送的信息很多，不利于使用。此外，也需要让 custom-apiserver 具有一般性的 HTTP Server 的能力。为此，可以在 NonGoRestfulMux 注册一个额外的 Path 前缀，如 /wukong，并通过 Echo、Gin 等框架管理此前缀下的 API endpoint。

要添加非标 endpoint（也被称为 apiexts），需要在 `pkg/apiexts/` 下创建对应的包，例如 Xxx 的包名是 xxx。此包名也作为其子资源 endpoint 的前缀，如 /wukong/xxx/。该包应该提供一个 Register() 函数，能够将路由注册给 Echo。

#### 启动 web server

在 `cmd/manager/main.go` 中创建、启动 apiextsserver

```go
apiextServer := apiextsserver.Server(apiextAddr, apiServer, token, cmgr.GetClient())
go func() {
  err = apiextServer.ListenAndServe()
  if err != nil {
	  panic(err)
  }
}()
```

#### 路由注册

在 `cmd/apiserver/main.go` 中的 `AdditionalHandlers` 中注册转发路由：

```go
AdditionalHandlers: map[string]func(clients client.Clients) http.Handler{
	fmt.Sprintf("%s/", teleportPrefix): func(clients client.Clients) http.Handler {
  return httputil.NewSingleHostReverseProxy(apiextURL)
  },
},
```

### kubectl 配置

自行开发的 custom-apiserver 默认支持 kubectl，kubectl 的配置文件在 `configs/kubeconfig` 中，采用 token 认证模式。

## 代码

### Registry

在安装 APIGroup 时，需要为每个 API 组的每个版本的每种资源指定存储后端：

```go
// 每个组
apiGroupInfo := genericapiserver.NewDefaultAPIGroupInfo(wardle.GroupName, Scheme, metav1.ParameterCodec, Codecs)
// 每个版本
v1alpha1storage := map[stcongring]rest.Storage{}
// 每种资源提供一个rest.Storage
v1alpha1storage["flunders"] = wardleregistry.RESTInPeace(flunderstorage.NewREST(Scheme, c.GenericConfig.RESTOptionsGetter))
 
// 安装APIGroup
s.GenericAPIServer.InstallAPIGroup(&apiGroupInfo)
```

默认情况下，使用的是 genericregistry.Store，它会对接到 Etcd。要实现自己的存储后端，实现相关接口即可。

#### FS YAML Registry

调用 NewStore 即可创建一个 rest.Storage，但它没有：

1. 发现正在删除中的资源，并在 CRUD 时作出适当响应
2. 进行资源合法性校验。genericregistry.Store 的做法是，调用 strategy 进行校验
3. 自动填充某些元数据字段，包括 creationTimestamp、selfLink 等

### 多版本转换





### 子资源



## Lab

- 启动 APIServer：

```shell
go run main.go \
--backend=etcd \
--etcd-servers=http://127.0.0.1:2379 \
--file-rootpath=/tmp/custom-apiserver \
--request-timeout=60m \
--token-auth-file=configs/token.csv \
--unrestricted-update=true \
--kubectl-disabled=false \
--kubectl-ephemeral-token=false \
--authn-allow-localhost=true
```



```shell
curl -k --cert /tmp/custom-apiserver/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1alpha1/pizzas

curl --insecure https://127.0.0.1:8443/apis
curl -XGET http://127.0.0.1:6080/apis
curl -XGET http://127.0.0.1:6080/apis/restaurant.wukong.com/v1alpha1/pizzas
```







## Ref

1. [编写 Kubernetes 风格的 APIServer](https://blog.gmem.cc/kubernetes-style-apiserver)

