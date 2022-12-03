package main

import (
	"math/rand"
	"time"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/manager"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	manager.NewApp("kubecontroller").Run()
}
