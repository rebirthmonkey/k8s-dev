# App-controller

## 简介

基于 sample controller 来创建一个名为 App 的 CRD，然后由 controller 自动创建/删除 deployment、service、ingress 资源对象。

<img src="figures/image-20220912133710917.png" alt="image-20220912133710917" style="zoom:50%;" />

## Lab


- 创建 controller 的脚手架

```shell
./scripts/update-codegen.sh
```

- 自动生成 manifests/CRD

```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```


- 手写 CR 文件
- 启动程序

```shell
go run ./cmd/app.go --kubeconfig=$HOME/.kube/config 
```

- 在 k8s 内启动资源

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

