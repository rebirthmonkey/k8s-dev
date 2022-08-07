# controller

## 简介

在k8s中，controller实现了一个控制循环，它通过kube-apiserver观测集群中的共享状态，进行必要的变更，尝试把资源对应的当前状态期望的目标状态。controller负责执行例行性任务来保证集群尽可能接近其期望状态。典型情况下控制器读取.spec字段，运行一些逻辑，然后修改 .status 字段。K8S自身提供了大量的控制器，并由控制器管理器统一管理。controller可以对k8s的核心资源（如pod、deployment）等进场操作，通常是controller manager的一部分，但它也可以观察并操作用户自定义资源。

自定义实现一个简单的 controller，其作用是一旦watch到有新的、带有特殊annotation的service资源被添加，则会自动添加ingress资源。

### 控制循环

- 读取资源的状态：通常采用事件驱动模式
- 改变资源的状态：
- 通过kube-apiserver更新资源的状态：
- 循环执行以上3步：






## Lab

### Install

- 安装 Nginx Ingress Controller


### 通过 GO 进程运行

- 启动 controller
```shell
go run main.go 
```

- 部署 deployment 和 service
```shell
kubectl apply -f deploy-service.yaml 
```

- 验证 ingress 是否自动添加

```shell
kubectl get ingress # 有ingress，已经自动启动 
```

- 验证 Nginx Service 页面
```shell
curl -H 'Host:wukong.com' http://127.0.0.1:80
```

- 在 service 中删除 annotations

```shell
kubectl edit svc nginx
kubectl get ingress # ingress自动删除
```

### 通过 k8s 容器运行

- Docker 镜像 build
```shell
docker build -t ingress-manager:1.0.1 .
```

- 启动 ingress manager
```shell
kubectl apply -f manifests
```

- 部署 deployment 和 service
```shell
kubectl apply -f deploy-service.yaml 
```

- 验证 ingress 是否自动生成
```shell
kubectl get ingress
```

- 验证 Nginx Service 页面
```shell
curl -H 'Host:wukong.com' http://127.0.0.1:80
```

