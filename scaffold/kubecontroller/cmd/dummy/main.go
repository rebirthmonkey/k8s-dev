package main

import (
	"math/rand"
	"time"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager"
	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager/reconcilers/dummy"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	opts := manager.ParseOptionsFromFlags(false)
	manager.Main(opts, nil)
}
