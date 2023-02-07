package devworkspacerouting_test

import (
	"fmt"

	controllerv1alpha1 "github.com/devfile/devworkspace-operator/apis/controller/v1alpha1"
	"github.com/devfile/devworkspace-operator/pkg/common"
	"github.com/devfile/devworkspace-operator/pkg/config"
	"github.com/devfile/devworkspace-operator/pkg/constants"
	"github.com/devfile/devworkspace-operator/pkg/infrastructure"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	routeV1 "github.com/openshift/api/route/v1"
	corev1 "k8s.io/api/core/v1"
	networkingv1 "k8s.io/api/networking/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
)

var _ = Describe("DevWorkspaceRouting Controller", func() {
	AfterEach(func() {
		infrastructure.InitializeForTesting(infrastructure.Kubernetes)
	})
	Context("Basic DevWorkspaceRouting Tests", func() {
		It("Gets Preparing status", func() {
			By("Creating a new DevWorkspaceRouting object")
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

		It("Gets Ready Status on OpenShift", func() {
			By("Setting infrastructure to OpenShift")
			infrastructure.InitializeForTesting(infrastructure.OpenShiftv4)

			By("Creating a new DevWorkspaceRouting object")

			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			createdDWR := createDWR(workspaceID, devWorkspaceRoutingName)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should exist in cluster")

			By("Checking DevWorkspaceRouting Status is updated to Ready")
			Eventually(func() (phase controllerv1alpha1.DevWorkspaceRoutingPhase, err error) {
				if err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR); err != nil {
					return "", err
				}
				return createdDWR.Status.Phase, nil
			}, timeout, interval).Should(Equal(controllerv1alpha1.RoutingReady), "DevWorkspaceRouting should have Ready phase")

			Expect(createdDWR.Status.Message).ShouldNot(BeNil(), "Status message should be set for preparing DevWorkspaceRoutings")

			deleteDevWorkspaceRouting(devWorkspaceRoutingName)
			deleteService(workspaceID, testNamespace)
			deleteRoute(workspaceID, testNamespace)
		})

		It("Gets Ready Status on Kubernetes", func() {
			By("Setting infrastructure to Kubernetes")
			infrastructure.InitializeForTesting(infrastructure.Kubernetes)

			By("Creating a new DevWorkspaceRouting object")

			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			createdDWR := createDWR(workspaceID, devWorkspaceRoutingName)

			Eventually(func() bool {
				err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should exist in cluster")

			By("Checking DevWorkspaceRouting Status is updated to Ready")
			Eventually(func() (phase controllerv1alpha1.DevWorkspaceRoutingPhase, err error) {
				if err := k8sClient.Get(ctx, dwrNamespacedName, createdDWR); err != nil {
					return "", err
				}
				return createdDWR.Status.Phase, nil
			}, timeout, interval).Should(Equal(controllerv1alpha1.RoutingReady), "DevWorkspaceRouting should have Ready phase")

			Expect(createdDWR.Status.Message).ShouldNot(BeNil(), "Status message should be set for preparing DevWorkspaceRoutings")

			// No finalizer check required as basic_solver and cluster_solver don't require finalizers

			// No deletion timestamp on ingress, routes and services

			deleteDevWorkspaceRouting(devWorkspaceRoutingName)
			deleteService(workspaceID, testNamespace)
			deleteIngress(workspaceID, testNamespace)
		})
	})

	Context("Kubernetes - DevWorkspaceRouting Objects creation", func() {

		BeforeEach(func() {
			infrastructure.InitializeForTesting(infrastructure.Kubernetes)
			createDWR(workspaceID, devWorkspaceRoutingName)
		})

		AfterEach(func() {
			deleteDevWorkspaceRouting(devWorkspaceRoutingName)
			deleteService(workspaceID, testNamespace)
			deleteIngress(workspaceID, testNamespace)
		})
		It("Creates services", func() {
			createdDWR := getExistingDevWorkspaceRouting(devWorkspaceRoutingName)

			// TODO: Factor out labels?
			expectedLabels := make(map[string]string)
			expectedLabels[constants.DevWorkspaceIDLabel] = workspaceID

			By("Checking single service for all exposed endpoints is created")
			createdService := &corev1.Service{}
			serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")

			Expect(createdService.Spec.Selector).Should(Equal(createdDWR.Spec.PodSelector), "Service should have pod selector from DevWorkspace metadata")
			Expect(createdService.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP), "Service type should be Cluster IP")

			Expect(createdService.Labels).Should(Equal(expectedLabels), "Service should contain DevWorkspace ID label")
			expectedOwnerReference := devWorkspaceRoutingOwnerRef(createdDWR)
			Expect(createdService.OwnerReferences).Should(ContainElement(expectedOwnerReference), "Service should be owned by DevWorkspaceRouting")

			By("Checking service has expected ports")
			var expectedServicePorts []corev1.ServicePort
			expectedServicePorts = append(expectedServicePorts, corev1.ServicePort{
				Name:       common.EndpointName(exposedEndPointName),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(exposedTargetPort),
				TargetPort: intstr.FromInt(exposedTargetPort),
			})

			expectedServicePorts = append(expectedServicePorts, corev1.ServicePort{
				Name:       common.EndpointName(discoverableEndpointName),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(discoverableTargetPort),
				TargetPort: intstr.FromInt(discoverableTargetPort),
			})

			Expect(len(createdService.Spec.Ports)).Should(Equal(2), fmt.Sprintf("Only two ports should be exposed: %s and %s. The remaining endpoint in the DevWorkspaceRouting spec has None exposure.", exposedEndPointName, discoverableEndpointName))
			Expect(createdService.Spec.Ports).Should(Equal(expectedServicePorts), "Service should contain expected ports")

			By("Checking service is created for discoverable endpoint")
			discoverableEndpointService := &corev1.Service{}
			discoverableEndpointServiceNamespacedName := namespacedName(common.EndpointName(discoverableEndpointName), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, discoverableEndpointServiceNamespacedName, discoverableEndpointService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service for discoverable endpoint should exist in cluster")
			Expect(len(discoverableEndpointService.Spec.Ports)).Should(Equal(1), "Service for discoverable endpoint should only have a single port")
			Expect(discoverableEndpointService.Spec.Ports[0].Port).Should(Equal(int32(discoverableTargetPort)))
			Expect(discoverableEndpointService.Spec.Ports[0].TargetPort).Should(Equal(intstr.FromInt(discoverableTargetPort)))
		})

		It("Creates ingress", func() {
			createdDWR := getExistingDevWorkspaceRouting(devWorkspaceRoutingName)

			// TODO: Factor out labels?
			expectedLabels := make(map[string]string)
			expectedLabels[constants.DevWorkspaceIDLabel] = workspaceID

			By("Checking ingress is created")
			createdIngress := networkingv1.Ingress{}
			ingressNamespacedName := namespacedName(common.RouteName(workspaceID, exposedEndPointName), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, ingressNamespacedName, &createdIngress)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Ingress should exist in cluster")

			Expect(createdIngress.Labels).Should(Equal(expectedLabels), "Ingress should contain DevWorkspace ID label")
			expectedOwnerReference := devWorkspaceRoutingOwnerRef(createdDWR)
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
			Expect(createdIngress.Spec.Rules[0].IngressRuleValue.HTTP.Paths[0].Backend.Service.Port).Should(Equal(networkingv1.ServiceBackendPort{Number: int32(exposedTargetPort)}), "Incorrect ingress backend service port")

			By("Checking ingress points to service")
			createdService := &corev1.Service{}
			serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")

			var targetPorts []intstr.IntOrString
			var ports []int32
			for _, servicePort := range createdService.Spec.Ports {
				targetPorts = append(targetPorts, servicePort.TargetPort)
				ports = append(ports, servicePort.Port)
			}
			Expect(len(createdIngress.Spec.Rules)).Should(Equal(1), "Expected only a single rule for the ingress")
			ingressRule := createdIngress.Spec.Rules[0]

			Expect(ingressRule.HTTP.Paths[0].Backend.Service.Name).Should(Equal(createdService.Name), "Ingress backend service name should be service name")
			Expect(ports).Should(ContainElement(ingressRule.HTTP.Paths[0].Backend.Service.Port.Number), "Ingress backend service port should be in service ports")
			Expect(targetPorts).Should(ContainElement(intstr.FromInt(int(ingressRule.HTTP.Paths[0].Backend.Service.Port.Number))), "Ingress backend service port should be service target ports")
		})

	})

	Context("OpenShift - DevWorkspaceRouting Objects creation", func() {

		BeforeEach(func() {
			infrastructure.InitializeForTesting(infrastructure.OpenShiftv4)
			createDWR(workspaceID, devWorkspaceRoutingName)
		})

		AfterEach(func() {
			deleteDevWorkspaceRouting(devWorkspaceRoutingName)
			deleteService(workspaceID, testNamespace)
			deleteRoute(workspaceID, testNamespace)
		})
		It("Creates services", func() {
			createdDWR := getExistingDevWorkspaceRouting(devWorkspaceRoutingName)

			// TODO: Factor out labels?
			expectedLabels := make(map[string]string)
			expectedLabels[constants.DevWorkspaceIDLabel] = workspaceID

			By("Checking single service for all exposed endpoints is created")
			createdService := &corev1.Service{}
			serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")

			Expect(createdService.Spec.Selector).Should(Equal(createdDWR.Spec.PodSelector), "Service should have pod selector from DevWorkspace metadata")
			Expect(createdService.Spec.Type).Should(Equal(corev1.ServiceTypeClusterIP), "Service type should be Cluster IP")
			Expect(createdService.Labels).Should(Equal(expectedLabels), "Service should contain DevWorkspace ID label")

			expectedOwnerReference := devWorkspaceRoutingOwnerRef(createdDWR)
			Expect(createdService.OwnerReferences).Should(ContainElement(expectedOwnerReference), "Service should be owned by DevWorkspaceRouting")

			By("Checking service has expected ports")
			var expectedServicePorts []corev1.ServicePort
			expectedServicePorts = append(expectedServicePorts, corev1.ServicePort{
				Name:       common.EndpointName(exposedEndPointName),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(exposedTargetPort),
				TargetPort: intstr.FromInt(exposedTargetPort),
			})

			expectedServicePorts = append(expectedServicePorts, corev1.ServicePort{
				Name:       common.EndpointName(discoverableEndpointName),
				Protocol:   corev1.ProtocolTCP,
				Port:       int32(discoverableTargetPort),
				TargetPort: intstr.FromInt(discoverableTargetPort),
			})

			Expect(createdService.Spec.Ports).Should(Equal(expectedServicePorts), "Service should contain expected ports")

			By("Checking service is created for discoverable endpoint")
			discoverableEndpointService := &corev1.Service{}
			discoverableEndpointServiceNamespacedName := namespacedName(common.EndpointName(discoverableEndpointName), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, discoverableEndpointServiceNamespacedName, discoverableEndpointService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service for discoverable endpoint should exist in cluster")
			Expect(len(discoverableEndpointService.Spec.Ports)).Should(Equal(1), "Service for discoverable endpoint should only have a single port")
			Expect(discoverableEndpointService.Spec.Ports[0].Port).Should(Equal(int32(discoverableTargetPort)))
			Expect(discoverableEndpointService.Spec.Ports[0].TargetPort).Should(Equal(intstr.FromInt(discoverableTargetPort)))
		})

		It("Creates route", func() {
			createdDWR := getExistingDevWorkspaceRouting(devWorkspaceRoutingName)

			// TODO: Factor out labels?
			expectedLabels := make(map[string]string)
			expectedLabels[constants.DevWorkspaceIDLabel] = workspaceID

			By("Checking route is created")
			createdRoute := routeV1.Route{}
			routeNamespacedName := namespacedName(common.RouteName(workspaceID, exposedEndPointName), testNamespace)
			Eventually(func() error {
				err := k8sClient.Get(ctx, routeNamespacedName, &createdRoute)
				return err
			}, timeout, interval).Should(BeNil(), "Route should exist in cluster")

			Expect(createdRoute.Labels).Should(Equal(expectedLabels), "Route should contain DevWorkspace ID label")
			expectedOwnerReference := devWorkspaceRoutingOwnerRef(createdDWR)
			Expect(createdRoute.OwnerReferences).Should(ContainElement(expectedOwnerReference), "Route should be owned by DevWorkspaceRouting")

			By("Checking route points to service")
			createdService := &corev1.Service{}
			serviceNamespacedName := namespacedName(common.ServiceName(workspaceID), testNamespace)
			Eventually(func() bool {
				err := k8sClient.Get(ctx, serviceNamespacedName, createdService)
				return err == nil
			}, timeout, interval).Should(BeTrue(), "Service should exist in cluster")

			var targetPorts []intstr.IntOrString
			var ports []int32
			for _, servicePort := range createdService.Spec.Ports {
				targetPorts = append(targetPorts, servicePort.TargetPort)
				ports = append(ports, servicePort.Port)
			}
			Expect(targetPorts).Should(ContainElement(createdRoute.Spec.Port.TargetPort), "Route target port should be in service target ports")
			Expect(ports).Should(ContainElement(createdRoute.Spec.Port.TargetPort.IntVal), "Route target port should be in service ports")
			Expect(createdRoute.Spec.To.Name).Should(Equal(createdService.Name), "Route target reference should be service name")
		})
	})

	Context("DevWorkspaceRouting failure cases", func() {

		BeforeEach(func() {
			infrastructure.InitializeForTesting(infrastructure.Kubernetes)
			createDWR(workspaceID, devWorkspaceRoutingName)
			getReadyDevWorkspaceRouting(workspaceID)
		})

		AfterEach(func() {
			config.SetGlobalConfigForTesting(testControllerCfg)
			deleteDevWorkspaceRouting(devWorkspaceRoutingName)
			deleteService(workspaceID, testNamespace)
			deleteIngress(workspaceID, testNamespace)
		})
		It("Fails DevWorkspaceRouting with no routing class", func() {
			// TODO: set no routing class when creating?
			// TODO: What about changing routing classes?
			By("Removing DevWorkspaceRouting's routing class")

			Eventually(func() error {
				createdDWR := getReadyDevWorkspaceRouting(devWorkspaceRoutingName)
				createdDWR.Spec.RoutingClass = ""
				return k8sClient.Update(ctx, createdDWR)
			}, timeout, interval).Should(Succeed(), "DevWorkspaceRouting routing class should be updated on cluster")

			By("Checking that the DevWorkspaceRouting's has the failed status")
			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			updatedDWR := &controllerv1alpha1.DevWorkspaceRouting{}
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, dwrNamespacedName, updatedDWR)
				if err != nil {
					return false, err
				}
				return updatedDWR.Status.Phase == controllerv1alpha1.RoutingFailed, nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should be in failed phase")

		})

		It("Fails DevWorkspaceRouting with cluster-tls routing class on Kubernetes", func() {
			By("Setting cluster-tls DevWorkspaceRouting's routing class")
			Eventually(func() error {
				createdDWR := getReadyDevWorkspaceRouting(devWorkspaceRoutingName)
				createdDWR.Spec.RoutingClass = "cluster-tls"
				return k8sClient.Update(ctx, createdDWR)
			}, timeout, interval).Should(Succeed(), "DevWorkspaceRouting routing class should be updated on cluster")

			By("Checking that the DevWorkspaceRouting's has the failed status")
			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			updatedDWR := &controllerv1alpha1.DevWorkspaceRouting{}
			Eventually(func() (bool, error) {
				err := k8sClient.Get(ctx, dwrNamespacedName, updatedDWR)
				if err != nil {
					return false, err
				}
				return updatedDWR.Status.Phase == controllerv1alpha1.RoutingFailed, nil
			}, timeout, interval).Should(BeTrue(), "DevWorkspaceRouting should be in failed phase")

		})

		It("Fails DevWorkspaceRouting when cluster host suffix missing on Kubernetes", func() {

			By("Removing cluster host suffix from DevWorkspace Operator Configuration")
			dwoc := &controllerv1alpha1.DevWorkspaceOperatorConfig{}
			dwoc.Config = &controllerv1alpha1.OperatorConfiguration{}
			dwoc.Config.Routing = &controllerv1alpha1.RoutingConfig{}
			dwoc.Config.Routing.ClusterHostSuffix = ""
			config.SetGlobalConfigForTesting(dwoc.Config)

			By("Triggering a reconcile")
			Eventually(func() error {
				createdDWR := getReadyDevWorkspaceRouting(devWorkspaceRoutingName)
				createdDWR.Annotations = make(map[string]string)
				createdDWR.Annotations["test"] = "test"
				return k8sClient.Update(ctx, createdDWR)
			}, timeout, interval).Should(Succeed(), "DevWorkspaceRouting annotations should be updated on cluster")

			By("Checking that the DevWorkspaceRouting's has the failed status")
			dwrNamespacedName := namespacedName(devWorkspaceRoutingName, testNamespace)
			updatedDWR := &controllerv1alpha1.DevWorkspaceRouting{}
			Eventually(func() (controllerv1alpha1.DevWorkspaceRoutingPhase, error) {
				err := k8sClient.Get(ctx, dwrNamespacedName, updatedDWR)
				if err != nil {
					return "", err
				}
				return updatedDWR.Status.Phase, nil
			}, timeout, interval).Should(Equal(controllerv1alpha1.RoutingFailed), "DevWorkspaceRouting should be in failed phase")
		})

		// TODO: Could add test for failing workspace when an additional ingress rule is added?
	})

})
