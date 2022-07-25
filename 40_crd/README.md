# CRD

## 简介

在 K8s 中使用的 Deployment、DamenSet、StatefulSet、Service、Ingress、ConfigMap、Secret 这些都是资源。而对这些资源的创建、更新、删除的动作都会被称为为事件(Event)，K8s 的 Controller Manager 负责事件监听，并触发相应的动作来满足期望（Spec）。这种方式也就是声明式，即用户只需要关心应用程序的最终状态。当在使用中发现现有的这些资源不能满足需求时，K8s 提供了自定义资源（Custom Resource）和 opertor 为应用程序提供基于 K8s 扩展。CRD与其他k8s的核心资源放在同一个ETCD实例中，并且由同一个k8s-apiserver提供服务。

CRD（Custom Resource Definition） 是对自定义资源的描述，也就是介绍这个资源有什么属性呀，这些属性的类型是什么，结构是怎样的这类。例如 postgres-operator 的 CRD：

```yaml
apiVersion: apiextensions.k8s.io/v1beta1
kind: CustomResourceDefinition
metadata:
  name: postgresqls.acid.zalan.do
  labels:
    app.kubernetes.io/name: postgres-operator
  annotations:
    "helm.sh/hook": crd-install
spec:
  group: acid.zalan.do
  names:
    kind: postgresql
    listKind: postgresqlList
    plural: postgresqls
    singular: postgresql
    shortNames:
    - pg  additionalPrinterColumns:
  - name: Team
    type: string
    description: Team responsible for Postgres CLuster
    JSONPath: .spec.teamId
  - name: Version
    type: string
    description: PostgreSQL version
    JSONPath: .spec.postgresql.version
  - name: Pods
    type: integer
    description: Number of Pods per Postgres cluster
    JSONPath: .spec.numberOfInstances
  - name: Volume
    type: string
    description: Size of the bound volume
    JSONPath: .spec.volume.size
...
```

上面的 CRD 可以看到它主要包括 apiVersion、kind、metadata 和 spec 四个部分，其中最关键的是 apiVersion 和kind。apiVersion 表示资源所属组织和版本，apiVersion 一般由 APIGourp 和 Version 组成，这里 APIGourp 是 [http://apiextensions.k8s.io](https://link.zhihu.com/?target=http%3A//apiextensions.k8s.io)，Version 是v1beta1，相关信息可以通过 kubectl api-resoures 查看。kind 表示资源类型，这里是 CustomResourceDefinition，表示是一个自定义的资源描述。

- 查看所有资源：包括内建及自定义
```shell
kubectl api-resources 
```

### Properties

- metadata.name：该资源的 ID
- spec.name：只是一个名为 name 的属性

### 架构

<img src="figures/image-20220725092436287.png" alt="image-20220725092436287" style="zoom:50%;" />

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

