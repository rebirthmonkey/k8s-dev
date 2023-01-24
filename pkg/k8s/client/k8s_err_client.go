package client

//
//import (
//	"context"
//
//	"k8s.io/apimachinery/pkg/api/meta"
//	"k8s.io/apimachinery/pkg/runtime"
//	"sigs.k8s.io/controller-runtime/pkg/client"
//)
//
//type errClient struct {
//	err    error
//	scheme *runtime.Scheme
//}
//
//type errStatusClient struct {
//	err error
//}
//
//func ToErrClient(cli client.Client) (bool, error) {
//	errC, ok := cli.(*errClient)
//	if ok {
//		return ok, errC.err
//	}
//	return false, nil
//}
//
//var _ client.Client = &errClient{}
//
//func (cli *errClient) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOptions) error {
//	return cli.err
//}
//
//func (cli *errClient) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
//	return cli.err
//}
//
//// Create saves the object obj in the Kubernetes cluster.
//func (cli *errClient) Create(ctx context.Context, obj client.Object, opts ...client.CreateOption) error {
//	return cli.err
//}
//
//// Delete deletes the given obj from Kubernetes cluster.
//func (cli *errClient) Delete(ctx context.Context, obj client.Object, opts ...client.DeleteOption) error {
//	return cli.err
//}
//
//func (cli *errClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
//	return cli.err
//}
//
//func (cli *errClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
//	return cli.err
//}
//
//func (cli *errClient) DeleteAllOf(ctx context.Context, obj client.Object, opts ...client.DeleteAllOfOption) error {
//	return cli.err
//}
//
//func (cli *errClient) Scheme() *runtime.Scheme {
//	return cli.scheme
//}
//
//func (cli *errClient) RESTMapper() meta.RESTMapper {
//	return nil
//}
//
//func (cli *errClient) Status() client.StatusWriter {
//	return &errStatusClient{cli.err}
//}
//
//func (status *errStatusClient) Update(ctx context.Context, obj client.Object, opts ...client.UpdateOption) error {
//	return status.err
//}
//func (status *errStatusClient) Patch(ctx context.Context, obj client.Object, patch client.Patch, opts ...client.PatchOption) error {
//	return status.err
//}
