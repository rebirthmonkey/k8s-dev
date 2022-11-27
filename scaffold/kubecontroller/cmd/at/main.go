package main

import (
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager"
	"math/rand"
	"time"

	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager/reconcilers/at"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	manager.NewApp("kubecontroller").Run()
	//opts := manager.ParseOptionsFromFlags(false)
	//manager.Main(opts, nil)
}
