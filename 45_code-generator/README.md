# code-generator

使用 code-generator 来创建 CRD 的 deepcopy、clientset、lister、informer


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
