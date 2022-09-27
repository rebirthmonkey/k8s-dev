# CRD&Operator

## CRD

“资源”对应着 k8s API中的一个 Endpoint，它存储了某种类型的 API 对象。在 k8s 中使用的 Deployment、DamenSet、StatefulSet、Service、Ingress、ConfigMap、Secret 这些都是资源。而对这些资源的创建、更新、删除的动作都会被称为为事件（Event），k8s 的 Controller Manager 负责事件监听，并触发相应的动作来满足期望（Spec）。这种方式也就是声明式，即用户只需要关心应用程序的最终状态。当在使用中发现现有的这些资源不能满足需求时，k8s 提供了自定义资源 CR（Custom Resource）和 controller 为应用程序提供基于 k8s 扩展。CR 与其他 k8s 的核心资源放在同一个 Etcd 中，并且由同一个 k8s-apiserver 提供服务。

自定义资源 CR（Custom Resources）是对 k8s API 的扩展，代表某种自定义的配置或独立运行的服务。和内置资源一样，CR 本身仅仅是一段结构化数据，仅仅和相应自定义 controller 联用后，才能作为声明式 API。CR 描述了期望的资源状态，由 controller 来尽力达到此状态。自定义 controller 由用户部署到集群，这种 controller 独立于集群本身的生命周期。尽管自定义c controller 可以和任何类型的资源配合，但是对于 CR 特别有意义。CoreOS 提出的 Operator Framework，就是自定义 controller 联用 CR 的例子。

### CRD

CRD（Custom Resource Definition）是一种 API 资源，它实际上是自定义资源的 Schema、利用它可以定义 CR，而 k8s 负责 CRD 的存储。使用 CRD 而非 aa-server 可以免去编写次级 API Server 的烦恼，但是其灵活性不如 aa-server。

CRD 是对自定义资源的描述，也就是介绍这个资源有什么属性呀，这些属性的类型是什么，结构是怎样的这类。在没有对应 controller 的情况下，它仅仅用于少量配置信息的保存，与 k8s 原生的资源放在同一个 Etcd 中。CRD/CR 仅仅是一段声明信息，如果需要发挥更大作用，必须配合相应的 controller 才有价值。

下面的 CRD 例子可以看到它主要包括 apiVersion、kind、metadata 和 spec 四个部分，其中最关键的是 apiVersion 和 kind。apiVersion 表示资源所属组织和版本，apiVersion 一般由 APIGourp 和 Version 组成，这里 APIGourp 是 [http://apiextensions.k8s.io](https://link.zhihu.com/?target=http%3A//apiextensions.k8s.io)，Version 是v1beta1，相关信息可以通过 kubectl api-resoures 查看。kind 表示资源类型，这里是 CustomResourceDefinition，表示是一个自定义的资源描述。

- 查看所有资源：包括内建及自定义

```shell
kubectl api-resources 
```

Properties

- metadata.name：该资源的 ID
- spec.name：只是一个名为 name 的属性

例如 postgres-operator 的 CRD：

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

### CR

一旦 CRD 创建成功，就可以创建对应类型的 CR（Custom Resource）了。自定义资源可以包含任意的自定义字段，例如：

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

### apisextensions-apiserver

创建 CRD 对象后，kube-apiserver 的 apiextensions-apiserver 会检查其名字，确认是否与其他资源的名字发生冲突以及它内部的定义是否一致。

<img src="figures/image-20220910121231143.png" alt="image-20220910121231143" style="zoom:50%;" />

### 子资源

子资源是特殊的 HTTP endpoint，通过在普通资源的 HTTP 路径后加上后缀得到，如在 Pod 的 /api/v1/namespaces/namespace/pods/name 后加上 /logs、/portforward、/exec、/status 等。当前 k8s 的 CR 支持 2 种子资源：/scale 和 /status。

#### /status

/status 子资源用于吧用户的 CR 实例的展示和管控分离，实现权限隔离：

- 用户不需要更新状态字段
- controller 不需要更新资源规格字段

/status 将资源与展示分开，提供了不同的 endpoint，每个 endpoint 由独立的 RBAC 规则来管理。

#### /scale

/scale 用于查看和修改资源中制定的副本数，用于类似 Deployment 或 ReplicSet 这样的具有副本数的资源，通过 /scale 可以进行资源的扩容和缩容。它往往结合 kubectl 的 scale 命令配合使用，如

```shell
kubectl scale --replicas=3 atxxx -v=7
```



## Operator

Operator 是一种 K8s 的扩展形式，利用 CR 来管理应用和组件，允许用户以 K8s 的声明式 API 风格来管理应用及服务，支持 kubectl 命令行。同时 Operator 开发者可以像使用原生 API 进行应用管理一样，通过声明式的方式定义一组业务应用的期望终态，并且根据业务应用的自身特点进行相应控制器逻辑编写，以此完成对应用运行时刻生命周期的管理并持续维护与期望终态的一致性。这样的设计范式使得应用部署者只需要专注于配置自身应用的期望状态，而无需再投入大量的精力在手工部署或是业务在运行时刻的繁琐运维操作中。

在 K8s 的实现思想中，会使用 Controller 模式对 Etcd 里的 API 资源对象变化保持不断的监听（Watch），并在 Controller 中对指定事件进行响应处理，针对不同的 API 资源可以在对应的控制器中添加相应的业务逻辑，通过这种方式完成应用编排中各阶段的事件处理。而 Operator 正是基于 controller 模式，允许应用开发者通过扩展 k8s API 资源的方式，将复杂的分布式应用集群抽象为一个自定义的 API 资源，通过对自定义 API 资源的请求可以实现基本的运维操作，而在 Operator 中开发者可以专注实现应用在运行时管理中遇到的相关复杂逻辑。Operator 定义了一组在 k8s 集群中打包和部署复杂业务应用的方法，Operator 主要是为解决特定应用或服务关于如何部署、运行及出现问题时如何处理提供的一种特定的自定义方式。比如：

- 按需部署应用服务（总不能用一大堆 configmap 来管理吧）
- 实现应用状态的备份和还原，完成版本升级
- 为分布式应用进行 master 选举，例如 etcd

Operator 首先是一个 controller，通过扩展 k8s API 来创建、配置和管理一个复杂、有状态的应用。它一般基于 k8s 的资源和 controller 相关概念开发，但还包含了很多业务领域或应用相关的逻辑，实现了一些自动化操作代替用户的手工操作。这些自动化运维操作往往是一些最佳实践，例如动态扩容等。Operator 定义了一组在 k8s 集群中打包和部署复杂业务应用的方法，它可以方便地在不同集群中部署并在不同的客户间传播共享。同时 Operator 还提供了一套应用在运行时的监控管理方法，领域专家通过将业务关联的运维逻辑写入到 operator  自身控制器中，而一个运行中的 Operator 就像一个 7*24 不间断工作的优秀运维团队，可以时刻监控应用自身状态和该应用在 k8s 集群中的关注事件，并在毫秒级别基于期望终态做出对监听事件的处理，比如对应用的自动化容灾响应或是滚动升级等高级运维操作。

Operator 的发布一般包括：

- CRD + CR：CRD 用于定义领域相关的 schema，与之对应的 CR 用于描述实例级别的领域相关信息。
- Controller：用来管理 CR，同时也会涉及到一些核心资源。

### 历史

CoreOS 在 2016 年底提出了 Operator 的概念，当时的一段官方定义如下：“An Operator represents human operational knowledge in software, to reliably manage an application.”

谷歌率先提出了 TPR（Third Party Resource）的概念，允许开发者根据业务需求以插件化形式扩展出相应的 K8s API 对象模型，同时提出了自定义 controller 的概念用于编写面向领域知识的业务控制逻辑，基于 Third Party Resource，K8s 社区在 1.7 版本中提出了[custom resources and controllers](https://mp.weixin.qq.com/s?__biz=MzUzNzYxNjAzMg==&mid=2247493984&idx=1&sn=59238e3a71d14702191e5c518b749836&chksm=fae6e2afcd916bb92a15170911643357ac56fa72ae9bf844b500fbd5e5253fd741f5af0d797c&token=10947129&lang=zh_CN&scene=21#wechat_redirect) 的概念，这正是 Operator 的核心概念。

CoreOS 是最早的一批基于 k8s 提供企业级容器服务解决方案的厂商之一，他们很敏锐地捕捉到了 TPR  和控制器模式对企业级应用开发者的重要价值。并很快由邓洪超等人基于 TPR 实现了历史上第一个 Operator：etcd-operator。它可以让用户通过短短的几条命令就快速的部署一个 Etcd 集群，并且基于 kubectl  命令行一个普通的开发者就可以实现 etcd 集群滚动更新、灾备、备份恢复等复杂的运维操作，极大的降低了 etcd 集群的使用门槛，在很短的时间就成为当时 K8s 社区关注的焦点项目。

而 Operator 的出现打破了社区传统意义上的格局，对于谷歌团队而言，Controller 作为 k8s 原生 API  的核心机制，应该交由系统内部的 Controller Manager 组件进行管理，并且遵从统一的设计开发模式，而不是像 Operator 那样交由应用开发者自由地进行 Controller 代码的编写。另外 Operator 作为 K8s 生态系统中与终端用户建立连接的桥梁，作为 K8s 项目的设计和捐赠者，谷歌当然也不希望错失其中的主导权。同时 Brendan Burns 突然宣布加盟微软的消息，也进一步加剧了谷歌团队与 Operator 项目之间的矛盾。于是，2017 年开始谷歌和 RedHat 开始在社区推广 Aggregated Apiserver，应用开发者需要按标准的社区规范编写一个自定义的  Apiserver，同时定义自身应用的 API 模型。通过原生 Apiserver 的配置修改，扩展 Apiserver 会随着原生组件一同部署，并且限制自定义 API 在系统管理组件下进行统一管理。之后，谷歌和 RedHat 开始在社区大力推广使用聚合层扩展  K8s API，同时建议废弃 TPR 相关功能。

然而，巨大的压力并没有让 Operator 昙花一现，就此消失。相反，社区大量的 Operator 开发和使用者仍旧拥护着 Operator 清晰自由的设计理念，继续维护演进着自己的应用项目。同时很多云服务提供商也并没有放弃 Operator，Operator  简洁的部署方式和易复制，自由开放的代码实现方式使其维护住了大量忠实粉丝。在用户的选择面前，强如谷歌、红帽这样的巨头也不得不做出退让。最终，TPR 并没有被彻底废弃，而是由 CRD（Custom Resource Definition）资源模型范式代替。2018 年初，RedHat 完成了对 CoreOS 的收购，并在几个月后发布了 Operator Framework，通过提供 SDK 等管理工具的方式进一步降低了应用开发与 K8s 底层 API 知识体系之间的依赖。至此，Operator 进一步巩固了其在 K8s 应用开发领域的重要地位。【1】

### 生命周期

- 开发者使用 Operator SDK 创建一个 Operator 项目；
- SDK 生成 Operator 对应的脚手架代码，然后扩展相应业务模型和 API，实现业务逻辑完成 Operator 的代码编写；
- 参考社区[测试指南](https://mp.weixin.qq.com/s?__biz=MzUzNzYxNjAzMg==&mid=2247493984&idx=1&sn=59238e3a71d14702191e5c518b749836&chksm=fae6e2afcd916bb92a15170911643357ac56fa72ae9bf844b500fbd5e5253fd741f5af0d797c&token=10947129&lang=zh_CN&scene=21#wechat_redirect)进行业务逻辑的本地测试以及打包和发布格式的本地校验；
- 在完成测试后可以根据[规定格式](https://mp.weixin.qq.com/s?__biz=MzUzNzYxNjAzMg==&mid=2247493984&idx=1&sn=59238e3a71d14702191e5c518b749836&chksm=fae6e2afcd916bb92a15170911643357ac56fa72ae9bf844b500fbd5e5253fd741f5af0d797c&token=10947129&lang=zh_CN&scene=21#wechat_redirect)向社区提交[PR](https://mp.weixin.qq.com/s?__biz=MzUzNzYxNjAzMg==&mid=2247493984&idx=1&sn=59238e3a71d14702191e5c518b749836&chksm=fae6e2afcd916bb92a15170911643357ac56fa72ae9bf844b500fbd5e5253fd741f5af0d797c&token=10947129&lang=zh_CN&scene=21#wechat_redirect)，会有专人进行 review；
- 待社区审核通过完成 merge 后，终端用户就可以在 OperatorHub.io 页面上找到业务对应的 Operator 的说明文档和安装指南，通过简单的命令行操作即可在目标集群上完成 Operator 实例的安装；
- Operator 实例会根据配置创建所需的业务应用，OLM 和 Operator Metering 等组件可以帮助用户完成业务应用对应的运维和监控采集等管理操作。

<img src="figures/image-20220905093941148.png" alt="image-20220905093941148" style="zoom:33%;" />

## Lab

### CRD

- Create CRD
```shell
kubectl get crds
kubectl apply -f 10_crd/crd.yaml
kubectl get crds
```

- Create CR
```shell
kubectl apply -f 10_crd/cr.yaml
kubectl get crd1s 
```

### code-generator

- [code-generator](30_code-generator/README.md)

### 基于sample controller的App

- [基于 sample controller 的 App](50_app/README.md)

### app-controller

- [基于 sample controller 实现 App](55_app-controller/README.md)


## Ref

1. [kubebuilder Book](https://book.kubebuilder.io/introduction.html)
2. [扩展K8S](https://blog.gmem.cc/crd)
3. 



