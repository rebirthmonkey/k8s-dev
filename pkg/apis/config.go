package apis

import (
	"fmt"
)

// +kubebuilder:object:generate=false
type ResourcesConfigsMap map[string]map[string]ResourceConfig

func (rc ResourcesConfigsMap) Add(group, version, resource string, config ResourceConfig) {
	gv := fmt.Sprintf("%s/%s", group, version)
	rcv, ok := rc[gv]
	if !ok {
		rcv = make(map[string]ResourceConfig)
		rc[gv] = rcv
	}
	rcv[resource] = config
}

var ResourcesConfigs = make(ResourcesConfigsMap)
