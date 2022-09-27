package store

import (
	"50_custom-apiserver/pkg/apiserver/client"
	//"50_custom-apiserver/pkg/apiserver/store/file"
	"50_custom-apiserver/pkg/apiserver/strategy"
	"k8s.io/apiserver/pkg/registry/generic/registry"
	"k8s.io/apiserver/pkg/registry/rest"
)

// PartialUpdateStore decorates a parent storage and only updates
// part of the resource, such as subresource, when updating
func PartialUpdateStore(parentStore rest.StandardStorage, prepareForUpdateFunc strategy.PrepareForUpdateFunc,
	validateUpdateFunc strategy.ValidateUpdateFunc, canonicalizeFunc strategy.CanonicalizeFunc,
	clientFactory client.KubeClientFactory) rest.Storage {
	switch pstor := parentStore.(type) {
	case *registry.Store:
		pstor.UpdateStrategy = &strategy.PartialUpdateStrategy{
			RESTUpdateStrategy:   pstor.UpdateStrategy,
			PrepareForUpdateFunc: prepareForUpdateFunc,
			ValidateUpdateFunc:   validateUpdateFunc,
			CanonicalizeFunc:     canonicalizeFunc,
			ClientFactory:        clientFactory,
		}
		//case *file.Store:
		//	pstor.UpdateStrategy = &strategy.PartialUpdateStrategy{
		//		RESTUpdateStrategy:   pstor.UpdateStrategy,
		//		PrepareForUpdateFunc: prepareForUpdateFunc,
		//		ValidateUpdateFunc:   validateUpdateFunc,
		//		CanonicalizeFunc:     canonicalizeFunc,
		//		ClientFactory:        clientFactory,
		//	}
	}
	return &partialUpdateStore{
		StandardStorage: parentStore,
	}
}

var _ rest.Getter = &partialUpdateStore{}
var _ rest.Updater = &partialUpdateStore{}

type partialUpdateStore struct {
	rest.StandardStorage
}
