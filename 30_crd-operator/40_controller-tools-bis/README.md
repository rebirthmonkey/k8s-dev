# controller-tools

基于 code-generator，controller-tools 进一步自动化 type.go 文件及 CRD yaml 文件的生成。

## Installation

```shell
git clone https://github.com/kubernetes-sigs/controller-tools.git
cd controller-tools & git checktout -b v0.8.0
go install ./cmd/{controller-gen,type-scaffold}
```

## Lab

- 生成 types.go 文件

```shell
type-scaffold --kind Foo > types.go # 需要把内容拷贝到/pkg/apis/wukong.com/v1/types.go 文件中
```

- 在 types.go 中添加包头

```go
package v1

import metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
```

- 在 pkg/apis/wukong.com/v1/ 下生成 zz_generated.deepcopy.go 文件

```shell
controller-gen object paths=./pkg/apis/wukong.com/v1/types.go
```

- 制定 marker 标记：pkg/apis/wukong.com/v1/register.go 文件，并添加相应代码

```go
// +groupName=wukong.com
package v1

import (
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/apimachinery/pkg/runtime/serializer"
)

var (
	Scheme       = runtime.NewScheme()
	GroupVersion = schema.GroupVersion{Group: "wukong.com", Version: "v1"}
	Codecs       = serializer.NewCodecFactory(Scheme)
)
```

- 生成 CRD 文件

```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```

- 更新 types.go 文件，在 FooSpec 中添加

```go
	Name string `json:"name"`
	Replicas int32 `json:"replicas"`
```

- 重新生成 CRD 文件

```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```

- 在 ks8 中创建 CRD

```shell
kubectl apply -f manifests/wukong.com_foos.yaml
kubectl get crds
```

- 自建 CR 文件，并 apply

```shell
kubectl apply -f manifests/cr.yaml
```

- 启动 controller

```shell
go run ./cmd/controller-tools.go 
```

## 标准代码

更新后标准的代码在[这里](../40_controller-tools-bis/README.md)

