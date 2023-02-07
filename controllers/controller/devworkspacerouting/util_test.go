// Copyright (c) 2019-2023 Red Hat, Inc.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package devworkspacerouting_test

import (
	"fmt"
	"time"

	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/common"
	"github.com/devfile/devworkspace-operator/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routeV1 "github.com/openshift/api/route/v1"
	crclient "sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	k8sErrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

const (
	timeout  = 10 * time.Second
	interval = 250 * time.Millisecond
)

func createPreparingDWR(workspaceID string, name string) *controllerv1alpha1.DevWorkspaceRouting {
	mainAttributes := controllerv1alpha1.Attributes{}
	mainAttributes.PutString("type", "main")
	exposedEndpoint := controllerv1alpha1.Endpoint{
		Name:       exposedEndPointName,
		Attributes: mainAttributes,
		// TODO: This seems kinda of hacky? Ask Angel about this
		// Lack of target port causes preparing state
	}
	endpointsList := map[string]controllerv1alpha1.EndpointList{
		exposedEndPointName: {
			exposedEndpoint,
		},
	}

	dwr := &controllerv1alpha1.DevWorkspaceRouting{
		Spec: controllerv1alpha1.DevWorkspaceRoutingSpec{
			DevWorkspaceId: workspaceID,
			RoutingClass:   controllerv1alpha1.DevWorkspaceRoutingBasic,
			Endpoints:      endpointsList,
			PodSelector: map[string]string{
				constants.DevWorkspaceIDLabel: workspaceID,
			},
		},
	}

	dwr.SetName(name)
	dwr.SetNamespace(testNamespace)

	Expect(k8sClient.Create(ctx, dwr)).Should(Succeed())
	return dwr
}

func createDWR(workspaceID string, name string) *controllerv1alpha1.DevWorkspaceRouting {
	mainAttributes := controllerv1alpha1.Attributes{}
	mainAttributes.PutString("type", "main")
	discoverableAttributes := controllerv1alpha1.Attributes{}
	discoverableAttributes.PutBoolean(string(controllerv1alpha1.DiscoverableAttribute), true)

	exposedEndpoint := controllerv1alpha1.Endpoint{
		Name:       exposedEndPointName,
		Exposure:   controllerv1alpha1.PublicEndpointExposure,
		Attributes: mainAttributes,
		TargetPort: exposedTargetPort,
	}
	nonExposedEndpoint := controllerv1alpha1.Endpoint{
		Name:       nonExposedEndpointName,
		Exposure:   controllerv1alpha1.NoneEndpointExposure,
		TargetPort: nonExposedTargetPort,
	}
	discoverableEndpoint := controllerv1alpha1.Endpoint{
		Name:       discoverableEndpointName,
		Exposure:   controllerv1alpha1.PublicEndpointExposure,
		Attributes: discoverableAttributes,
		TargetPort: discoverableTargetPort,
	}
	endpointsList := map[string]controllerv1alpha1.EndpointList{
		exposedEndPointName: {
			exposedEndpoint,
			nonExposedEndpoint,
			discoverableEndpoint,
		},
	}

	dwr := &controllerv1alpha1.DevWorkspaceRouting{
		Spec: controllerv1alpha1.DevWorkspaceRoutingSpec{
			DevWorkspaceId: workspaceID,
			RoutingClass:   controllerv1alpha1.DevWorkspaceRoutingBasic,
			Endpoints:      endpointsList,
			PodSelector: map[string]string{
				constants.DevWorkspaceIDLabel: workspaceID,
			},
		},
	}

	dwr.SetName(name)
	dwr.SetNamespace(testNamespace)

	Expect(k8sClient.Create(ctx, dwr)).Should(Succeed())
	return dwr
}

func getExistingDevWorkspaceRouting(name string) *controllerv1alpha1.DevWorkspaceRouting {
	By(fmt.Sprintf("Getting existing DevWorkspaceRouting %s", name))
	dwr := &controllerv1alpha1.DevWorkspaceRouting{}
	dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
	Eventually(func() (string, error) {
		if err := k8sClient.Get(ctx, dwrNamespacedName, dwr); err != nil {
			return "", err
		}
		return dwr.Spec.DevWorkspaceId, nil
	}, timeout, interval).Should(Not(BeEmpty()), "DevWorkspaceRouting should exist in cluster")
	return dwr
}

func getReadyDevWorkspaceRouting(name string) *controllerv1alpha1.DevWorkspaceRouting {
	dwr := getExistingDevWorkspaceRouting(name)

	dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
	Eventually(func() (bool, error) {
		if err := k8sClient.Get(ctx, dwrNamespacedName, dwr); err != nil {
			return false, err
		}
		return controllerv1alpha1.DevWorkspaceRoutingPhase(dwr.Status.Phase) == controllerv1alpha1.RoutingReady, nil
	}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should exist in cluster")
	return dwr
}

func deleteService(workspaceID string, namespace string) {
	createdService := &corev1.Service{}
	serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), namespace)
	Eventually(func() bool {
		err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
		return err == nil
	}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")
	deleteObject(createdService)
}

func deleteRoute(workspaceID string, namespace string) {
	createdRoute := routeV1.Route{}
	// TODO: Add endpointName as a function parameter?
	routeNamespacedName := namespacedName(common.RouteName(workspaceID, exposedEndPointName), namespace)
	Eventually(func() bool {
		err := k8sClient.Get(ctx, routeNamespacedName, &createdRoute)
		return err == nil
	}, timeout, interval).Should(BeTrue(), "Route should exist in cluster")
	deleteObject(&createdRoute)
}

func deleteIngress(workspaceID string, namespace string) {
	createdIngress := networkingv1.Ingress{}
	// TODO: Add endpointName as a function parameter?
	ingressNamespacedName := namespacedName(common.RouteName(workspaceID, exposedEndPointName), namespace)
	Eventually(func() bool {
		err := k8sClient.Get(ctx, ingressNamespacedName, &createdIngress)
		return err == nil
	}, timeout, interval).Should(BeTrue(), "Ingress should exist in cluster")
	deleteObject(&createdIngress)
}

func deleteDevWorkspaceRouting(name string) {
	dwNN := namespacedName(name, testNamespace)
	dwr := &controllerv1alpha1.DevWorkspaceRouting{}
	dwr.Name = name
	dwr.Namespace = testNamespace
	// Do nothing if already deleted
	err := k8sClient.Delete(ctx, dwr)
	if k8sErrors.IsNotFound(err) {
		return
	}
	Expect(err).Should(BeNil())

	Eventually(func() bool {
		err := k8sClient.Get(ctx, dwNN, dwr)
		return err != nil && k8sErrors.IsNotFound(err)
	}, 10*time.Second, 250*time.Millisecond).Should(BeTrue(), "DevWorkspaceRouting not deleted after timeout")
}

func devWorkspaceRoutingOwnerRef(dwr *controllerv1alpha1.DevWorkspaceRouting) metav1.OwnerReference {
	boolTrue := true
	return metav1.OwnerReference{
		APIVersion:         "controller.devfile.io/v1alpha1",
		Kind:               "DevWorkspaceRouting",
		Name:               dwr.Name,
		UID:                dwr.UID,
		Controller:         &boolTrue,
		BlockOwnerDeletion: &boolTrue,
	}
}

func createObject(obj crclient.Object) {
	Expect(k8sClient.Create(ctx, obj)).Should(Succeed())
	Eventually(func() error {
		return k8sClient.Get(ctx, namespacedName(obj.GetName(), obj.GetNamespace()), obj)
	}, 10*time.Second, 250*time.Millisecond).Should(Succeed(), "Creating %s with name %s", obj.GetObjectKind(), obj.GetName())
}

func deleteObject(obj crclient.Object) {
	Expect(k8sClient.Delete(ctx, obj)).Should(Succeed())
	Eventually(func() bool {
		err := k8sClient.Get(ctx, namespacedName(obj.GetName(), obj.GetNamespace()), obj)
		return k8sErrors.IsNotFound(err)
	}, 10*time.Second, 250*time.Millisecond).Should(BeTrue(), "Deleting %s with name %s", obj.GetObjectKind(), obj.GetName())
}

func namespacedName(name, namespace string) types.NamespacedName {
	return types.NamespacedName{
		Name:      name,
		Namespace: namespace,
	}
}
