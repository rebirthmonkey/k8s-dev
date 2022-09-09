# kubebuilder

## 架构

### Manager



<img src="figures/image-20220608172034690.png" alt="image-20220608172034690" style="zoom:50%;" />

### Controller



### Webhook



### Cluster

#### Client



#### Cache



## controller-runtime

该项目包含若干 Go 库，用于快速构建 controller。kubebuilder 依赖于此项目，使用 controller-runtime 的 Client 接口来实现针对 k8s 资源的 CRUD 操作。

- Manager：用于启动（Manager.Start） controller，管理被多个 controller 共享的依赖，例如 Cache、Client、Scheme。通过 manager.Manager 来创建 client.Client，SDK 生成的代码中包含创建 Manager 的逻辑，Manager 持有一个 Cache 和一个 Client。
- Cache：为读客户端提供本地缓存，支持监听更新缓存的事件。
  - DelegatingClient：从 Cache 中读取（Get/List），写入（Create/Update/Delete）请求则直接发送给 API Server。使用 Cache 可以大大减轻 API Server 的压力，随着缓存的更新，读操作会达成最终一致。
- Client：实现针对 kube-apiserver 的 CRUD 操作，读写客户端通常是分离（split）的。

### Controller

controller 持有一个 Reconciler，此外它从 Manager 得到各种共享对象，它自己创建一个工作队列。Controller 可能会监控多种类型的对象（如 Pod + ReplicaSet + Deployment），但是 Controller 的 Reconciler 一般仅仅处理单一类型的对象。

当 A 类型的对象发生变化后，如果 B 类型的对象必须更新以响应，可以使用 EnqueueRequestFromMapFunc 来将一种类型的事件映射为另一种类型。如 Deployment 的 Controller 可以使用 EnqueueRequestForObject、EnqueueRequestForOwner 实现：

1. 监控 Deployment 事件，并将 Deployment 的 Namespace/Name 入队
2. 监控 ReplicaSet 事件，并将创建它的 Deployment（Owner）的 Namespace/Name 入队

类似 ReplicaSet 的控制器也可以监控 ReplicaSet 和 Pod 事件。

reconcile.Request 入队时会自动去重，也就是说一个 ReplicaSet 创建的多个 Pod 事件可能仅仅触发 ReplicaSet 控制器的单次调用。

#### Reconciler

Reconciler 是 Controller 的核心逻辑所在，它负责调和使 status  逼近期望状态 spec。例如，当针对 ReplicaSet 对象调用 Reconciler 时，发现 ReplicaSet 要求 5 实例，但是当前系统中只有 3 个 Pod。这时 Reconciler 应该创建额外的两个 Pod，并且将这些 Pod 的 OwnerReference 指向前面的 ReplicaSet。

Reconciler 通常仅处理一种类型的对象，OwnerReference 用于从子对象（如 Pod）触发父对象的调和（如 ReplicaSet）操作。

## 开发步骤

### group/version/xx_types.go

建立、更新 CRD 对应的 struct，然后需要运行 `make`。

### controller/kind/xx_controller.go

在 Reconcile() 中写入核心业务逻辑，然后可以运行 operator `make run`。

### Config/samples/xx.yaml

需要根据 CRD 建立自己的 CR。


## Lab

### kubebuilder-demo

- 初始化 kubebuilder
```shell
kubebuilder init --domain wukong.com --repo github.com/rebirthmonkey/kubebuilder-demo
```

- 创建 API
```shell
kubebuilder create api --group ingress --version v1 --kind App
```

- install CRDS
```shell
make install
```

- 在 controller/Reconcile() 中添加代码
```go
    _ = log.FromContext(ctx)
	fmt.Println("XXXXXXXX app changed", "ns", req.Namespace)
	return ctrl.Result{}, nil
```

- run operator
```shell
make run  
```

- 添加 CR：此处 CR 未填入具体内容，因为只是为了测试 Reconsile() log 是否输出。
```shell
kubectl delete -f config/samples/ingress_v1beta1_app.yaml 
```


### kubebuilder-at

启动一个 称为 AT 的 CR，在 AT 中 schedule 配置的 UTC 时间、执行在 CR 中 command 配置的命令。

整个执行过程（CR 的 status）分为 3 个阶段：pending、running、done。

#### 创建脚手架
```shell
$ mkdir kubebuilder-at && cd $_

$ kubebuilder init \
              --domain wukong.com \
              --repo github.com/rebirthmonkey/k8s-dev/kubebuilder-at
```

#### 创建 API/controller
```shell
$ kubebuilder create api \
              --group cnat \
              --version v1alpha1 \
              --kind At
Create Resource under pkg/apis [y/n]?
y
Create Controller under pkg/controller [y/n]?
y
...
```

#### 创建/安装 CRD
```shell
make install
```

#### 运行 Operator
```shell
make run
```

#### 添加 CR Yaml

```shell
kubectl apply -f config/samples/cnat_v1alpha1_at.yaml
kubectl delete -f config/samples/cnat_v1alpha1_at.yaml
```

#### 修改 api/v1alpha1/at_types.go

修改了 const、AtSpec 和 AtStatus 三处

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

```shell
make
```

#### 修改 controllers/at_controllers.go
详细代码见 [at_controllers.go](30_kubebuilder-at/at_controller.go)

#### 启动 operator
```shell
make run
```

#### 创建 CR
CR 内容改为
```yaml
apiVersion: cnat.wukong.com/v1alpha1
kind: At
metadata:
  name: at-sample
spec:
  schedule: "2022-06-11T03:50:31Z"
  command: "echo YAY"
```

```shell
kubectl apply -f config/samples/cnat_v1alpha1_at.yaml
```

