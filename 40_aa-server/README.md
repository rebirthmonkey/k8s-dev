# aa-server

## 简介

aa-server（aggregated-apiserver）的设计思路是允许用户编写一个自定义的 APIServer，在这里面添加自定义 API。这个 APIServer 就可以跟 k8s 原生的 kube-apiserver 绑定部署在一起统一提供服务了。同时，构建出的 API 接口更加规范整齐，能利用 k8s 原生的认证、授权、准入机制。

不同于 CRD，aa-server 中的 API 资源是通过代码向 k8s 注册资源类型的方式实现的，而 CRD 是直接通过 yaml 文件创建自定义资源的方式实现的。因此 aa-server 有着更大的自由度，没有太多限制。

<img src="figures/image-20220912172905515.png" alt="image-20220912172905515" style="zoom:50%;" />

### 使用场景

- 非 Etcd 存储
- 支持 protobuf，而非只有 JSON
- 需要扩展 /status 和 /scale 子资源，如 /logs、/port-forward 等
- 可以用 Go 高效实现所有操作，包括验证、准入和转换，尤其是支持大规模场景

## 架构

### 整体架构

aa-server 与 kube-apiserver 都是基于 `k8s.io/apiserver` 这个库来实现的。但最大的区别在于 aa-server 会在一个 k8s 集群内运行，有一个可用的 kube-apiserver 来代理或获取其他 k8s 资源。kube-aggregator 在 kube-apiserver 中用于用于代理、转发 aa-server 请求。它知道 自定义的 aa-server 以及它提供的 API 资源，以便正确地转发请求到对应的 aa-server 上。其处理流程包括：

- kube-apiserver 收到请求
- kube-apiserver 的 handling filter 进行处理，包括身份认证、日志审计、用户切换、限流和授权
- kube-aggregator 代理、转发 aa-server 提供的 API 服务给 aa-server

<img src="figures/image-20220912172937828.png" alt="image-20220912172937828" style="zoom:50%;" />

### 内部架构

aa-server 有着：

- 与 kube-apiserver 有着相同的结构
- 拥有自己的处理链：包括身份认证、日志审计、用户切换、限流和授权，但有些操作会委托给 kube-apiserver 来处理
- 拥有自己的 resource 处理链：包括解码、转换、准入、REST mapping 和编码
- 调用准入 webhook：
- 数据写入 Etcd：
- 拥有自己的 scheme 注册表：
- 委托 AuthN：通过 TokenAccessReview 委托 kube-apiserver 进行身份认证
- 审计：自己进行 audit
- 委托 AuthZ：通过 SubjectAccessReivew 委托 kube-apiserver 进行授权

<img src="figures/image-20220809165522802.png" alt="image-20220809165522802" style="zoom:50%;" />

#### Auth

由于 aa-server 位于 kube-apiserver（kube-aggregator）之后，所以请求到达时已经被 kube-apiserver 认证过了。kube-apiserver 会将认证的结果放在 HTTP header 里，通常是 X-Remote-User 和 X-Remote-Group 中。aa-server 通过客户端 CA 对这些 header 进行认证。

#### Authz

aa-server 通过 SubjectAccessReview 代理给 kube-apiserver。而 kube-apiserver 收到请求后，会基于集群的 RBAC 规则做出判断，返回一个 SubjectAccessReview 对象。

## Lab

### sample-apiserver

- [sample-apiserver](10_sample-apiserver/README.md)

### Pizza

本示例通过一个 aa-server 来实现一个 Pizza 店的 API。该 API 提供 2 种 Kind：

- Topping：配料，包括 salami、mozzarella 或 tomato
- Pizza：提供 Pizza 类型，可以包含多种 Topping。

在实例中，会首先引入 v1alpha1 版本，然后在 v1beta1 中更换 topping 的表达方式。

