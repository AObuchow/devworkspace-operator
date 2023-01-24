package devworkspacerouting_test

import (
	"fmt"
	"os"
	"path/filepath"

	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/common"
	"github.com/devfile/devworkspace-operator/pkg/constants"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/yaml"
)

func loadObjectFromFile(objName string, obj client.Object, filename string) error {
	path := filepath.Join("testdata", filename)
	bytes, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	err = yaml.Unmarshal(bytes, obj)
	if err != nil {
		return err
	}
	obj.SetNamespace(testNamespace)
	obj.SetName(objName)

	return nil
}

//TODO: Move to util test
func createPreparingDWR(workspaceID string, name string) *controllerv1alpha1.DevWorkspaceRouting {
	mainAttributes := controllerv1alpha1.Attributes{}
	mainAttributes.PutString("type", "main")
	exposedEndpoint := controllerv1alpha1.Endpoint{
		Name:       "test-endpoint",
		Attributes: mainAttributes,
		// TODO: This seems kinda of hacky? Ask Angel about this
		// Lack of target port causes preparing state
	}
	endpointsList := map[string]controllerv1alpha1.EndpointList{
		"test-endpoint": {
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
	exposedEndpoint := controllerv1alpha1.Endpoint{
		Name:       "test-endpoint",
		Attributes: mainAttributes,
		TargetPort: 7777,
	}
	endpointsList := map[string]controllerv1alpha1.EndpointList{
		"test-endpoint": {
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

var _ = Describe("DevWorkspaceRouting Controller", func() {
	Context("Basic DevWorkspaceRouting Tests", func() {
		It("Gets Preparing status", func() {
			By("Creating a new DevWorkspaceRouting object")
			workspaceID := "test-id"
			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			createdDWR := createPreparingDWR(workspaceID, devWorkspaceRoutingName)
			defer deleteDevWorkspaceRouting(devWorkspaceRoutingName)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should exist in cluster")

			By("Checking DevWorkspaceRouting Status is updated to starting")
			Eventually(func() (phase controllerv1alpha1.DevWorkspaceRoutingPhase, err error) {
				if err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR); err != nil {
					return "", err
				}
				return createdDWR.Status.Phase, nil
			}, timeout, interval).Should(Equal(controllerv1alpha1.RoutingPreparing), "DevWorkspaceRouting should have Preparing phase")

			Expect(createdDWR.Status.Message).ShouldNot(BeNil(), "Status message should be set for preparing DevWorkspaceRoutings")

		})

		// TODO: Move ingresses and services thing to new context "object creation"
		It("Gets Ready phase, ingresses and services are created", func() {
			By("Creating a new DevWorkspaceRouting object")
			workspaceID := "test-id"

			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			createdDWR := createDWR(workspaceID, devWorkspaceRoutingName)
			defer deleteDevWorkspaceRouting(devWorkspaceRoutingName)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should exist in cluster")

			By("Checking DevWorkspaceRouting Status is updated to starting")
			Eventually(func() (phase controllerv1alpha1.DevWorkspaceRoutingPhase, err error) {
				if err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR); err != nil {
					return "", err
				}
				return createdDWR.Status.Phase, nil
			}, timeout, interval).Should(Equal(controllerv1alpha1.RoutingReady), "DevWorkspaceRouting should have Ready phase")

			Expect(createdDWR.Status.Message).ShouldNot(BeNil(), "Status message should be set for preparing DevWorkspaceRoutings")

			By("Checking service is created")
			createdService := &corev1.Service{}
			serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")

			Expect(createdService.Spec.Selector).Should(Equal(createdDWR.Spec.PodSelector), "Service should have pod selector from DevWorkspace metadata")
			Expect(createdService.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP), "Service type should be Cluster IP")
			expectedLabels := make(map[string]string)
			expectedLabels[constants.DevWorkspaceIDLabel] = workspaceID
			Expect(createdService.Labels).Should(Equal(expectedLabels), "Service should contain DevWorkspace ID label")

			expectedOwnerReference := devWorkspaceRoutingOwnerRef(createdDWR)
			Expect(createdService.OwnerReferences).Should(ContainElement(expectedOwnerReference), "Service should be owned by DevWorkspaceRouting")

			targetPort := 7777
			endPointName := "test-endpoint"
			var expectedServicePorts []corev1.ServicePort
			expectedServicePorts = append(expectedServicePorts, corev1.ServicePort{
				Name:       common.EndpointName(endPointName),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(targetPort),
				TargetPort: intstr.FromInt(targetPort),
			})

			Expect(createdService.Spec.Ports).Should(Equal(expectedServicePorts), "Service should contain expected ports")

			By("Checking ingress is created")
			createdIngress := networkingv1.Ingress{}
			ingressNamespacedName := namespacedName(common.RouteName(workspaceID, endPointName), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ingressNamespacedName, &createdIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Ingress should exist in cluster")

			Expect(createdIngress.Labels).Should(Equal(expectedLabels), "Ingress should contain DevWorkspace ID label")
			Expect(createdIngress.OwnerReferences).Should(ContainElement(expectedOwnerReference), "Ingress should be owned by DevWorkspaceRouting")

			Expect(createdIngress.Spec.Rules).ShouldNot(BeEmpty(), "Ingress should have rules")
			// TODO: Not sure about these.. seem kinda redundant.
			// TODO: Could also be more informative in expected/actual value
			// TODO: Could create variables so its not a chain of structs within the spec
			Expect(len(createdIngress.Spec.Rules)).Should(Equal(1), "Ingress should have a single rule")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue).ShouldNot(BeNil(), "Ingress should have a rule value")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP).ShouldNot(BeNil(), "Ingress should have a HTTP rule value")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths).ShouldNot(BeEmpty(), "Ingress should have a HTTP rule value path")
			Expect(len(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths)).Should(Equal(1), "Ingress should have a single HTTP rule value path")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service).ShouldNot(BeNil(), "Ingress should have a backend service")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Name).Should(Equal(common.ServiceName(workspaceID)), "Incorrect ingress backend service name")
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port).Should(Equal(networkingv1.ServiceBackendPort{Number: int32(targetPort)}), "Incorrect ingress backend service port")

			// TODO: Check owner references for services
			// TODO: Check finalizers exist before DWR is deleted and removed after deleted for ingress, routes and services
			// TODO: Check deletion timestamp on ingress, routes and services

			// TODO: Add an endpoint that isin't controllerv1alpha1.PublicEndpointExposure to test no ingress is made for it

			// TODO: Check routes (maybe not cause not openshift?)
			// TODO: Check exposed endpoints
			// TODO: Check finalizers

			// TODO: Failure cases

		})

	})

})
