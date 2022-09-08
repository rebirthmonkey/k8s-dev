# App
基于 k8s 的 [sample controller](https://github.com/kubernetes/sample-controller) 改造而成

- 下载 vendor 包
```shell
go mod tidy
go mod vendor 
```

- 自动生成 deepcopy、clientset、informer、lister
```shell
./scripts/update-codegen.sh 
```

- 替换代码中的类型等参数

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

- 启动程序

```shell
go run ./cmd/app.go --kubeconfig=$HOME/.kube/config 
```

- 在 k8s 中生成 CRD 及 CR 资源

```shell
kubectl apply -f app.wukong.com_apps.yaml
kubectl apply -f example.apps.yaml 
```
