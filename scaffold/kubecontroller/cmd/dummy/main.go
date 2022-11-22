package main

import (
	"math/rand"
	"time"

	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/controllers/dummy"
	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/controllers/manager"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	opts := manager.ParseOptionsFromFlags(false)
	manager.Main(opts, nil)
}
