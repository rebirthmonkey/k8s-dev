# kube-apiserver

## 简介

kube-apiserver：

- 将 k8s 的所有资源对象封装成 REST 风格的 API 接口进行管理
- 将集群的所有数据和状态存户的 Etcd 中
- 提供丰富的安全访问机制，包括认证、授权及准入控制（admission control）
- 提供了集群各组件间的通讯和交互功能

k8s 基于 go-restful 框架，因为其具有很好的可定制化性。

### 架构

k8s 内包含 3 个独立的 HTTP server，分别是：

- kube-aggregator：以下 3 个 server 的 proxy。而代理 API 请求到 apiextension-server 的过程被称为 API aggregation。
- apiextension-server：提供了 CRD 的自定义资源服务，并通过apiextensionserver.Scheme 注册管理 CRD 相关资源。
- kube-apiserver：k8s 内置核心资源服务，并通过 legacyscheme.Scheme 注册管理资源。
- aa-server（APIAggregator）：通过 AA 对 k8s 进行扩展，并通过 aggregatorscheme.Scheme 注册管理相关资源

其中 apiextensionserver 和 aa-server 都可以在不修改 k8s 核心代码的前提下扩展 k8s 的资源。但这 3 个 server 都是基于 GenericAPIServer实现。

<img src="figures/image-20220807184929435.png" alt="image-20220807184929435" style="zoom:50%;" />

### 启动流程

- 注册资源：将所支持的 resource 注册到 scheme 中。它是通过 pkg 的 init() 来实现的，调用了 k8s 各个资源下的 install 包。
- Cobra 命令行参数解析：通过 NewServerRunOptions() 初始化 options 结构，通过 Complete() 填充默认参数，通过 Validate() 验证参数的合法性，最后通过 Run() 将参数传入组件启动逻辑。
- 创建 kube-apiserver 通用配置：
  - genericConfig：
  - OpenAPI 配置：
  - Etcd 配置：
  - Authentication 配置：
  - Authorization 配置：
  - Admission 配置：
- 创建 apiextension-server：
  - 创建 genericapiserver：
- 创建 kube-apiserver：
  - 创建 genericapiserver：
  - 实例化 master：
  - 注册 /api 资源：
  - 注册 /apis 资源：
- 创建 aa-server：
  - 创建 genericapiserver：
  - 实例化 APIAggregator：
  - 实例化 APIGroupInfo：
  - 注册 APIGroup：
- 创建 genericapiserver：以上 3 种 server 都依赖于 GenericAPIServer，通过它将 k8s 的资源与 REST API 进行映射
- 启动服务：
  - 启动 HTTP 服务：
  - 启动 HTTPS 服务：
- 权限控制：
  - AuthN：
  - AuthZ：
  - AdmissionControl：拦截 HTTP 请求，进行校验、修改或拒绝等操作。

### 处理流程

- 请求经过处理链处理：复用 kube-apiserver 的处理链，包括身份认证、日志审计、切换用户、限流、授权
- 拦截请求：由 kube-aggregator 对 `/apis/aggregated-API-group-name` 路径下的请求进行拦截
- 转发请求：由 kube-aggregator 将拦截的请求转发给 aa-server

## 处理请求

### Authentication

- CA证书
- HTTP Token: 用token来表明user身份，`kube-apiserver`通过私钥来识别用户的合法性
- HTTP Base：把`UserName:Password`用BASE64编码后放入Authorization Header中发送给`kube-apiserver`

### Authorization

API server收到一个request后，会根据其中数据创建access policy object，然后将这个object与access policy逐条匹配，如果有至少一条匹配，则鉴权通过。

#### ABAC

通过如subject user、group，resource/object apiGroup、namespace、resource等现有的attribute来鉴权。

#### RBAC

- Role：一个NS中一组permission/rule的集合
- ClusterRole：整个k8s集群的一组permission/rule的集合
- RoleBinding：把一个role绑定到一个user/group/serviceAccount，roleBinding也可使用clusterRole，把一个clusterRole运用在一个NS内。
- ClusterRoleBinding：把一个clusterRole绑定到一个user

#### WebHook

k8s调用外部的access control service来进行用户授权。

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

#### vs. Webhook

admission controller是一组标准的控制器，拦截API请求，进行请求验证/修改。admission webhook就是由这些控制器调用的，运行在k8s外部的http服务，用来实现修改、验证等逻辑。因为这部分check牵涉到“业务逻辑”，不适合编写在k8s里面，所以采用动态扩展、可拔插的模式。

### ServiceAccount

Service account是一种给pod里的进程而不是给用户的account，它为pod李的进程提供必要的身份证明。
Pod访问`kube-apiserver`时是以service方式访问kubernetes这个service。

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
