# controller

## Lab

### Go 启动
- 安装 Nginx Ingress Controller
- 部署 deployment 和 service
- check 自动添加

```shell
kubectl get ingress # 有ingress，已经自动启动 
```

- 删除 annotations
```shell
kubectl edit svc ingress
kubectl get ingress # ingress自动删除
```

- check Ngix 页面
```shell
curl -H 'Host:example.com' http://127.0.0.1:80
```



### k8s 启动

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

- check ingress 的自动生成
```shell
kubectl get ingress
```