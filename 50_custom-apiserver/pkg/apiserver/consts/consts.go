package consts

type HeaderInfoKeyType int

const (
	HeaderInfoKey HeaderInfoKeyType = iota

	StorageBackendEtcd = "etcd"
	StorageBackendFile = "file"

	SuperUserUID  = "0"
	SuperUserName = "teleport"

	Kubectl = "kubectl"
)
