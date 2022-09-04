# CRD&Operator

## 简介

“资源”对应着Kubernetes API中的一个端点（Endpoint），它存储了某种类型的API对象。自定义资源（Custom Resources）是对Kubernetes API的扩展，代表某种自定义的配置或独立运行的服务。和内置资源一样，自定义资源本身仅仅是一段结构化数据，仅仅和相应自定义控制器联用后，才能作为声明式API。自定义资源描述了你期望的资源状态，由控制器来尽力达到此状态。自定义控制器由用户部署到集群，这种控制器独立于集群本身的生命周期。尽管自定义控制器可以和任何类型的资源配合，但是对于自定义资源特别有意义。CoreOS提出的Operator Framework，就是自定义控制器联用自定义资源的例子。

在 K8s 中使用的 Deployment、DamenSet、StatefulSet、Service、Ingress、ConfigMap、Secret 这些都是资源。而对这些资源的创建、更新、删除的动作都会被称为为事件(Event)，K8s 的 Controller Manager 负责事件监听，并触发相应的动作来满足期望（Spec）。这种方式也就是声明式，即用户只需要关心应用程序的最终状态。当在使用中发现现有的这些资源不能满足需求时，K8s 提供了自定义资源（Custom Resource）和 opertor 为应用程序提供基于 K8s 扩展。CRD与其他k8s的核心资源放在同一个ETCD实例中，并且由同一个k8s-apiserver提供服务。

### CRD

CRD是一种API资源，利用它你可以定义“自定义资源”，K8S负责CRD的存储。使用CRD而非API聚合可以免去编写次级API Server的烦恼，但是其灵活性不如API聚合。CRD从1.7版本开始引入，到1.8版本进入Beta状态。最新的1.11版本CRD获得增强，支持scale、status子资源。CRD/CR仅仅是一段声明信息，必须配合相应的控制器才有价值。

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

Properties

- metadata.name：该资源的 ID
- spec.name：只是一个名为 name 的属性

#### CR

一旦CRD创建成功，你就可以创建对应类型的自定义资源（Custom Resource，CR）了。自定义资源可以包含任意的自定义字段，例如：

```yaml
apiVersion: "k8s.gmem.cc/v1"
kind: CronTab
metadata:
  name: cron
  # 删除前钩子
  finalizers:
  # 下面的形式非法
  - finalizer.k8s.gmem.cc
  # 必须这样
  - finalizer.k8s.gmem.cc/xxx
spec:
  cronSpec: "* * * * */5"
  image: crontab:1.0.0
```

### Operator

Operator 首先是一个 controller，通过扩展 k8s API 来创建、配置和管理一个复杂、有状态的应用。它一般基于 k8s 的资源和 controller 相关概念开发，但还包含了很多业务领域或应用相关的逻辑，实现了一些自动化操作代替用户的手工操作。这些自动化运维操作往往是一些最佳实践，例如动态扩容等。

Operator 的发布一般包括：

- CRD + CR：CRD 用于定义领域相关的 schema，与之对应的 CR 用于描述实例级别的领域相关信息。
- Controller：用来管理 CR，同时也会涉及到一些核心资源。

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

