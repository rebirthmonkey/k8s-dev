# 10_10_sample-apiserver

k8s 提供了 [kubernetes/10_10_sample-apiserver](https://github.com/kubernetes/10_10_sample-apiserver) release-1.25 这个示例，但是这个例子依赖于主 kube-apiserver。即使不使用 authn/authz 或 kube-aggregator，也是如此。需要通过 --kubeconfig 来指向一个主 kube-apiserver，示例中的 SharedInformer 依赖于会连接到主 kube-apiserver 来访问 k8s 资源。

## 准备 kube-apiserver

准备一个 k8s 集群，提供主 kube-apiserver

```shell
go install sigs.k8s.io/kind@v0.14.0
kind create cluster
kubectl config use-context kind-kind
```

## Lab

### 客户端访问凭证

```shell
cd /tmp
openssl req -nodes -new -x509 -keyout ca.key -out ca.crt # 可随意填写
openssl req -out client.csr -new -newkey rsa:4096 -nodes -keyout client.key -subj "/CN=development/O=system:masters"
openssl x509 -req -days 365 -in client.csr -CA ca.crt -CAkey ca.key -set_serial 01 -out client.crt
openssl pkcs12 -export -in ./client.crt -inkey ./client.key -out client.p12 # 密码设置为 P@ssw0rd
```

### 构建/启动 aa-server

```shell
cd 10_10_sample-apiserver
go mod tidy
go mod vendor
hack/update-codegen.sh
etcd # 启动 Etcd 数据库
GODEBUG=x509sha1=1 go run main.go --secure-port 8443 --etcd-servers http://127.0.0.1:2379   --kubeconfig ~/.kube/config --authentication-kubeconfig ~/.kube/config --authorization-kubeconfig ~/.kube/config --client-ca-file=/tmp/ca.crt # Go 1.18 之后得注明 GODENBUG 参数
```

### 测试

直接通过 URL 调用 aa-server，如果要用 kubectl，还需要配置 kind k8s 集群。

```shell
curl -fv -k --cert-type P12 --cert /tmp/client.p12:P@ssw0rd \
https://127.0.0.1:8443/apis/wardle.example.com/v1alpha1/namespaces/default/flunders
```

