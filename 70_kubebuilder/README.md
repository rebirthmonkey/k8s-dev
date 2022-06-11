# kubebuilder

## 简介



## 架构

### Manager



<img src="figures/image-20220608172034690.png" alt="image-20220608172034690" style="zoom:50%;" />

### Controller



### Webhook



### Cluster

#### Client



#### Cache





## Dev

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

#### 添加 CR
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


## Ref

1. [kubebuilder Book](https://book.kubebuilder.io/introduction.html)
2. 

