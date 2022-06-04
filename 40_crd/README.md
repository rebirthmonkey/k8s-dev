# CRD

## 简介

- 查看所有资源：包括内建及自定义
```shell
kubectl api-resources 
```

## Lab

### CRD

- Create CRD
```shell
kubectl apply -f crd.yaml
kubectl get crd
```

- Create CR
```shell
kubectl apply -f cr.yaml
kubectl get crds 
```

### code-generator

- 安装 code-generator
```shell
git clone https://github.com/kubernetes/code-generator.git
go install code-generator/cmd/{client-gen,lister-gen,informer-gen,deepcopy-gen}
```