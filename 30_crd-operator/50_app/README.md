# Sample Controller/App
基于 k8s 社区的例子 [sample controller](https://github.com/kubernetes/sample-controller) 改造而成的 app，其原理是当创建一个 app 资源时，对应的创建 deployment、service 和 ingress 资源。

## Lab

### Install

- 下载 vendor 包
```shell
go mod tidy
go mod vendor 
```

### 自动生成代码

- 自动生成 deepcopy、clientset、informer、lister

```shell
./scripts/update-codegen.sh 
```

- 替换代码中的类型等参数
### 编写 controller

在 `/pkg/controller` 目录下的 controller.go 文件中编写 controller 的业务逻辑。

### 生成 CRD/CR

- 自动生成 manifests/CRD

```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```

- 手动生成 manifests/CR：example.apps.yaml
```yaml
apiVersion: app.wukong.com/v1
kind: App
metadata:
  name: app-demo
spec:
  deployment:
    name: app-deployment-demo
    image: nginx:latest
    replicas: 2
  service:
    name: app-service-demo
  ingress:
    name: app-ingress-demo
```

### 启动

- 启动程序

```shell
go run ./cmd/app.go --kubeconfig=$HOME/.kube/config 
```

- 在 k8s 中生成 CRD 及 CR 资源

```shell
kubectl apply -f ./manifests/app.wukong.com_apps.yaml
kubectl apply -f ./manifests/example.apps.yaml 
```

### 标准代码

更新后标准的代码在[这里](../50_app-bis/README.md)

