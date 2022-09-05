# Ingress-Manager Controller

自定义实现一个简单的 controller，其作用是一旦 watch 到有新的、带有特殊 annotation 的 service 资源被添加，则会自动添加 ingress 资源。

## Install

- 安装 Nginx Ingress Controller

## 通过 Go 进程运行

- 启动 controller
```shell
go run cmd/controller.go
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

## 通过 k8s 容器运行

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

