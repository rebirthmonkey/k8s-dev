# kubebuilder

kubebuilder 为 Operator 搭建好了基本的代码框架，生成了一堆文件，涵盖了自定义 controller 的代码和一个示例 CRD。

## controller-runtime

controller-runtime 库包含若干 Go 库，用于快速构建

- controller-manager：
- controller：
- dynamic clientset：

kubebuilder 依赖于 controller-runtime 库，使用 controller-runtime 的 Client 接口来实现针对 k8s 资源的 CRUD 操作。

### Manager

controller-runtime 由 Manager（等价于 k8s 的 controller-manager）串联起来，用于启动（Manager.Start） controller，管理被多个 controller 共享的依赖，例如 Cache、Client、Scheme。通过 manager.Manager 来创建 client.Client，SDK 生成的代码中包含创建 Manager 的逻辑，Manager 持有一个 Cache 和一个 Client。

<img src="figures/image-20220608172034690.png" alt="image-20220608172034690" style="zoom:50%;" />

#### 启动流程

- 创建 Manager：
  - 创建并注册 scheme
  - 创建 cluster（client+cache）
  - 为 runnable 创建 map
- 注册 Runnable：添加 runnable 到 map
- 启动 Manager：启动所有注册的 runnable（map）

### Controller

Controller 可能会监控多种类型的对象（如 Pod + ReplicaSet + Deployment），但是 Controller 的 Reconciler 一般仅仅处理单一类型的对象。controller 从 Manager 得到各种共享对象，它自己创建一个工作队列。并从工作队列中获取 event，转给 Reconciler。

当 A 类型的对象发生变化后，如果 B 类型的对象必须更新以响应，可以使用 EnqueueRequestFromMapFunc 来将一种类型的事件映射为另一种类型。如 Deployment 的 Controller 可以使用 EnqueueRequestForObject、EnqueueRequestForOwner 实现：

1. 监控 Deployment 事件，并将 Deployment 的 Namespace/Name 入队
2. 监控 ReplicaSet 事件，并将创建它的 Deployment（Owner）的 Namespace/Name 入队

类似 ReplicaSet 的控制器也可以监控 ReplicaSet 和 Pod 事件。

reconcile.Request 入队时会自动去重，也就是说一个 ReplicaSet 创建的多个 Pod 事件可能仅仅触发 ReplicaSet 控制器的单次调用。

#### Reconciler

Reconciler 是 Controller 的核心逻辑所在，它负责调和使 status 逼近期望状态 spec。例如，当针对 ReplicaSet 对象调用 Reconciler 时，发现 ReplicaSet 要求 5 实例，但是当前系统中只有 3 个 Pod。这时 Reconciler 应该创建额外的两个 Pod，并且将这些 Pod 的 OwnerReference 指向前面的 ReplicaSet。

Reconciler 通常仅处理一种类型的对象，OwnerReference 用于从子对象（如 Pod）触发父对象的调和（如 ReplicaSet）操作。

##### SetupWithManager()

Controller 还应该实现一个 SetupWithManager(mgr ctrl.Manager) 方法，此方法将本 controller 注册给 controller-manager：

```go
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
// 创建一个被mgr管理的控制器，指定控制器喧嚣
  return ctrl.NewControllerManagedBy(mgr).WithOptions(controller.Options{
// 并发
    MaxConcurrentReconciles: r.Concurrence,
  }).For(&tcmv1.MoveToVpc{}).Complete(r)
}
```

##### WithEventFilter()

如果想在进入 controller 之前就过滤掉不符合条件（如已经标记为删除）的资源，则需要修改 SetupWithManager() 方法，增加 WithEventFilter 调用：

```go
func (r *Reconciler) SetupWithManager(mgr ctrl.Manager) error {
  return ctrl.NewControllerManagedBy(mgr).WithOptions(controller.Options{
MaxConcurrentReconciles: r.Concurrence,}).For(&tcmv1.MoveToVpc{}).WithEventFilter(predicate.Funcs{
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
  }).Complete(r)
}
```

#### 注册 Controller

在 `cmd/manager/main.go` 中实例化 controller，并调用 SetupWithManager() 注册到 controller-manager 中：

```go
    if err = movetovpc.NewMoveToVpcReconciler(
        movetovpc.Config{
            Client:      mgr.GetClient(),
            Concurrence: concurrence,
        },
    ).SetupWithManager(mgr); err != nil {
        setupLog.Error(err, "unable to create controller", "controller", "MoveToVpc")
        os.Exit(1)
    }
```

### Cluster

#### Client

Client 是对 client-go 中 clientSet 的封装，用于实现针对 kube-apiserver 的 CRUD 操作，读写客户端通常是分离（split）的。

#### Cache

Cache 实际是 client-go 中 Informer 的包装，为读客户端提供本地缓存，支持监听更新缓存的事件。

- DelegatingClient：从 Cache 中读取（Get/List），写入（Create/Update/Delete）请求则直接发送给 API Server。使用 Cache 可以大大减轻 API Server 的压力，随着缓存的更新，读操作会达成最终一致。

### Webhook





## 开发步骤

以 xxx 为例。

### 添加 GVK

添加 group name 和 version

### 定义 struct

在 group/version/xx_types.go 文件中建立、更新 CRD 对应的 struct。通常至少需要定义 Xxx（资源名的驼峰式大小写）、XxxcList（表示资源的列表）两个结构，Xxx 结构至少包含 Spec、Status 两个额外字段，对应结构 XxxSpec、XxxStatus，分别代表规格（输入参数）和状态（当前状态）。此外，相关结构上必须提供必要的 kubebuilder 注解、所有字段都应该提供JSON tag（驼峰式大小写）：

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

##### 代码生成

添加完新资源后需要执行下面的命令，重新生成zz_generated.deepcopy.go文件，该文件包含了一系列和深拷贝有关的代码：

```
make generate
```

注意，每当修改资源的任何字段，该命令都需要再次执行。makefile 已经正确处理好依赖，所有依赖 generate 的目标都会自动调用它。

### 编写业务逻辑

在 controller/kind/xx_controller.go 文件的 Reconcile() 中写入核心业务逻辑，然后可以运行 operator `make run`。

### Scheme 注册

需要在 apis/v1/xxx_types.go 文件的 init() 方法中，将定义的资源、资源列表注册到 Scheme：

```go
func init() {
  SchemeBuilder.Register(&Xxxx{}, &XxxList{})
}
```

### __internal 注册

修改 apis/v1/groupversion_info.go，将资源注册到 __internal 版本：

```go
SchemeBuilderInternal = runtime.NewSchemeBuilder(func(s *runtime.Scheme) error {
  s.AddKnownTypes(GroupVersionInternal, &Xxx{}, &XxxList{})
  return nil
})
```

### 资源生命周期回调函数注册

修改 cmd/apiserver/main.go 中的 ResourcesConfig，为资源注册生命周期回调函数：

```go
ResourcesConfig: map[string]map[string]apiserver.ResourceConfig{
  apiv1.ResourceMovetovpcs: {
    NewFunc: func() runtime.Object { return &apiv1.Xxx{} },  // 创建资源对象的回调
    NewListFunc: func() runtime.Object { return &apiv1.XxxList{} },  // 创建资源对象列表的回调
  },
}
```

### 编写 Artifects/Manifests

需要根据 CRD 建立自己的 CR yaml 文件。

## Lab

### Install

```shell
curl -L -o kubebuilder https://go.kubebuilder.io/dl/latest/darwin/amd64
chmod +x kubebuilder && mv kubebuilder /usr/local/bin/
```

### kubebuilder-demo

- 默认需求：go 1.8，kubebuilder 3.6.0
- 初始化 kubebuilder

```shell
mkdir kubebuilder-demo & cd kubebuilder-demo
kubebuilder init \
	--domain wukong.com \
	--repo github.com/rebirthmonkey/k8s-dev/kubebuilder-demo
```

- 创建 API：创建对应的 controller

```shell
kubebuilder create api --group ingress --version v1 --kind App
```

- 部署 CRD
```shell
make install
kubectl get crds
```

- 在 controller/Reconcile() 中添加代码
```go
    _ = log.FromContext(ctx)
	fmt.Println("XXXXXXXX app changed", "ns", req.Namespace)
	return ctrl.Result{}, nil
```

- 运行 operator
```shell
make run  
```

- 部署 CR：此处 CR 未填入具体内容，因为只是为了测试 Reconsile() log 是否输出。
```shell
kubectl apply -f config/samples/ingress_v1_app.yaml
kubectl delete -f config/samples/ingress_v1_app.yaml 
```

- 容器镜像打包

```shell
export IMG=docker.io/xxx/yyy:v1
make docker-build
make docker-push
```

### kubebuilder-at

At 是个工具，用于在指定时间运行指定的命令，通过它的 schedule 和 command 2 个属性来设置。启动一个 称为 AT 的 CR，在 AT 中 schedule 配置的 UTC 时间、执行在 CR 中 command 配置的命令。整个执行过程（CR 的 status）分为 3 个阶段：pending、running、done。

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
              --group at \
              --version v1 \
              --kind At
```

- 创建/安装 CRD

```shell
make install
```

- 运行 Operator

```shell
make run
```

- 添加 CR Yaml

```shell
kubectl apply -f config/samples/at_v1_at.yaml
kubectl delete -f config/samples/cnat_v1alpha1_at.yaml
make uninstall
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
make run
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

