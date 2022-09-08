# CRD Lab
## 简介

基于 55_lab-crd-app 生成 CRD 以及 controller。


## Lab

- 下载 vendor 包
```shell
go mod vendor 
```

- 自动生成 deepcopy、clientset、informer、lister
```shell
./hack/update-codegen.sh 
```

- 替换代码中的类型等参数

- 生成 manifests
```shell
controller-gen crd paths=./... output:crd:dir=manifests 
```

- 生成 CRD 及 CR 资源
```shell
kubectl apply -f appcontroller.wukong.com_apps.yaml
kubectl apply -f example.apps.yaml 
```

- 启动程序
```shell
go run . --kubeconfig=/Users/ruan/.kube/config 
```