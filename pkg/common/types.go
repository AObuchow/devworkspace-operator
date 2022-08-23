package common

import (
	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"

	dw "github.com/devfile/api/v2/pkg/apis/workspaces/v1alpha2"
)

//TODO: Rename?
// TODO: Move
type DevWorkspaceWithConfig struct {
	dw.DevWorkspace `json:",inline"`
	Config          controllerv1alpha1.OperatorConfiguration
}
