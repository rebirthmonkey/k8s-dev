# Operator Framework

## 简介

[Operator Framework](https://www.zhihu.com/search?q=operator+framework&search_source=Entity&hybrid_search_source=Entity&hybrid_search_extra={"sourceType"%3A"article"%2C"sourceId"%3A246550722}) SDK 是 CoreOS 公司开发和维护的用于快速创建 Operator 的工具。

### 原理

<img src="figures/image-20211210085908961.png" alt="image-20211210085908961" style="zoom:50%;" />

### 安装 Operator-SDK

```shell
export ARCH=$(case $(uname -m) in x86_64) echo -n amd64 ;; aarch64) echo -n arm64 ;; *) echo -n $(uname -m) ;; esac)

export OS=$(uname | awk '{print tolower($0)}')

export OPERATOR_SDK_DL_URL=https://github.com/operator-framework/operator-sdk/releases/download/v1.15.0

curl -LO ${OPERATOR_SDK_DL_URL}/operator-sdk_${OS}_${ARCH}
gpg --keyserver keyserver.ubuntu.com --recv-keys 052996E2A20B5C7E

curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt

curl -LO ${OPERATOR_SDK_DL_URL}/checksums.txt.asc
gpg -u "Operator SDK (release) <cncf-operator-sdk@cncf.io>" --verify checksums.txt.asc

grep operator-sdk_${OS}_${ARCH} checksums.txt | sha256sum -c -

chmod +x operator-sdk_${OS}_${ARCH} 

sudo mv operator-sdk_${OS}_${ARCH} /usr/local/bin/operator-sdk

operator-sdk --help # 确认安装成功
```

### Memcached 流程

#### 创建项目

```shell
mkdir memcached-operator

cd memcached-operator

operator-sdk init --domain example.com --repo github.com/example/memcached-operator
```

创建项目结构目录如下：

```shell
tree
├── Dockerfile
├── Makefile
├── PROJECT
├── config
│   ├── default
│   │   ├── kustomization.yaml
│   │   ├── manager_auth_proxy_patch.yaml
│   │   └── manager_config_patch.yaml
│   ├── manager
│   │   ├── controller_manager_config.yaml
│   │   ├── kustomization.yaml
│   │   └── manager.yaml
│   ├── manifests
│   │   └── kustomization.yaml
│   ├── prometheus
│   │   ├── kustomization.yaml
│   │   └── monitor.yaml
│   ├── rbac
│   │   ├── auth_proxy_client_clusterrole.yaml
│   │   ├── auth_proxy_role.yaml
│   │   ├── auth_proxy_role_binding.yaml
│   │   ├── auth_proxy_service.yaml
│   │   ├── kustomization.yaml
│   │   ├── leader_election_role.yaml
│   │   ├── leader_election_role_binding.yaml
│   │   ├── role_binding.yaml
│   │   └── service_account.yaml
│   └── scorecard
│       ├── bases
│       │   └── config.yaml
│       ├── kustomization.yaml
│       └── patches
│           ├── basic.config.yaml
│           └── olm.config.yaml
├── go.mod
├── go.sum
├── hack
│   └── boilerplate.go.txt
└── main.go
```

#### 创建 CRD

```shell
operator-sdk create api --group cache --version v1alpha1 --kind Memcached --resource --controller
```

CRD: api/v1alpha1/memcached_types.go

- Size:
- Nodes:

```go
// MemcachedSpec defines the desired state of Memcached
type MemcachedSpec struct {
    //+kubebuilder:validation:Minimum=0
    // Size is the size of the memcached deployment
    Size int32 `json:"size"`
}

// MemcachedStatus defines the observed state of Memcached
type MemcachedStatus struct {
    // Nodes are the names of the memcached pods
    Nodes []string `json:"nodes"`
}
```

- Maker: //+kubebuilder:subresource:status

```go
// Memcached is the Schema for the memcacheds API
//+kubebuilder:subresource:status
type Memcached struct {
    metav1.TypeMeta   `json:",inline"`
    metav1.ObjectMeta `json:"metadata,omitempty"`
    Spec   MemcachedSpec   `json:"spec,omitempty"`
    Status MemcachedStatus `json:"status,omitempty"`
}
```

- Generate：更新 api/v1alpha1/zz_generated.deepcopy.go

```shell
make generate
```

- Manifest：生成 CRD 的 YAML 文件config/crd/bases/cache.example.com_memcacheds.yaml

```go
make manifests
```

#### 编写 Controller

创建好 CRD 后就需要编写 controllers/memcached_controller.go

##### SetupWithManager()

- 初始化
- 观察对应资源的变化，给 Reconsile() 发送 Request

##### Reconcile() Loop

- 声明式 API 的实现，确保 actual state 向 desired state发展

- RBAC marker：确保 k8s 上的 objects 可以被使用

```Go
//+kubebuilder:rbac:groups=cache.example.com,resources=memcacheds,verbs=get;list;watch;create;update;patch;delete
```

##### Manifests

```shell
make manifests
```

#### 配置镜像

- Makefile

```makefile
IMAGE_TAG_BASE ?= wukongsun
IMG ?= wukongsun/memcached-operator:v0.0.1
```

- build & push

```shell
make docker-build docker-push
```

#### 运行 Operator

- As Deployment

```shell
make deploy
$ kubectl get deployment -n memcached-operator-system
```

- As OLM：OLM 无法安装

```shell
operator-sdk olm install # 安装 OLM
make bundle bundle-build bundle-push
operator-sdk run bundle wukongsun/memcached-operator-bundle:v0.0.1
```

#### 使用 Operator

- config/samples/cache_v1alpha1_memcached.yaml

```yaml
apiVersion: cache.example.com/v1alpha1
kind: Memcached
metadata:
  name: memcached-sample
spec:
  size: 3
```

- create resource

```shell
kubectl apply -f config/samples/cache_v1alpha1_memcached.yaml
kubectl get deployment
kubectl get pods
kubectl get memcached/memcached-sample -o yaml
```

- edit resource

```yaml
kubectl patch memcached memcached-sample -p '{"spec":{"size": 5}}' --type=merge
kubectl get deployment
```

#### Cleanup

```shell
kubectl delete -f config/samples/cache_v1alpha1_memcached.yaml
make undeploy
```

## Ref

1. [云原生应用实现规范 - 初识 Operator](https://mp.weixin.qq.com/s/MveSspUcFFWSum1m_XtRlg)
2. [十分钟弄懂 k8s Operator 应用的制作流程](https://zhuanlan.zhihu.com/p/246550722)
