package common

import (
	dw "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/config"
	"k8s.io/apimachinery/pkg/types"
)

//TODO: Rename?
// TODO: Move
type DevWorkspaceWithConfig struct {
	dw.DevWorkspace `json:",inline"`
	Config          controllerv1alpha1.OperatorConfiguration
}

func GetConfig(namespacedName types.NamespacedName) controllerv1alpha1.OperatorConfiguration {
	// TODO: Implement

	// If external config exists on the cluster, merge it with global one and return
	// Otherwise return the global config
	return *config.InternalConfig
}
