package apis

import (
	"fmt"

	"github.com/thoas/go-funk"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/rebirthmonkey/k8s-dev/pkg/utils"
)

func RemoveFinalizer(obj client.Object, finalizer string) {
	of := obj.GetFinalizers()
	nf := []string{}
	for _, item := range of {
		if item == finalizer {
			continue
		}
		nf = append(nf, item)
	}
	obj.SetFinalizers(nf)
}

func AddFinalizer(obj client.Object, finalizers ...string) {
	ofs := obj.GetFinalizers()
	for _, f := range finalizers {
		if !funk.ContainsString(ofs, f) {
			ofs = append(ofs, f)
		}
	}
	obj.SetFinalizers(ofs)
}

func EqualFinalizers(obj client.Object, finalizers []string) bool {
	return utils.ArrayEqual(obj.GetFinalizers(), finalizers)
}

func GetFinalizers(obj client.Object) (res []string) {
	fs := obj.GetFinalizers()
	res = make([]string, len(fs))
	for i, f := range fs {
		res[i] = f
	}
	return
}

func Finalizer(ty, id string) string {
	return fmt.Sprintf("cloud.tencent.com/%s/%s", ty, id)
}
