# custom-apiserver

## 简介

基于 aaserver，实现一个独立运行、使用 HTTP 协议的 custom-apiserver。

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





## HTTP Server

在启动服务器之前，可以直接访问 `GenericAPIServer.Handler.NonGoRestfulMux`，`NonGoRestfulMux` 实现了：

```go
type mux interface {
    Handle(pattern string, handler http.Handler)
}
```

调用 Handle 即可为任何路径注册处理器。



## Ref

1. [编写 Kubernetes 风格的 APIServer](https://blog.gmem.cc/kubernetes-style-apiserver)

