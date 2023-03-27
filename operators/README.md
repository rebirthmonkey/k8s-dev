# Operators

## 最佳实践

### YAML

- YAML数据读取：在定义完 `api/v1/workflow_types.go` 中的数据结构后，通过 `make generate` 自动生成如 `deepcopy` 等函数后，通过 `make manifests` 自动生成 CRD 和 CR 的 YAML 文件，后续在 sample 的 YAML 文件中添加相关键值队后，Operator 会自动加载到该 CR 内部的 status、spec 数据结构中。

### Reconcile()

- 共享信息：Reconcile() 是“无状态”的，因此需要将所有需要多个 Reconcile() 之间共享的信息存在 Reconciler 结构体中。
- 可以在 Reconcile() 中设置多个 Phase，使 Reconcile() 每次根据 status.Phase 进入不用的 Phase。 
- 在 Reconcile() 中修改 status 的值后，需要通过 `r.Status().Update(ctx, wf)` l来更新到 k8s 集群中 CR 的真实状态。
- 如果需要 Reconcile() 重复循环地执行，需要 `return ctrl.Result{Requeue: true}, nil`。

## Workflow

[代码见此](workflow/README.md)