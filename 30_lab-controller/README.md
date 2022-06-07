# controller

自定义简单版 controller


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

- 在 service 中删除 annotations
```shell
kubectl edit svc ingress
kubectl get ingress # ingress自动删除
```

- 验证 Nginx Service 页面
```shell
curl -H 'Host:wukong.com' http://127.0.0.1:80
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

