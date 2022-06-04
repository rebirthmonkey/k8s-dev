# controller-tools

基于 code-generator，进一步自动化 type.go 文件及 CRD yaml 文件的自动化生成。

## Installation
```shell
git clone https://github.com/kubernetes-sigs/controller-tools.git
git checktout v0.8.0
go install ./cmd/{controller-gen,type-scaffold}
```

## Lab
- 生成 type.go 文件
```shell
type-scaffold --kind Foo # 需要把内容拷贝到 type.go 文件中 
```

- 生成 deepcopy
```shell
controller-gen object paths=./pkg/apis/wukong.com/v1/types.go
```

- 制定 marker 标记：register.go 文件
```go
// +groupName=wukong.go
package v1
```

- 生成 CRD 文件
```shell
controller-gen crd paths=./... output:crd:dir=congfig/crd 
```

- 更新 types.go 文件，在 FooSpec 中 添加
```go
	Name string `json:"name"`
	Replicas int32 `json:"replicas"`
```

- 重新生成 CRD 文件
```shell
controller-gen crd paths=./... output:crd:dir=congfig/crd 
```

- 自建 CR 文件，并 apply
```shell
kubectl apply config/cr/test.yaml 
```

- 执行 main.go 文件
```shell
go run main.go 
```

