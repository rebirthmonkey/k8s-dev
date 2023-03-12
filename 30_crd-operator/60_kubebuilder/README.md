# kubebuilder

kubebuilder 为创建一个 Controller/Operator 搭建好了一套完整的代码框架，生成了一堆文件，涵盖了自定义 controller 的代码和一个示例 CRD。对于开发者来说，只需要实现`Reconcile()` 方法，即 `sample-controller`中的`syncHandler`，其他步骤`kubebuilder`已经帮着实现了。【1】

大部分 Controller 的开发会直接基于 kubebuilder，而不是使用到之前介绍的 Controller 模式。

## controller-runtime

controller-runtime 库包含若干 Go 库，用于快速构建（controller-manager、controller、dynamic clientset）。kubebuilder 依赖于 controller-runtime 库，使用 controller-runtime 的 Client 接口来实现针对 k8s 资源的 CRUD 操作。

### Manager

controller-runtime 由 Manager 封装起来（等价于 k8s 的 controller-manager），用于启动 controller（Manager.Start） ，并且管理被多个 controller 依赖的组件（其中 Scheme、Client 与 Cache 共同又被称作 Cluster）：

- Cache：Cache 实际是 Controller 中 Informer 的包装，为读客户端提供本地缓存，支持监听更新缓存的事件。如 DelegatingClient 从 Cache 中读取（Get/List），而写入请求（Create/Update/Delete）则直接发送给 kube-apiserver，随着缓存的更新，读操作会达成最终一致。使用 Cache 可以大大减轻 kube-apiserver 的压力。
  
- Client：Client 是对 Controller 中 client 的封装，用于实现针对 kube-apiserver 的 CRUD 操作，读写客户端通常是分离（split）的。manager.Manager 会创建 client.Client。
  
- Scheme：k8s GVK 的注册表。

- Controller：
  - Predicate：指明哪些 Event 会触发 Reconciler。
  - Reconciler：Controller 真正的业务逻辑代码。


<img src="figures/image-20220608172034690.png" alt="image-20220608172034690" style="zoom:50%;" />

启动流程：

- 创建 Manager：
  - 创建并注册 scheme
  - 创建 cluster（client+cache）
  - 为 runnable 创建 map
- 注册 Runnable：添加 runnable 到 map
- 启动 Manager：启动 map 中所有注册的 runnable

### Controller

Controller 会监控多种类型的 API resource（如 Pod + ReplicaSet + Deployment），但是 Controller 的 Reconciler 一般仅仅处理单一类型的对象。controller 从 Manager 得到各种共享对象，它自己创建一个工作队列，并从工作队列中获取 Event，转给 Reconciler。

当 A 类型的对象发生变化后，如果 B 类型的对象必须更新以响应，可以使用 EnqueueRequestFromMapFunc 来将一种类型的事件映射为另一种类型。如 Deployment 的 Controller 可以使用 EnqueueRequestForObject、EnqueueRequestForOwner 实现：

1. 监控 Deployment 事件，并将 Deployment 的 Namespace/Name 入队。
2. 监控 ReplicaSet 事件，并将创建它的 Deployment（Owner）的 Namespace/Name 入队。

类似 ReplicaSet 的控制器也可以监控 ReplicaSet 和 Pod 事件。

reconcile.Request 入队时会自动去重，也就是说一个 ReplicaSet 创建的多个 Pod 事件可能仅仅触发 ReplicaSet 控制器的单次调用。

#### Predicate

指明哪些 Event 会触发 Reconciler。

#### Reconciler

Reconciler 是 Controller 的核心逻辑所在，它负责调和使 status 逼近期望状态 spec。例如，当针对 ReplicaSet 对象调用 Reconciler 时，发现 ReplicaSet 要求 5 实例，但是当前只有 3 个 Pod。这时 Reconciler 应该创建额外的 2 个 Pod，并且将这些 Pod 的 OwnerReference（被管理的组件）指向当前的 ReplicaSet。Reconciler 通常仅处理一种类型的对象。

- Concurrence：具体启多少个 Reconciler，每个 Reconciler 每次只能处理一个 event。
- OwnerReference：用于从子对象（如 Pod）触发父对象的调和（如 ReplicaSet）操作。

##### 定义SetupWithManager()

在 controller.go 文件中定义 SetupWithManager(mgr ctrl.Manager) 方法将本 controller 注册给 Manager：

```go
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&ingressv1.App{}).
		Complete(r)
}
```

- NewControllerManagedBy()：基于现有的 Manager 创建一个空壳的 Controller。
- WithOptions()：
- For()：指明本 controller 操作哪类 Go Type struct。
- Complete()：为空壳 Controller 绑定 Reconciler。
- WithEventFilter()：在进入 controller 之前就过滤掉不符合条件（如已经标记为删除）的 Event 资源，则需要修改 SetupWithManager() 方法，增加 WithEventFilter 调用：

```go
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
  return ctrl.NewControllerManagedBy(mgr).
    WithOptions(controller.Options{
MaxConcurrentReconciles: r.Concurrence,}).
    For(&ingressv1.App{}).
    WithEventFilter(predicate.Funcs{
  // 分别对资源增加、删除、更新事件进行过滤
  CreateFunc: func(event event.CreateEvent) bool {
    return r.predicate(event.Object)
  },
  DeleteFunc: func(event event.DeleteEvent) bool {
    return r.predicate(event.Object)
  },
  UpdateFunc: func(event event.UpdateEvent) bool {
    return r.predicate(event.ObjectNew)
  },
  }).
    Complete(r)
}
```

##### 注册Controller/Reconciler

在 `main.go` 中实例化 Controller/Reconciler，并调用 SetupWithManager() 注册到 Manager 中：

```go
	if err = (&controllers.Reconciler{
		Client: mgr.GetClient(),
		Scheme: mgr.GetScheme(),
	}).SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "App")
		os.Exit(1)
	}
```

### Webhook



### scheme注册

Scheme 注册采用了 Builder 设计模式，用于在不同模块初始化时“主动”将自身信息注册到 scheme 中，从而实现每个模块的“热插拔”。其原理是构建一个回调函数列表，并在某一时刻统一执行。它先通过 Register() 注册一堆用于将 GVK-Type 添加到 scheme 中的回调函数，然后通过 AddToManager() 执行所有回调函数实现真正的 scheme 注册。

#### 数据结构

scheme 包含一组 schemeBuilder（每个 schemeBuilder 对应一个 GV），每个 Builder 就是一个回调函数列表。每个回调函数都是用于操作 scheme，其实就是调用 scheme.AddKnownTypes() 方法将自身 GV-Type 注册到 scheme 中。

#### 流程

- 创建 schemeBuilder：在 `groupversion_info.go` 文件中为某个 GroupVersion 创建一个 schemeBuilder。

```go
SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
```

- 将回调函数添加到 schemeBuilder 中：在 xxx_type.go 文件的 init() 函数中，通过 SchemeBuilder.Register() 函数，将对应的 Type（Go struct）注册到本 GV 的 schemeBuilder 中。

```go
func init() {
  SchemeBuilder.Register(&App{}, &AppList{})
}
// 其背后实际注册的函数是：
func(scheme *runtime.Scheme) error {		scheme.AddKnownTypes(builder.GroupVersion, &App{}, &AppList{})}
```

- 执行 schemeBuilder 内所有注册的回调函数：在 main.go 文件的 init() 函数中，通过对应 GV 的 schemeBuilder，所有注册的回调函数要延迟到此刻通过 AddToScheme() 才真正执行，GVK-Type 被真正添加到 scheme 中。

```go
utilruntime.Must(atv1.AddToScheme(scheme))
```

### Reconciler组装

如 Controller 中介绍，每个 ctrl.Manager 包含一个 Controller，而在 Controller 中包含了 Reconciler。在完成 Reconciler struct 的定义后，需要通过 Reconciler 的 SetupWithManager() 函数将它组装到一个 Manager 内，具体方法包括：

```go
(&controllers.AtReconciler{
		Client: mgr.GetClient(),
		Scheme: scheme,
	}).SetupWithManager(mgr)
```

## Layout

### api/

```shell
api
├── doc.go
├── fullvpcmigration_types.go
├── v1
│   ├── conversion.go
│   ├── doc.go
│   ├── fullvpcmigration_types.go
│   ├── register.go
│   ├── zz_generated.conversion.go
│   ├── zz_generated.deepcopy.go
│   └── zz_generated.openapi.go
├── v2
│   ├── doc.go
│   ├── fullvpcmigration_types.go
│   ├── register.go
│   ├── zz_generated.conversion.go
│   ├── zz_generated.deepcopy.go
│   └── zz_generated.openapi.go
└── zz_generated.deepcopy.go
```

#### xxx_types.go



#### doc.go

提供包级别的注释

```go
// +k8s:openapi-gen=true
// +groupName=wukong.com
// +kubebuilder:object:generate=true
 
package api
```

#### register.go

用于将 GVK-type 注册到 Scheme 中

```go
// __internal 版本
package api
 
import (
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
)
 
const (
    GroupName = "wukong.com"
)
 
var (
    // GroupVersion is group version used to register these objects
    GroupVersion = schema.GroupVersion{Group: GroupName, Version: runtime.APIVersionInternal}
 
    // SchemeBuilder is used to add go types to the GroupVersionKind scheme
    // no &scheme.Builder{} here, otherwise vk __internal/WatchEvent will double registered to k8s.io/apimachinery/pkg/apis/meta/v1.WatchEvent &
    // k8s.io/apimachinery/pkg/apis/meta/v1.InternalEvent, which is illegal
    SchemeBuilder = runtime.NewSchemeBuilder()
 
    // AddToScheme adds the types in this group-version to the given scheme.
    AddToScheme = SchemeBuilder.AddToScheme
)
 
// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
    return GroupVersion.WithKind(kind).GroupKind()
}
 
// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
    return GroupVersion.WithResource(resource).GroupResource()
}
```

```go
// v2 版本
package v2
 
import (
    "cloud.tencent.com/teleport/api"
    metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
    "k8s.io/apimachinery/pkg/runtime"
    "k8s.io/apimachinery/pkg/runtime/schema"
)
 
var (
    // GroupVersion is group version used to register these objects
    GroupVersion = schema.GroupVersion{Group: api.GroupName, Version: "v2"}
 
    // SchemeBuilder is used to add go types to the GroupVersionKind scheme
    SchemeBuilder = runtime.NewSchemeBuilder(func(scheme *runtime.Scheme) error {
        metav1.AddToGroupVersion(scheme, GroupVersion)
        return nil
    })
    localSchemeBuilder = &SchemeBuilder
 
    // AddToScheme adds the types in this group-version to the given scheme.
    AddToScheme = SchemeBuilder.AddToScheme
)
 
// Kind takes an unqualified kind and returns a Group qualified GroupKind
func Kind(kind string) schema.GroupKind {
    return GroupVersion.WithKind(kind).GroupKind()
}
 
// Resource takes an unqualified resource and returns a Group qualified GroupResource
func Resource(resource string) schema.GroupResource {
    return GroupVersion.WithResource(resource).GroupResource()
}
```

#### zz_generated.openapi.go

这是每个普通版本都需要生成的 OpenAPI 定义。这些 OpenAPI 定义必须注册到 kube-apiserver，否则将会导致 kubectl apply 等命令报 404 错误。

#### zz_generated.deepcopy.go

这个文件是 __internal 版本、普通版本中的资源对应 Go Type struct 都需要生成的深拷贝函数。

## 开发流程

以 xxx API resource 为例。

### 定义GV

在 api/v1/groupversion_info.go 文件中添加 group 和 version

### 创建GV对应的Builder

需要在 api/v1/groupversion_info.go 中创建该 GV 的 Builder。

```go
SchemeBuilder = &scheme.Builder{GroupVersion: GroupVersion}
```

### 定义Type

在 api/v1/xxx_types.go 文件中建立、更新 type struct。通常至少需要定义 Xxx（资源名的驼峰式大小写）、XxxcList（表示资源的列表）两个结构，Xxx 结构至少包含 Spec、Status 两个额外字段，对应结构 XxxSpec、XxxStatus，分别代表规格（输入参数）和状态（当前状态）。此外，相关结构上必须提供必要的 kubebuilder 注解、所有字段都应该提供 JSON tag（驼峰式大小写）：

```go
//+kubebuilder:object:root=true
//+kubebuilder:subresource:status

// Xxx is the Schema for the xxx API
type Xxx struct {
	metav1.TypeMeta   `json:",inline"`
	metav1.ObjectMeta `json:"metadata,omitempty"`
	Spec              XxxSpec   `json:"spec"`
	Status            XxxStatus `json:"status,omitempty"`
}

//+kubebuilder:object:root=true

// MoveToVpcList contains a list of MoveToVpc
type XxxList struct {
	metav1.TypeMeta `json:",inline"`
	metav1.ListMeta `json:"metadata,omitempty"`
	Items           []Xxx `json:"items"`
}
```

### 生成deepcopy

添加完新资源后需要执行下面的命令，重新生成 zz_generated.deepcopy.go 文件，该文件包含了一系列和深拷贝有关的代码：

```
make generate
```

注意，每当修改资源的任何字段，该命令都需要再次执行。makefile 已经正确处理好依赖，所有依赖 generate 的目标都会自动调用它。

### 添加GVK-Type到Builder

需要在 api/v1/xxx_types.go 文件的 init() 方法中，将定义的资源、资源列表注册到 Scheme 中的 GV 中（每个 SchemeBuilder 对应一个 GV）：

```go
func init() {
  SchemeBuilder.Register(&Xxx{}, &XxxList{})
}
```

### 注册GVK-Type到Scheme

在 main.go 的 init() 中执行 `AddToScheme()`，真正将 Xxx Type 添加到 scheme 中。

```go
utilruntime.Must(xxxv1.AddToScheme(scheme))
```

#### 注册__internal

如需要 internal 版本，修改 apis/v1/groupversion_info.go，将资源注册到 __internal 版本：

```go
SchemeBuilderInternal = runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
  s.AddKnownTypes(GroupVersionInternal, &Xxx{}, &XxxList{})
  return nil
})
```

### 定义Reconciler

kubebuilder 封装了 controller-runtime，在主文件中主要初始了`manager`，以及填充的`Reconciler`与`Webhook`，最后启动`manager`。

在 controllers/xxx_controller.go 文件中，创建 Reconciler struct。并给 Reconciler 添加 Reconcile() 方法，并在其中写入核心业务逻辑。

Reconcile() 的触发是通过 Cache 中的 Informer 获取到资源的变更事件，然后再通过生产者消费者的模式触发自己填充的 Reconcile() 方法的。

### 创建Reconciler

创建一个 Xxx API resource 对应的 Controller，其中：

- Scheme：为整个 Manager 统一的 Scheme
- Client：为整个 Manager 共享的 client

```go
if err = (&controllers.XxxReconciler{
   Client: mgr.GetClient(),
   Scheme: mgr.GetScheme(),
})
```

### 创建Manager

在`NewManager()`中主要初始化了各种配置：

- Scheme：
- Port：
- MetricsBindAddress：
- HealthProbBindAddress：
- LeaderElection：ture、false
- LeaderElectionID：

### 添加Reconciler到Manager

SetupWithManager() 把创建的 Controller 添加到 Manager 中

```go
Xxx.SetupWithManager(mgr); err != nil {
		setupLog.Error(err, "unable to create controller", "controller", "At")
		os.Exit(1)
}
```

其背后实际的工作是：

- NewControllerManagedBy()：基于现有的 Manager 创建一个空壳的 Controller。
- For()：指明本 controller 操作哪种 Type。
- Complete()：为空壳 Controller 添加 Reconciler。

```go
ctrl.NewControllerManagedBy(mgr).
		For(&xxxv1.Xxx{}).
		Complete(r)
```

### 启动Manager

```go
err := mgr.Start(ctrl.SetupSignalHandler())
```

其内部主要流程包括：

- serveMetrics()：启动监控服务
- serveHealthProbes()：启动健康检查服务
- startNonLeaderElectionRunnables()：
  - waitForCache()：启动 cache
  - startRunnable()：通过 Controller.Start() 正式启动 Controller
    - c.processNextWorkItem(ctx) --> processNextWorkItem() --> reconcileHandler() --> Do.Reconcile(ctx, req)
- startLeaderElection()：启动选主服务

### 创建CRD YAML

```shell
make manifests
```

### 创建CR YAML

需要根据 CRD 建立自己的 CR yaml 文件。

## Lab

### Install

```shell
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/darwin/amd64
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

### kubebuilder-demo

#### 构建代码

- 默认需求：go 1.8，kubebuilder 3.6.0
- 初始化 kubebuilder

```shell
mkdir kubebuilder-demo & cd kubebuilder-demo
kubebuilder init \
--domain wukong.com \
--repo github.com/rebirthmonkey/k8s-dev/kubebuilder-demo
```

- 创建 API：创建对应的 api/ 和 controllers/

```shell
kubebuilder create api --group ingress --version v1 --kind App
```

- 创建 manifests：在 config/crd/bases 中创建了对应 CRD 的 YAML 文件
```shell
make manifests
```

- 部署 CRD：将 CRD 部署到 k8s 集群中

```shell
make install
kubectl get crds
make uninstall
```

- 在 api/v1/app_types.go 中添加代码：
```shell
# 本例中不添加额外代码
```

- 在 controller/Reconcile() 中添加业务代码

```go
_ = log.FromContext(ctx)
fmt.Println("XXXXXXXX app changed", "ns", req.Namespace)
return ctrl.Result{}, nil
```

#### Go进程运行

- 运行 controller

```shell
make run  
```

- 部署 CR：此处 CR 未填入具体内容，因为只是为了测试 Reconsile() log 是否输出。
```shell
kubectl apply -f config/samples/ingress_v1_app.yaml
kubectl delete -f config/samples/ingress_v1_app.yaml 
```

#### 容器运行

- 容器镜像打包

```shell
export IMG=wukongsun/kubebuilder-demo:v1
make docker-build
```

- 在 k8s 集群中部署、运行：在 config/ 目录下创建、渲染 YAML 文件，并执行。

```shell
make deploy
make undeploy
```

- 上传镜像到镜像仓库：默认是到 dockerhub

```shell
make docker-push
```



### kubebuilder-at

AT 是个工具，用于在指定时间运行指定的命令，通过它的 schedule 和 command 2 个属性来设置。启动一个称为 AT 的 CR，在 AT 中 schedule 配置的 UTC 时间、执行在 CR 中 command 配置的命令。整个执行过程（CR 的 status）分为 3 个阶段：pending、running、done。

- 创建脚手架

```shell
mkdir kubebuilder-at && cd kubebuilder-at
kubebuilder init \
--domain wukong.com \
--repo github.com/rebirthmonkey/k8s-dev/kubebuilder-at
```

- 创建 API/controller

```shell
kubebuilder create api \
--group demo \
--version v1 \
--kind At
```

- 定义 API：在 `api/v1/at_types.go` 文件中定义 Go Type
- 创建 CRD：CRD 将会在 `config/crd/bases` 中创建

```shell
make manifests
```

- 安装 CRD

```shell
kubectl apply -f config/crd/bases/demo_v1_at.yaml
```

- 创建 CR：内容更新到 `config/samples` 中
- 添加 CR Yaml

```shell
kubectl apply -f config/samples/at_v1_at.yaml
```

- 编写 controller：`controllers/at_controller.go` 中添加逻辑代码

- 运行 operator

```shell
go run main.go
```

- 容器镜像打包

```shell
export IMG=docker.io/xxx/yyy:v1
make docker-build
make docker-push
```

#### 开发步骤

- 修改 api/v1alpha1/at_types.go：修改了 const、AtSpec 和 AtStatus 三处

```go
const (
  PhasePending = "PENDING"
  PhaseRunning = "RUNNING"
  PhaseDone    = "DONE"
)

// EDIT THIS FILE!  THIS IS SCAFFOLDING FOR YOU TO OWN!
// NOTE: json tags are required.  Any new fields you add must have json tags for the fields to be serialized.

// AtSpec defines the desired state of At
type AtSpec struct {
// INSERT ADDITIONAL SPEC FIELDS - desired state of cluster
// Important: Run "make" to regenerate code after modifying this file

  Schedule string `json:"schedule,omitempty"`
  // Command is the desired command (executed in a Bash shell) to be executed.
  Command string `json:"command,omitempty"`
}

// AtStatus defines the observed state of At
type AtStatus struct {
  // INSERT ADDITIONAL STATUS FIELD - define observed state of cluster
  // Important: Run "make" to regenerate code after modifying this file
  Phase string `json:"phase,omitempty"`
}
```

- 更新修改

```shell
make
```

- 修改 controllers/at_controllers.go：详细代码见 [at_controllers.go](kubebuilder-at/controllers/at_controller.go)

- 启动 operator

```shell
go run main.go
```

- 创建 CR：内容改为

```yaml
apiVersion: at.wukong.com/v1
kind: At
metadata:
  name: at-sample
spec:
  schedule: "2022-06-11T03:50:31Z"
  command: "echo YAY"
```

```shell
kubectl apply -f config/samples/at_v1_at.yaml
```

## Ref

1. [kubebuilder Book](https://book.kubebuilder.io/)
