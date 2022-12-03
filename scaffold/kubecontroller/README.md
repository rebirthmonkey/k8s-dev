# kubecontroller脚手架

## Install

- controller-gen：
```shell
go install sigs.k8s.io/controller-tools/cmd/controller-gen@v0.9.2
controller-gen -h 
```

## Development

### 生成 Go Type

- 创建 `xxx_type.go` 文件，并定义 `xxx` 与 `xxxList` 结构体
- 构建 DeepCopy
```shell
controller-gen object paths=./apis/xxx/v1/xxx_types.go
controller-gen object paths=./apis/demo/v1/dummy_types.go
```



## Tmp

manager 中可以包含 1 个或多个 controller。初始化`Controller`调用`ctrl.NewControllerManagedBy`来创建`Builder`，通过 Build 方法完成初始化：

- WithOptions()：填充配置项
- For()：设置 reconcile 处理的资源
- Owns()：设置监听的资源
- Complete()：通过调用 Build() 函数来间接地调用：
  - doController() 函数来初始化了一个 Controller，这里面传入了填充的 Reconciler 以及获取到的 GVK
  - doWatch() 函数主要是监听想要的资源变化，`blder.ctrl.Watch(src, hdler, allPredicates...)` 通过过滤源事件的变化，`allPredicates`是过滤器，只有所有的过滤器都返回 true 时，才会将事件传递给 EventHandler，这里会将 Handler 注册到 Informer 上。