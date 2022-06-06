# CRD

## 简介

- 查看所有资源：包括内建及自定义
```shell
kubectl api-resources 
```

### Properties

- metadata.name：该资源的 ID
- spec.name：只是一个名为 name 的属性


## Lab

### CRD

- Create CRD
```shell
kubectl apply -f crd.yaml
kubectl get crds
```

- Create CR
```shell
kubectl apply -f cr.yaml
kubectl get crd1s 
```

