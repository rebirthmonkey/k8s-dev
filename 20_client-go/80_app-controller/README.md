# App-Controller

## 简介

本案例来源于[此代码](https://github.com/baidingtech/operator-lesson-demo.git)，它自定义实现一个简单的 controller，其作用是 watch 到有新的、带有特殊 annotation（ingress/http:true) 的 service 资源，一旦有新的资源被添加，则会自动添加对应的 ingress 资源。

<img src="figures/image-20220912100551553.png" alt="image-20220912100551553" style="zoom:50%;" />

Controller 监听 Service 资源

- 新增 Service：
  - 包含指定 annotation（ingress/http:true），创建 ingress 资源对象
  - 不包含指定 annotation，忽略
- 删除 Service：
  - 包含指定 annotation（ingress/http:true），删除 ingress 资源对象
  - 不包含指定 annotation，忽略
- 更新 Service：
  - 包含指定 annotation（ingress/http:true），检查 ingress 资源对象是否存在。不存在则创建，存在则忽略。
  - 不包含指定 annotation，检查 ingress 资源是否存在。存在则删除，不存在则忽略。

## Lab

### Install

- 安装 Nginx Ingress Controller
- 验证 Nginx Ingress Controller

### 通过 Go 进程运行

- 启动 controller
```shell
go run cmd/app-controller.go
```

- 验证 ingress 为空
```shell
kubectl get ingress # 无ingress 
```

- 部署 deployment 和 service

```shell
kubectl apply -f test/deploy-service.yaml 
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
docker build -t app-controller:1.0.1 .
```

- 启动 ingress manager
```shell
kubectl apply -f manifests
```

- 部署 deployment 和 service
```shell
kubectl apply -f test/deploy-service.yaml 
```

- 验证 ingress 是否自动生成
```shell
kubectl get ingress
```

- 验证 Nginx Service 页面
```shell
curl -H 'Host:wukong.com' http://127.0.0.1:80
```

