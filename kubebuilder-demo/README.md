# kubebuilder



## Lab

- 初始化 kubebuilder
```shell
kubebuilder init --domain wukong.com --repo github.com/rebirthmonkey/kubebuilder-demo
```

- 创建 API
```shell
kubebuilder create api --group ingress --version v1 --kind App
```

- install CRDS
```shell
make install
```

- 在 controller/Reconcile() 中添加代码
```go
    _ = log.FromContext(ctx)
	fmt.Println("XXXXXXXX app changed", "ns", req.Namespace)
	return ctrl.Result{}, nil
```

- run operator
```shell
make run  
```

- 添加 CR
```shell
kubectl delete -f config/samples/ingress_v1beta1_app.yaml 
```