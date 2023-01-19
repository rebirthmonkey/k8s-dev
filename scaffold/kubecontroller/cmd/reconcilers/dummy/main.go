package main

import (
	"math/rand"
	"time"

	"github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/reconcilerapp"
	_ "github.com/rebirthmonkey/k8s-dev/scaffold/kubecontroller/pkg/reconcilers/dummy"
)

func main() {
	rand.Seed(time.Now().UnixNano())

	reconcilerapp.NewApp("kubecontroller").Run()
}