package etcd

import (
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

type shortNamesProvider struct {
	*registry.Store
	shortNames []string
}

func (s *shortNamesProvider) ShortNames() []string {
	return s.shortNames
}

func WithShortNames(store *registry.Store, shortNames []string) rest.StandardStorage {
	return &shortNamesProvider{
		Store:      store,
		shortNames: shortNames,
	}
}
