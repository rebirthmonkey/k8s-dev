# sample-apiserver

k8s 提供了 [kubernetes/sample-apiserver](https://github.com/kubernetes/sample-apiserver) release-1.25 这个示例，但是这个例子依赖于主 kube-apiserver。即使不使用 authn/authz 或 kube-aggregator，也是如此。需要通过 --kubeconfig 来指向一个主 kube-apiserver，示例中的 SharedInformer 依赖于会连接到主 kube-apiserver 来访问 k8s 资源。

## Lab

### 环境准备

#### 准备 k8s 集群

准备一个 k8s 集群，提供主 kube-apiserver

```shell
go install sigs.k8s.io/kind@v0.14.0
kind create cluster
kubectl config use-context kind-kind
```

#### 客户端访问凭证

```shell
cd configs/cert
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt # 可随意填写
openssl req -out client.csr -new -newkey rsa:4096 -nodes -keyout client.key -subj "/CN=development/O=system:masters"
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out client.crt
openssl pkcs12 -export -in ./client.crt -inkey ./client.key -out client.p12 # 密码设置为 P@ssw0rd
```

#### 代码更新

```shell
go mod tidy
go mod vendor
hack/update-codegen.sh
```

#### 启动 Etcd

```shell
etcd # 启动 Etcd 数据库
```

### 进程部署

#### 启动服务

通过进程启动 aa-server

```shell
GODEBUG=x509sha1=1 go run main.go --secure-port 8443 --etcd-servers http://127.0.0.1:2379   --kubeconfig ~/.kube/config --authentication-kubeconfig ~/.kube/config --authorization-kubeconfig ~/.kube/config --client-ca-file=configs/cert/ca.crt # Go 1.18 之后得注明 GODENBUG 参数
```

#### 测试

##### 直接调用aaserver

直接通过 URL 调用 aaserver，如果要用 kubectl，还需要配置 kind k8s 集群。

- List all API resources：

```shell
curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis
```

- List flunders resources：

```shell
curl -k --cert-type P12 --cert configs/cert/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/wardle.example.com/v1alpha1/namespaces/default/flunders
```

##### 通过kube-aggregator

- 创建 APIService

```shell
kubectl apply -f artifacts/example/ns.yaml
kubectl apply -f artifacts/example/apiservice.yaml
kubectl apply -f artifacts/example/service-ext.yaml
kubectl apply -f artifacts/example/endpoint.yaml
```

- 确认 aaserver 已注册资源

```shell
kubectl get apiservices.apiregistration.k8s.io | grep wardle
kubectl -n wardle get svc api -o yaml  
kubectl -n wardle get ep api -o yaml 
```

- 创建 flunders 资源

```shell
kubectl apply -f artifacts/flunders/flunder.yaml
```

- 通过 get -raw 调用

```shell
kubectl get --raw "/apis/wardle.example.com/v1alpha1/namespaces/wardle/flunders"
```

#### cleanup

```shell
kubectl delete -f artifacts/flunders/flunder.yaml
kubectl delete -f artifacts/example
```

### k8s 部署

#### 构建镜像

```shell
docker build -t wukongsun/sample-apiserver:0.1 .
kind load docker-image wukongsun/sample-apiserver:0.1 # load image to the kind cluster
docker exec kind-control-plane crictl images | grep wukongsun # 确认镜像已加载
```

#### 部署k8s资源

```shell
kubectl apply -f artifacts/example/ns.yaml
kubectl apply -f artifacts/example/sa.yaml
kubectl apply -f artifacts/example/rbac.yaml
kubectl apply -f artifacts/example/rbac-bind.yaml
kubectl apply -f artifacts/example/auth-delegator.yaml
kubectl apply -f artifacts/example/auth-reader.yaml
```

#### 启动服务

```shell
kubectl apply -f artifacts/example/deployment.yaml
kubectl -n wardle get pods
kubectl apply -f artifacts/example/service-k8s.yaml
```

#### 测试

- 注册 APIService

```shell
kubectl apply -f artifacts/example/apiservice.yaml
kubectl get apiservice v1alpha1.wardle.example.com # 等待直到aaserver服务运行，即AVAILABLE为true
```


- 创建 flunders 资源
```shell
kubectl apply -f artifacts/flunders/flunder.yaml
```

- 通过 get --raw 调用

```shell
kubectl get --raw "/apis/wardle.example.com/v1alpha1/namespaces/wardle/flunders"
```

#### cleanup

```shell
kubectl delete -f artifacts/flunders/flunder.yaml
kubectl delete -f artifacts/example
```


