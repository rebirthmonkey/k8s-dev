# cutom-aaserver

## 代码

整体代码通过 main.go 入口

### cmd

由 cmd 封装了一层 aaserver，它主要包括 apiserver（apis、registry）和 admission 2 部分。

- Options：Options 的核心是 genericoptions.RecommendedOptions，它用于提供运行apiserver 所需的“推荐”选项。推荐的选项取值可以由函数genericoptions.NewRecommendedOptions() 提供，也可以通过命令行参数获取选项取值。
- Options.Validate()：Validate方法调用RecommendedOptions进行选项（合并了用户提供的命令行参数）合法性校验。
- Options.Complete()：注册了一个Admission控制器。
- Options.Config()：将Options 转换为 Config。主要通过调用 ApplyTo 方法，将 RecommendedOptions 中的 Options 传递给了 RecommendedConfig。
- Options.RunCustomServer()：该方法启动 apiserver。它包含了从 Config 实例化 apiserver，并运行 apiserver 的整个流程。
- Server.PrepareRun()：安装诸如 Healthz 之类的接口。
- Server.Run()：通过 stopCh 启动 apiserver。

### apiserver

其主要目的是基于 config 实例化 apiserver。

- init()：将 k8s 的 core resource 的 GVK 安装到 scheme 里，从而实现 GVK 与 Go 类型之间的映射。
- Config
- Config.Complete()：填充 Config
- completedConfig.New()：创建了核心的 GenericAPIServer，并且将 APIGroupInfo 资源安装到 REST Registry 中，从而使 registry.REST 能够支持 API 对象的 CRUD 和 Watch。

### apis

#### restaurant

apis/restaurant 包以及它的子包，定义了 restaurant.wukong.com 组的 API。

- register.go：register.go 中定义了组，以及从 GV 获得 GVK、GVR 的函数。并且提供 AddToScheme 变量，用于将 API 注册到指定的 scheme。
- types.go：types.go 定义了 API 类型对应的 Go struct。子包 v1alpha1、v1beta1 定义了 API 的两个版本，它们包含和 restaurant 包类似的 GroupName、SchemeGroupVersion、SchemeBuilder、AddToScheme…等变量/函数，以及对应的 API 类型 struct。还包括自动生成的、与 APIVersionInternal 版本 API 进行转换的函数。
- install/install.go：Install 方法支持将 APIVersionInternal、v1alpha1、v1beta1 这些版本都注册到 scheme，会被 apiserver/init() 函数调用。

### registry

registry  的核心目的是为 APIGroupInfo 创建对应的 Go-REST 的 Web Service。registry.REST 是个空壳，实际是由 genericregistry.Store 负责的，为了创建 genericregistry.Store，需要两个信息：

1. Scheme：它提供了 GVK 与 Go 类型的映射关系，其中 Kind 是根据 Go 的类型名反射得到的。
2. generic.RESTOptionsGetter：用于获得 RESTOptions。

genericregistry.Store 还包含了三个 Strategy 字段：CreateStrategy、UpdateStrategy、DeleteStrategy。Strategy 能够影响增删改的行为，它能够生成对象名称、校验对象合法性，甚至修改对象。 

#### Go-REST处理流程

请求处理的整体逻辑：

1. GenericAPIServer.Handler 就是 http.Handler，可以注册给任何 HTTP 服务器，因此想绕开 HTTPS 的限制应该很容易。
2. GenericAPIServer.Handler 是一个层层包裹的处理器链，外层是一系列 filter，最里面是 director（HTTP handler）。
3. director 负责整体的请求分发：
   1. 对于 API 资源请求，如 /apis/restaurant.wukong.com/v1beta1，分发给 gorestfulContainer（在Go-REST 中 container 是一组 web service）。所有的 web service 都是由 APIGroupInfo 注册的。
   2. 对于非 API 资源请求：分发给 nonGoRestfulMux，最终由 PathRecorderMux 处理。可以利用这个扩展点，扩展任意形式的 HTTP 接口。
4. 在 GenericAPIServer.InstallAPIGroup 中，所有支持的 API 资源的所有版本，都注册为 go-restful 的一个Web Service。
5. 这些 Web Service 的逻辑包括（依赖于rest.Storage）：
   1. 将请求解码为资源对应的 Go 结构
   2. 将 Go 结构编码为JSON
   3. 将 JSON 存储到Etcd

#### Create

Create 请求被 genericregistry.Store 处理的过程：

1. 读取请求体，调用 NewFunc 反序列化 runtime.Obejct
2. 调用 PredicateFunc 判断是否能够处理该对象
3. 调用 CreateStrategy 校验、正规化对象
4. 调用 RESTOptions 存储到 Etcd

### admission

- Options.Config()：在之中注册 Admission Initializer，该 Initizlizer 会自动注册所有的 admission plugins
- Options.RunCustomServer()：在 PostStart 钩子中，会启动 Admission 所依赖的 SharedInformerFactory。
- custominitializer 包：是 Admission Initializer，它能够为任何 WantsInternalWardleInformerFactory 类型的 Admission 注入InformerFactory。
- Plugin/XXX：具体的 admission plugin 的实现，会在 Options.Complete() 函数中被注册。Admission plugin 的核心是 Admit() 函数，它可以修改或否决一个 API Server 的请求。

### 整体流程

APIServer 的核心类型是 GenericAPIServer，它是由 genericapiserver.CompletedConfig 的 New() 方法生成的。后者是 genericapiserver.RecommendedConfig 的 Complete() 方法生成的。而 RecommendedConfig 又是从 genericoptions.RecommendedOptions 得到的。apiserver 对Config、Option、Server 等对象都做了一层包装。

RecommendedOptions 对应了用户提供的各类选项，如 Etcd 地址、Etcd 存储前缀、APIServer 的基本信息等。调用 RecommendedOptions 的 ApplyTo 方法，会根据选项推导出 APIServer 所需的、完整的 Config。在这个方法中，甚至会进行自签名证书等重操作，而不是简单的将信息从 Options 复制给 Config。RecommendedOptions 会依次调用它的各个字段的 ApplyTo 方法，从而推导出 RecommendedConfig。

RecommendedConfig 的 Complete 方法，再一次进行配置信息的推导，主要牵涉到 OpenAPI 相关的配置。

CompletedConfig 的 New 方法实例化 GenericAPIServer，这一步最关键的逻辑是安装 APIGroup。APIGroup 定义了如何实现 GroupVersion 中 API 的增删改查，它将 GroupVersion 的每种资源映射到 registry.REST，后者具有处理 REST 风格请求的能力，并（默认）存储到 Etcd。

GenericAPIServer 同时提供了一些钩子来处理 Admission 的注册、初始化。以及另外一些钩子来对 API Server 的生命周期事件做出响应。

### artifacts

本示例通过一个 aa-server 来实现一个 Pizza 店的 API。该 API 提供 2 种 Kind：

- Topping：配料，包括 salami、mozzarella 或 tomato
- Pizza：提供 Pizza 类型，可以包含多种 Topping。

在实例中，会首先引入 v1alpha1 版本，然后在 v1beta1 中更换 topping 的表达方式。

## Lab

### 环境准备

#### 准备 k8s 集群

准备一个 k8s 集群，提供主 kube-apiserver

```shell
go install sigs.k8s.io/kind@v0.14.0
kind create cluster
kubectl config use-context kind-kind
```

#### 客户端访问凭证

```shell
cd configs/cert
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt # 可随意填写
openssl req -out client.csr -new -newkey rsa:4096 -nodes -keyout client.key -subj "/CN=development/O=system:masters"
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out client.crt
openssl pkcs12 -export -in ./client.crt -inkey ./client.key -out client.p12 # 密码设置为 P@ssw0rd
```

#### 代码更新

```shell
go mod tidy
go mod vendor
hack/update-codegen.sh
```

#### 启动 Etcd

```shell
etcd # 启动 Etcd 数据库
```

### 进程部署

#### 启动服务

通过进程启动 aa-server

```shell
GODEBUG=x509sha1=1 go run main.go --secure-port 8443 --etcd-servers http://127.0.0.1:2379   --kubeconfig ~/.kube/config --authentication-kubeconfig ~/.kube/config --authorization-kubeconfig ~/.kube/config --client-ca-file=configs/cert/ca.crt # Go 1.18 之后得注明 GODENBUG 参数
```

#### 测试

##### 直接调用aaserver

直接通过 URL 调用 aaserver，如果要用 kubectl，还需要配置 kind k8s 集群。

- List all API resources：

```shell
curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis
```

- List piaazs and toppings resources：

```shell
curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1alpha1/pizzas

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1alpha1/namespaces/default/pizzas

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1alpha1/toppings

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1alpha1/namespaces/default/topping # topping is not namespace

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1beta1/pizzas

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1beta1/namespaces/default/pizzas

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1beta1/toppings # topping is not registered

curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/restaurant.wukong.com/v1beta1/namespaces/default/toppings # topping is not registered
```

##### 通过kube-aggregator

- 创建 APIService

```shell
kubectl apply -f artifacts/example/ns.yaml
kubectl apply -f artifacts/example/apiservice.yaml
kubectl apply -f artifacts/example/service-ext.yaml
kubectl apply -f artifacts/example/endpoint.yaml
```

- 确认 aaserver 已注册资源

```shell
kubectl get apiservices.apiregistration.k8s.io | grep restaurant
kubectl -n restaurant get svc aaserver -o yaml  
kubectl -n restaurant get ep aaserver -o yaml 
```

- 创建 k8s 资源

```shell
kubectl apply -f artifacts/restaurant/topping-salami.yaml
kubectl apply -f artifacts/restaurant/topping-tomato.yaml
kubectl apply -f artifacts/restaurant/topping-mozzarella.yaml
kubectl get toppings
kubectl apply -f artifacts/restaurant/pizza-margherita.yaml
kubectl get pizzas
```

- 通过 get --raw 调用

```shell
kubectl get --raw "/apis/restaurant.wukong.com/v1alpha1/namespaces/default/pizzas"
```

#### cleanup

```shell
kubectl delete -f artifacts/restaurant
kubectl delete -f artifacts/example
```

### k8s 部署

#### 构建镜像

```shell
docker build -t wukongsun/custom-aaserver:0.1 .
kind load docker-image wukongsun/custom-aaserver:0.1 # load image to the kind cluster
docker exec kind-control-plane crictl images | grep wukongsun # 确认镜像已加载
```

#### 部署k8s资源

```shell
kubectl apply -f artifacts/example/ns.yaml
kubectl apply -f artifacts/example/sa.yaml
kubectl apply -f artifacts/example/rbac.yaml
kubectl apply -f artifacts/example/rbac-bind.yaml
kubectl apply -f artifacts/example/auth-delegator.yaml
kubectl apply -f artifacts/example/auth-reader.yaml
```

#### 启动服务

```shell
kubectl apply -f artifacts/example/deployment.yaml
kubectl -n wardle get pods
kubectl apply -f artifacts/example/service-k8s.yaml
```

#### 测试

- 注册 APIService

```shell
kubectl apply -f artifacts/example/apiservice.yaml
kubectl get apiservice v1alpha1.restaurant.wukong.com # 等待直到aaserver服务运行，即AVAILABLE为true
```


- 创建 k8s 资源
```shell
kubectl apply -f artifacts/restaurant/topping-salami.yaml
kubectl apply -f artifacts/restaurant/topping-tomato.yaml
kubectl apply -f artifacts/restaurant/topping-mozzarella.yaml
kubectl get toppings
kubectl apply -f artifacts/restaurant/pizza-margherita.yaml
kubectl get pizzas
```

- 通过 get --raw 调用

```shell
kubectl get --raw "/apis/restaurant.wukong.com/v1alpha1/namespaces/default/pizzas"
```

#### cleanup

```shell
kubectl delete -f artifacts/restaurant
kubectl delete -f artifacts/example
```

### 

