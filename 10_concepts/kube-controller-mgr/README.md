# kube-controller-manager

## 简介

kube-controller-manager 具有高可用，它基于 Etcd 集群的分布式锁实现 leader 选举机制。抢先获取锁的实例被称为 leader，并运行 kube-controller-manager 组件的主逻辑。而未获得锁的实例被称为 candidate，运行时处于阻塞状态。在 leader 节点因为某些原因推出后，candidate 则通过 leader 选举参与竞选，称为 leader 节点后阶梯 kube-controller-manager 的工作。

- Replication Controller：RC 所关联的 pod 副本数保持预设值，pod的RestartPolicy=Always
- Node Controller：kubelet通过API server注册自身节点信息
- ResourceQuota Controller：确保指定资源对象在任何时候都不会超量占用系统物力资源（需要Admission Control配合使用）
- Endpoint Controller：生成和维护同名server的所有endpoint（所有对应pod的service）
- Service Controller：监听、维护service的变化
- Namespace Controller
- ServiceAccount Controller
- Token Controller

