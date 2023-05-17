package require_secrets

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path"
	"time"

	"github.com/go-git/go-git/v5"

	"github.com/rancher/fleet/e2e/testenv"
	"github.com/rancher/fleet/e2e/testenv/githelper"
	"github.com/rancher/fleet/e2e/testenv/kubectl"

	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
)

const (
	port     = 8080
	repoName = "repo"

	// URL of the gitjob service, called from a git post-receive hook script to simulate webhook delivery
	gitjobServiceURL = "gitjob.cattle-fleet-system.svc.cluster.local"
)

// TODO factor this? Table-based test cases or simply helper functions for common logic?

var _ = Describe("Git Repo with polling", func() {
	var (
		tmpdir   string
		clonedir string
		k        kubectl.Command
		gh       *githelper.Git
		clone    *git.Repository
	)

	BeforeEach(func() {
		k = env.Kubectl.Namespace(env.Namespace)

		// Create git server
		out, err := k.Apply("-f", testenv.AssetPath("gitrepo/nginx_deployment.yaml"))
		Expect(err).ToNot(HaveOccurred(), out)

		out, err = k.Apply("-f", testenv.AssetPath("gitrepo/nginx_service.yaml"))
		Expect(err).ToNot(HaveOccurred(), out)

		time.Sleep(3 * time.Second) // give git server time to spin up

		ip, err := getExternalRepoIP(port, repoName)
		Expect(err).ToNot(HaveOccurred())
		gh = githelper.New(ip)

		// Build git repo URL reachable _within_ the cluster, for the GitRepo
		host, err := buildGitHostname(env.Namespace)
		Expect(err).ToNot(HaveOccurred())

		// For some reason, using an HTTP secret makes `git fetch` fail within tektoncd/pipeline;
		// Hence we resort to inline credentials here, for an ephemeral test setup.
		inClusterRepoURL := fmt.Sprintf("http://%s:%s@%s:%d/%s", gh.Username, gh.Password, host, port, repoName)

		err = testenv.ApplyTemplate(k, testenv.AssetPath("gitrepo/gitrepo.yaml"), struct {
			Repo            string
			Branch          string
			PollingInterval string
		}{
			inClusterRepoURL,
			gh.Branch,
			"15s", // default
		})
		Expect(err).ToNot(HaveOccurred())

		tmpdir, _ = os.MkdirTemp("", "fleet-")

		clonedir = path.Join(tmpdir, repoName)

		clone, err = gh.CreateHTTP(clonedir, testenv.AssetPath("gitrepo/sleeper-chart"), "examples")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
		_, _ = k.Delete("gitrepo", "gitrepo-test")
		_, _ = k.Delete("deployment", "git-server")
		_, _ = k.Delete("service", "git-service")
		_, _ = k.Delete("configmap", "hook-script")
	})

	When("updating a git repository monitored via polling", func() {
		It("updates the deployment", func() {
			By("checking the pod exists")
			Eventually(func() string {
				out, _ := k.Namespace("default").Get("pods")
				return out
			}).Should(ContainSubstring("sleeper-"))

			By("updating the git repository")
			replace(path.Join(clonedir, "examples", "Chart.yaml"), "0.1.0", "0.2.0")
			replace(path.Join(clonedir, "examples", "templates", "deployment.yaml"), "name: sleeper", "name: newsleep")

			commit, err := gh.Update(clone)
			Expect(err).ToNot(HaveOccurred())

			By("checking for the updated commit hash in gitrepo")
			Eventually(func() string {
				out, _ := k.Get("gitrepo", "gitrepo-test", "-o", "yaml")
				return out
			}).Should(ContainSubstring("commit: " + commit))

			By("checking the deployment's new name")
			Eventually(func() string {
				out, _ := k.Namespace("default").Get("deployments")
				return out
			}).Should(ContainSubstring("newsleep"))
		})
	})
})

// 1. Create local repo
// 2. Create (executable!) .git/hooks/post-receive hook which calls the external ingress IP with the right payload (use github webhook payload as example?)
// 3. Clone that repo in local folder
// 4. Make changes to clone and push
// 5. Check deployment
var _ = Describe("Git Repo with webhook", func() {
	var (
		tmpdir   string
		clonedir string
		k        kubectl.Command
		gh       *githelper.Git
		clone    *git.Repository
	)

	BeforeEach(func() {
		k = env.Kubectl.Namespace(env.Namespace)

		// Build git repo URL reachable _within_ the cluster, for the GitRepo
		host, err := buildGitHostname(env.Namespace)
		Expect(err).ToNot(HaveOccurred())

		// For some reason, using an HTTP secret makes `git fetch` fail within tektoncd/pipeline;
		// Hence we resort to inline credentials here, for an ephemeral test setup.
		// We extract username and password independently from the environment to prevent a bootstrap problem
		// with the config map.
		username := os.Getenv("GIT_HTTP_USER")
		password := os.Getenv("GIT_HTTP_PASSWORD")
		inClusterRepoURL := fmt.Sprintf("http://%s:%s@%s:%d/%s", username, password, host, port, repoName)

		// Create git server with git hook
		err = testenv.ApplyTemplate(k, testenv.AssetPath("gitrepo/hook_configmap.yaml"), struct {
			GitJobURL string
			RepoURL   string
		}{
			gitjobServiceURL,
			inClusterRepoURL,
		})
		Expect(err).ToNot(HaveOccurred())

		out, err := k.Apply("-f", testenv.AssetPath("gitrepo/nginx_deployment_with_hook.yaml"))
		Expect(err).ToNot(HaveOccurred(), out)

		out, err = k.Apply("-f", testenv.AssetPath("gitrepo/nginx_service.yaml"))
		Expect(err).ToNot(HaveOccurred(), out)

		time.Sleep(3 * time.Second) // give git server time to spin up

		// Build git repo URL reachable _within_ the cluster, for the GitRepo
		host, err = buildGitHostname(env.Namespace)
		Expect(err).ToNot(HaveOccurred())

		ip, err := getExternalRepoIP(port, repoName)
		Expect(err).ToNot(HaveOccurred())
		gh = githelper.New(ip)

		err = testenv.ApplyTemplate(k, testenv.AssetPath("gitrepo/gitrepo.yaml"), struct {
			Repo            string
			Branch          string
			PollingInterval string
		}{
			inClusterRepoURL,
			gh.Branch,
			"24h", // prevent polling
		})
		Expect(err).ToNot(HaveOccurred())

		tmpdir, _ = os.MkdirTemp("", "fleet-")

		// Clone previously created repo
		clonedir = path.Join(tmpdir, "clone")
		clone, err = gh.CreateHTTP(clonedir, testenv.AssetPath("gitrepo/sleeper-chart"), "examples")
		Expect(err).ToNot(HaveOccurred())
	})

	AfterEach(func() {
		os.RemoveAll(tmpdir)
		//_, _ = k.Delete("secret", "git-auth")
		_, _ = k.Delete("gitrepo", "gitrepo-test")
		_, _ = k.Delete("deployment", "git-server")
		_, _ = k.Delete("service", "git-service")
		_, _ = k.Delete("configmap", "hook-script")
	})

	When("updating a git repository monitored via webhook", func() {
		It("updates the deployment", func() {
			By("checking the pod exists")
			Eventually(func() string {
				out, _ := k.Namespace("default").Get("pods")
				return out
			}).Should(ContainSubstring("sleeper-"))

			By("updating the git repository")
			replace(path.Join(clonedir, "examples", "Chart.yaml"), "0.1.0", "0.2.0")
			replace(path.Join(clonedir, "examples", "templates", "deployment.yaml"), "name: sleeper", "name: newsleep")

			commit, err := gh.Update(clone)
			Expect(err).ToNot(HaveOccurred())

			By("checking for the updated commit hash in gitrepo")
			Eventually(func() string {
				out, _ := k.Get("gitrepo", "gitrepo-test", "-o", "yaml")
				return out
			}).Should(ContainSubstring("commit: " + commit))

			By("checking the deployment's new name")
			Eventually(func() string {
				out, _ := k.Namespace("default").Get("deployments")
				return out
			}).Should(ContainSubstring("newsleep"))
		})
	})
})

// replace replaces string s with r in the file located at path. That file must exist and be writable.
func replace(path string, s string, r string) {
	b, err := os.ReadFile(path)
	Expect(err).ToNot(HaveOccurred())

	b = bytes.ReplaceAll(b, []byte(s), []byte(r))

	err = os.WriteFile(path, b, 0644)
	Expect(err).ToNot(HaveOccurred())
}

// buildGitHostname builds the hostname of a cluster-local git repo from the provided namespace.
func buildGitHostname(ns string) (string, error) {
	if ns == "" {
		return "", errors.New("namespace is required")
	}

	return fmt.Sprintf("git-service.%s.svc.cluster.local", ns), nil
}

// getExternalRepoIP retrieves the external IP where our local git server can be reached, based on the provided port and
// repo name.
func getExternalRepoIP(port int, repoName string) (string, error) {
	systemk := env.Kubectl.Namespace(env.Namespace)

	externalIP, err := systemk.Get("service", "git-service", "-o", "jsonpath={.status.loadBalancer.ingress[0].ip}")
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("http://%s:%d/%s", externalIP, port, repoName), nil
}
