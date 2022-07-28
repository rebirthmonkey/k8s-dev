# code-generator

当需要具体操作一个CRD资源时，需要为该资源配备deepcopy、clientset、lister、informer代码。使用 code-generator 可以自动创建 CRD 的 deepcopy、clientset、lister、informer等所需的代码。在1.7-版本中，操控 CR 需要基于[client-go dynamic client](https://github.com/kubernetes/client-go/tree/master/dynamic)，使用起来并不方便。[code-generator](https://github.com/kubernetes/code-generator) 是K8S提供的一个代码生成器项目，可以用来：

1. 开发CRD的控制器时，生成版本化的、类型化的客户端代码（clientset），以及Lister、Informer代码
2. 开发API聚合时，在内部和版本化的类型、defaulters、protobuf编解码器、client、informer之间进行转换

code-generator 提供的，和 CRD 有关的生成器包括：

1. deepcopy-gen：为每个T类型生成 func (t* T) DeepCopy() *T方法。API类型都需要实现深拷贝
2. client-gen：为CustomResource API组生成强类型的clientset
3. informer-gen：为CustomResources生成Informer
4. lister-gen：为CustomResources生成Lister，Lister为GET/LIST请求提供只读缓存层

Informer 和 Lister 是构建控制器的基本要素。使用这4个代码生成器可以创建全功能的、和K8S上游控制器工作机制相同的production-ready的控制器。

code-generator还包含一些其它的代码生成器。例如Conversion-gen负责产生内外部类型的转换函数、Defaulter-gen负责处理字段默认值。

[crd-code-generation](https://github.com/openshift-evangelists/crd-code-generation)是使用代码生成器的一个示例项目，可以作为你的实际项目的起点。

但仍需要手动地去创建 type.go 和 CRD 的 yaml 文件。


## Lab

- deepcopy-gen：创建 deepcopy
```shell
deepcopy-gen \
--input-dirs ./pkg/apis/wukong.com/v1 \
-O zz_generated.deepcopy \
45_code-generator/pgk/generated \
45_code-generator/pkg/apis \
wukong.com:v1 --output-base ./ --go-header-file ./hack/boilerplate.go.txt 
```

- client-gen：创建 clientset
```shell
client-gen \
--fake-clientset=false \
--clientset-name "versioned" \
--input-base "45_code-generator/pkg/apis" \
--input "wukong.com/v1" \
--output-base ".." \
--output-package "45_code-generator/pkg/generated/clientset" \
-h "hack/boilerplate.go.txt" 
```

- lister-gen：创建 lister
```shell
lister-gen \
--input-dirs "45_code-generator/pkg/apis/wukong.com/v1" \
--output-base ".." \
--output-package "45_code-generator/pkg/generated/listers" \
-h "hack/boilerplate.go.txt" 
```

- informer-gen：创建 informer
```shell
informer-gen \
--input-dirs "45_code-generator/pkg/apis/wukong.com/v1" \
--versioned-clientset-package "45_code-generator/pkg/generated/clientset/versioned" \
--output-base ".." \
--output-package "45_code-generator/pkg/generated/informers" \
--listers-package "45_code-generator/pkg/generated/listers" \
-h "hack/boilerplate.go.txt"
```

- 创建 CRD 以及 CR
```shell
kubectl apply -f manifest/crd.yaml
kubectl apply -f manifest/cr.yaml
```

- 运行程序：通过创建的 clientset、lister、informer 来读取 CR
```shell
go run main.go
```

## 标准代码

更新后标准的代码在[这里](../45_code-generator-bis)
