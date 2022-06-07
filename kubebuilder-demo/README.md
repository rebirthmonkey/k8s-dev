# kubebuilder


## Lab

- 初始化 go mod
```shell
go mod init github.com/rebirthmonkey/kubebuilder-demo
go mod tidy 
```

- 初始化 kubebuilder
```shell
kubebuilder init --domain wukong.com
```

- 创建 API
```shell
kubebuilder create api --group ingress --version v1beta1 --kind App
```