# code-generator

Go 是一门相对简单的语言，但缺乏高层次的抽象，而解决这个的做法就是使用一个外部的代码生成器。在 k8s 早期，越来越多的资源类型被添加，也造成了越来越多的代码需要被重写。有了代码生成器后，维护这部分代码的工作就变得简单了许多。

## 产物

使用 code-generator 可以自动创建 CRD 的 deepcopy、clientset、lister、informer 等所需的代码。在 1.7 版本中，操控 CR 需要基于 [dynamicClient](https://github.com/kubernetes/client-go/tree/master/dynamic)，使用起来并不方便。[code-generator](https://github.com/kubernetes/code-generator) 是 k8s 提供的一个代码生成器项目，可以用来：

1. 在开发 CRD 的 controller 时，生成版本化的、类型化的客户端代码（clientSet）以及 Lister、Informer 代码。
2. 开发 aa-server 时，在内部和版本化的类型、defaulters、protobuf 编解码器、client、informer 之间进行转换。

code-generator 提供的，与 CRD 有关的生成器包括：

1. deepcopy-gen：为每个 T 类型生成 `func (t* T) DeepCopy() *T` 方法，因为 API 类型都需要实现 deepcopy。
2. client-gen：为 CR API 组生成强类型的 clientset。
3. informer-gen：为 CR 生成 Informer。
4. lister-gen：为 CR 生成 Lister，Lister 为 HTTP GET/LIST 请求提供只读缓存层。

Informer 和 Lister 是构建 controller 的基本要素。使用这 4 个代码生成器可以创建全功能的、和 k8s 上游 controller 工作机制相同的 production-ready 的 controller。

code-generator 还包含一些其它的代码生成器，如 conversion-gen 负责产生内外部类型的转换函数、defaulter-gen 负责处理字段默认值。[crd-code-generation ](https://github.com/openshift-evangelists/crd-code-generation) 是使用代码生成器的一个示例项目，可以作为实际项目的起点。但 code-generator 仍需要手动地去创建 type.go 和 CRD 的 yaml 文件。

## Tag

在 Go 的源代码中使用标签来控制代码生成时所使用的属性。

- 全局标签：在 doc.go 文件的 package 行前
- 局部标签：在类型声明前

## Lab

### 定义 Type

在 `pkg/apis/wukong.com/v1/` 目录下定义 doc.go、type.go 和 register.go 3 个文件。

### 自动生成代码

[code-generator](https://github.com/kubernetes/code-generator) 基于 [k8s.io/gengo](https://github.com/kubernetes/gengo) 实现，两者共享一部分命令行标记。大部分的生成器支持 `--input-dirs` 参数来读取一系列输入包，处理其中的每个类型，然后生成代码：

1. 部分代码生成到输入包所在目录，例如 deepcopy-gen，可以使用参数 --output-file-base "zz_generated.deepcopy" 来定义输出文件名。
2. 其它代码生成到 --output-package 指定的目录（通常为 pkg/client），例如 client-gen、informer-gem、lister-gen 等生成器：

```shell
cd 30_code-generator
```

- deepcopy-gen：在 pkg/apis/wukong.com/v1 下创建 zz_generated.deepcopy.go

```shell
deepcopy-gen \
--input-dirs ./pkg/apis/wukong.com/v1 \
-O zz_generated.deepcopy \
30_code-generator/pkg/generated \
30_code-generator/pkg/apis \
wukong.com:v1 \
--output-base ./ \
--go-header-file ./scripts/boilerplate.go.txt 
```

- client-gen：创建 clientset

```shell
client-gen \
--fake-clientset=false \
--clientset-name "versioned" \
--input-base "30_code-generator/pkg/apis" \
--input "wukong.com/v1" \
--output-base ".." \
--output-package "30_code-generator/pkg/generated/clientset" \
-h "scripts/boilerplate.go.txt" 
```

- lister-gen：创建 lister

```shell
lister-gen \
--input-dirs "30_code-generator/pkg/apis/wukong.com/v1" \
--output-base ".." \
--output-package "30_code-generator/pkg/generated/listers" \
-h "scripts/boilerplate.go.txt" 
```

- informer-gen：创建 informer

```shell
informer-gen \
--input-dirs "30_code-generator/pkg/apis/wukong.com/v1" \
--versioned-clientset-package "30_code-generator/pkg/generated/clientset/versioned" \
--output-base ".." \
--output-package "30_code-generator/pkg/generated/informers" \
--listers-package "30_code-generator/pkg/generated/listers" \
-h "scripts/boilerplate.go.txt"
```

### 编写 controller

在 `cmd/code-generator.go` 文件中编写 controller 核心逻辑。

### 运行程序

- 运行程序：通过创建的 clientset、lister、informer 来读取 CR

```shell
go run cmd/code-generator.go
```

- 创建 CRD 以及 CR

```shell
kubectl apply -f manifests/crd.yaml
kubectl get crds
kubectl apply -f manifests/cr.yaml
kubectl get foos
```

### 标准代码

更新后标准的代码在[这里](30_code-generator-bis)





