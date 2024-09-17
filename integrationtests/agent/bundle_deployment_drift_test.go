package agent

import (
	"context"
	"time"

	"github.com/rancher/fleet/integrationtests/utils"
	"github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	"sigs.k8s.io/controller-runtime/pkg/client"
)

var _ = Describe("BundleDeployment drift correction", Ordered, func() {

	const svcName = "svc-test"

	var (
		namespace    string
		name         string
		env          *specEnv
		correctDrift v1alpha1.CorrectDrift
	)

	createBundleDeployment := func(name string) {
		bundled := v1alpha1.BundleDeployment{
			ObjectMeta: metav1.ObjectMeta{
				Name:      name,
				Namespace: clusterNS,
			},
			Spec: v1alpha1.BundleDeploymentSpec{
				DeploymentID: "v1",
				Options: v1alpha1.BundleDeploymentOptions{
					DefaultNamespace: namespace,
					CorrectDrift:     &correctDrift,
					Helm: &v1alpha1.HelmOptions{
						MaxHistory: 2,
					},
				},
				CorrectDrift: &v1alpha1.CorrectDrift{
					Enabled: true,
				},
			},
		}

		err := k8sClient.Create(context.TODO(), &bundled)
		Expect(err).To(BeNil())
		Expect(bundled).To(Not(BeNil()))
	}

	createNamespace := func() string {
		newNs, err := utils.NewNamespaceName()
		Expect(err).ToNot(HaveOccurred())

		ns := &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: newNs}}
		Expect(k8sClient.Create(context.Background(), ns)).ToNot(HaveOccurred())

		return newNs
	}

	When("Drift correction is not enabled", func() {
		BeforeAll(func() {
			namespace = createNamespace()
			correctDrift = v1alpha1.CorrectDrift{Enabled: false}
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: namespace}})).ToNot(HaveOccurred())
			})
			env = &specEnv{namespace: namespace}

			name = "drift-disabled-test"
			createBundleDeployment(name)
			Eventually(env.isBundleDeploymentReadyAndNotModified).WithArguments(name).Should(BeTrue())

			DeferCleanup(func() {
				Expect(k8sClient.Delete(context.TODO(), &v1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{Namespace: clusterNS, Name: name},
				})).ToNot(HaveOccurred())
			})
		})

		Context("Modifying externalName in service resource", func() {
			It("Receives a modification on a service", func() {
				svc, err := env.getService(svcName)
				Expect(err).NotTo(HaveOccurred())
				patchedSvc := svc.DeepCopy()
				patchedSvc.Spec.ExternalName = "modified"
				Expect(k8sClient.Patch(ctx, patchedSvc, client.StrategicMergeFrom(&svc))).NotTo(HaveOccurred())
			})

			It("Preserves the modification on the service", func() {
				Consistently(func(g Gomega) {
					svc, err := env.getService(svcName)
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(svc.Spec.ExternalName).Should(Equal("modified"))
				}, 2*time.Second, 100*time.Millisecond)
			})
		})
	})

	When("Drift correction is enabled without force", func() {
		JustBeforeEach(func() {
			correctDrift = v1alpha1.CorrectDrift{Enabled: true}
			DeferCleanup(func() {
				Expect(k8sClient.Delete(ctx, &corev1.Namespace{
					ObjectMeta: metav1.ObjectMeta{Name: namespace}})).ToNot(HaveOccurred())
			})
			env = &specEnv{namespace: namespace}

			createBundleDeployment(name)
			Eventually(env.isBundleDeploymentReadyAndNotModified).WithArguments(name).Should(BeTrue())

			DeferCleanup(func() {
				Expect(k8sClient.Delete(context.TODO(), &v1alpha1.BundleDeployment{
					ObjectMeta: metav1.ObjectMeta{Namespace: clusterNS, Name: name},
				})).ToNot(HaveOccurred())
			})
		})

		Context("Modifying externalName in a service resource", func() {
			BeforeEach(func() {
				namespace = createNamespace()
				name = "drift-service-externalname-test"
			})

			It("Corrects drift", func() {
				By("Receiving a modification on a service")
				svc, err := env.getService(svcName)
				Expect(err).NotTo(HaveOccurred())
				patchedSvc := svc.DeepCopy()
				patchedSvc.Spec.ExternalName = "modified"
				Expect(k8sClient.Patch(ctx, patchedSvc, client.StrategicMergeFrom(&svc))).NotTo(HaveOccurred())

				By("Restoring the service resource to its previous state")
				Eventually(func(g Gomega) {
					svc, err := env.getService(svcName)
					g.Expect(err).NotTo(HaveOccurred())

					g.Expect(svc.Spec.ExternalName).Should(Equal("svc-test"))
				})
			})
		})

		Context("Drift correction fails", func() {
			BeforeEach(func() {
				namespace = createNamespace()
				name = "drift-test"
			})

			It("Updates the BundleDeployment status as not Ready, including the error message", func() {
				By("Receiving a modification on a service")
				svc, err := env.getService(svcName)
				Expect(err).NotTo(HaveOccurred())
				patchedSvc := svc.DeepCopy()
				patchedSvc.Spec.Ports[0].TargetPort = intstr.FromInt(4242)
				patchedSvc.Spec.Ports[0].Port = 4242
				patchedSvc.Spec.Ports[0].Name = "myport"
				Expect(k8sClient.Patch(ctx, patchedSvc, client.StrategicMergeFrom(&svc))).NotTo(HaveOccurred())

				By("Updating the bundle deployment status")
				Eventually(func(g Gomega) {
					modifiedStatus := v1alpha1.ModifiedStatus{
						Kind:       "Service",
						APIVersion: "v1",
						Namespace:  namespace,
						Name:       "svc-test",
						Create:     false,
						Delete:     false,
						Patch:      `{"spec":{"ports":[{"name":"myport","port":80,"protocol":"TCP","targetPort":9376},{"name":"myport","port":4242,"protocol":"TCP","targetPort":4242}]}}`,
					}
					isOK, status := env.isNotReadyAndModified(
						name,
						modifiedStatus,
						`cannot patch "svc-test" with kind Service: Service "svc-test" is invalid: spec.ports[1].name: Duplicate value: "myport"`,
					)
					g.Expect(isOK).To(BeTrue(), status)
				}).Should(Succeed())
			})
		})
	})
})
