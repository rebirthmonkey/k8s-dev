# Sample Controller/Controller

基于 sample controller 样例来创建一个名为 App 的 CRD，然后由 app-controller 自动创建/删除 App 中定义的 deployment、service、ingress 资源对象。

<img src="figures/image-20220912133710917.png" alt="image-20220912133710917" style="zoom:50%;" />

## Lab

### 自动生成代码


- 创建 controller 的脚手架，自动生成 deepcopy、clientset、informer、lister

```shell
./scripts/update-codegen.sh
```

### 生成 CRD/CR

- 自动生成 manifests/CRD

```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```


- 手写 CR 文件

### 编写 controller

在 `/pkg/controller` 目录下的 controller.go 文件中编写 controller 的业务逻辑。

### 启动


- 启动程序

```shell
go run ./cmd/app.go --kubeconfig=$HOME/.kube/config 
```

- 在 k8s 内创建资源

```shell
kubectl apply -f manifests/crd
kubectl get deployments
kubectl get svc
kubectl get ingress
```

- 验证可用性

```shell
curl -H 'Host:wukong.com' http://127.0.0.1:80
```

