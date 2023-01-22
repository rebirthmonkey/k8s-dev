package kube

import (
	pm2 "cloud.tencent.com/teleport/pkg/pm"
	"context"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/util/retry"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

func FetchResourceFunc(clientKey string, obj client.Object) func(ctx context.Context, key string) (pm2.Resource, error) {
	return func(ctx context.Context, key string) (pm2.Resource, error) {
		o := obj.DeepCopyObject().(client.Object)
		cli := ctx.Value(clientKey).(client.Client)
		err := cli.Get(ctx, pm2.ParseNamespacedName(key), o)
		return o, err
	}
}

func PersistStatusFunc(clientKey string) func(ctx context.Context, res pm2.Resource) error {
	return func(ctx context.Context, res pm2.Resource) error {
		cli := ctx.Value(clientKey).(client.Client)
		obj := res.(client.Object)
		r := 0
		return retry.RetryOnConflict(retry.DefaultRetry, func() error {
			if r > 0 {
				n := obj.DeepCopyObject().(client.Object)
				if err := cli.Get(context.Background(), types.NamespacedName{
					Namespace: obj.GetNamespace(),
					Name:      obj.GetName(),
				}, n); err == nil {
					obj.SetResourceVersion(n.GetResourceVersion())
				}
			}
			r++
			return cli.Status().Update(ctx, obj)
		})
	}
}
