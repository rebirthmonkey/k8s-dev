package pm

import (
	"context"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"k8s.io/apimachinery/pkg/types"
)

func ParseNamespacedName(key string) types.NamespacedName {
	ka := strings.Split(key, "/")
	var namespace, name string
	if len(ka) == 2 {
		namespace = ka[0]
		name = ka[1]
	} else {
		namespace = "default"
		name = key
	}
	return types.NamespacedName{
		Namespace: namespace,
		Name:      name,
	}
}

func UpdateSpec(ctx context.Context, cli client.Client, obj client.Object) error {
	if err := cli.Update(ctx, obj); err != nil {
		return err
	}
	return cli.Get(ctx, client.ObjectKey{Namespace: obj.GetNamespace(), Name: obj.GetName()}, obj)
}

func Now() *metav1.Time {
	now := metav1.Now()
	return &now
}
