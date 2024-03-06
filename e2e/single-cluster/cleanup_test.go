package singlecluster_test

import (
	"math/rand"
	"time"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/fleet/e2e/testenv"
	"github.com/rancher/fleet/e2e/testenv/kubectl"
)

var _ = Describe("Cleaning up orphan resources", func() {
	var (
		k               kubectl.Command
		targetNamespace string
		r               = rand.New(rand.NewSource(GinkgoRandomSeed()))
	)

	BeforeEach(func() {
		k = env.Kubectl.Namespace(env.Namespace)
	})

	When("Deleting a gitrepo", func() {
		JustBeforeEach(func() {
			targetNamespace = testenv.NewNamespaceName("target", r)

			err := testenv.CreateGitRepo(k, targetNamespace, "cleanup-bundles", "master", "simple-chart")
			Expect(err).ToNot(HaveOccurred())

			DeferCleanup(func() {
				out, err := k.Delete("ns", targetNamespace)
				Expect(err).ToNot(HaveOccurred(), out)
			})
		})

		It("deletes any orphan bundles", func() {
			By("checking the bundle exists")
			Eventually(func() string {
				out, _ := k.Namespace(env.Namespace).Get("bundles")
				return out
			}).Should(ContainSubstring("cleanup-bundles-simple-chart"))

			By("deleting the gitrepo")
			Eventually(func() error {
				_, err := k.Namespace(env.Namespace).Delete("gitrepo", "cleanup-bundles")
				return err
			}).ShouldNot(HaveOccurred())

			Eventually(func() string {
				out, _ := k.Namespace(env.Namespace).Get("bundles")
				return out
			}).WithTimeout(10 * time.Second).ShouldNot(ContainSubstring("cleanup-bundles-simple-chart"))
		})
	})
})
