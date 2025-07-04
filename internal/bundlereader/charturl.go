package bundlereader

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/Masterminds/semver/v3"
	fleet "github.com/rancher/fleet/pkg/apis/fleet.cattle.io/v1alpha1"
	"helm.sh/helm/v3/pkg/repo"
	"sigs.k8s.io/yaml"

	"oras.land/oras-go/v2/registry"
	"oras.land/oras-go/v2/registry/remote"
	"oras.land/oras-go/v2/registry/remote/auth"
	"oras.land/oras-go/v2/registry/remote/retry"
)

// ChartVersion returns the version of the helm chart from a helm repo server, by
// inspecting the repo's index.yaml
func ChartVersion(location fleet.HelmOptions, auth Auth) (string, error) {
	if hasOCIURL.MatchString(location.Repo) {
		tag, err := getOCITag(location, auth)

		if err != nil {
			return "", fmt.Errorf("could not find tag matching constraint %q in registry %s", location.Version, location.Repo)
		}

		return tag, nil
	}

	if location.Repo == "" {
		return location.Version, nil
	}

	if !strings.HasSuffix(location.Repo, "/") {
		location.Repo = location.Repo + "/"
	}

	chart, err := getHelmChartVersion(location, auth)
	if err != nil {
		return "", err
	}

	if len(chart.URLs) == 0 {
		return "", fmt.Errorf("no URLs found for chart %s %s at %s", chart.Name, chart.Version, location.Repo)
	}

	return chart.Version, nil
}

// chartURL returns the URL to the helm chart from a helm repo server, by
// inspecting the repo's index.yaml
func chartURL(location fleet.HelmOptions, auth Auth, isHelmOps bool) (string, error) {
	OCIField := location.Chart
	if isHelmOps {
		OCIField = location.Repo
	}

	if hasOCIURL.MatchString(OCIField) {
		return OCIField, nil
	}

	if location.Repo == "" {
		return location.Chart, nil
	}

	if !strings.HasSuffix(location.Repo, "/") {
		location.Repo = location.Repo + "/"
	}

	chart, err := getHelmChartVersion(location, auth)
	if err != nil {
		return "", err
	}

	if len(chart.URLs) == 0 {
		return "", fmt.Errorf("no URLs found for chart %s %s at %s", chart.Name, chart.Version, location.Repo)
	}

	chartURL, err := url.Parse(chart.URLs[0])
	if err != nil {
		return "", err
	}

	if chartURL.IsAbs() {
		return chart.URLs[0], nil
	}

	repoURL, err := url.Parse(location.Repo)
	if err != nil {
		return "", err
	}

	return repoURL.ResolveReference(chartURL).String(), nil
}

// getHelmChartVersion returns the ChartVersion struct with the information to the given location
// using the given authentication configuration
func getHelmChartVersion(location fleet.HelmOptions, auth Auth) (*repo.ChartVersion, error) {
	request, err := http.NewRequest("GET", location.Repo+"index.yaml", nil)
	if err != nil {
		return nil, err
	}

	if auth.Username != "" && auth.Password != "" {
		request.SetBasicAuth(auth.Username, auth.Password)
	}
	client := &http.Client{}
	if auth.CABundle != nil {
		pool, err := x509.SystemCertPool()
		if err != nil {
			pool = x509.NewCertPool()
		}
		pool.AppendCertsFromPEM(auth.CABundle)
		transport := http.DefaultTransport.(*http.Transport).Clone()
		transport.TLSClientConfig = &tls.Config{
			RootCAs:            pool,
			MinVersion:         tls.VersionTLS12,
			InsecureSkipVerify: auth.InsecureSkipVerify, // nolint:gosec
		}
		client.Transport = transport
	} else {
		if auth.InsecureSkipVerify {
			transport := http.DefaultTransport.(*http.Transport).Clone()
			transport.TLSClientConfig = &tls.Config{
				InsecureSkipVerify: auth.InsecureSkipVerify, // nolint:gosec
			}
			client.Transport = transport
		}
	}

	resp, err := client.Do(request)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != 200 {
		return nil, fmt.Errorf("failed to read helm repo from %s, error code: %v, response body: %s", location.Repo+"index.yaml", resp.StatusCode, bytes)
	}

	repo := &repo.IndexFile{}
	if err := yaml.Unmarshal(bytes, repo); err != nil {
		return nil, err
	}

	repo.SortEntries()

	chart, err := repo.Get(location.Chart, location.Version)
	if err != nil {
		return nil, err
	}

	return chart, nil
}

func getOCITag(location fleet.HelmOptions, a Auth) (string, error) {
	// TODO handle auth
	r, err := remote.NewRepository(location.Repo)
	if err != nil {
		return "", fmt.Errorf("failed to create OCI client: %w", err)
	}

	var hc *http.Client
	if !a.InsecureSkipVerify {
		hc = retry.DefaultClient
	} else {
		hc = &http.Client{
			Transport: &http.Transport{
				TLSClientConfig: &tls.Config{InsecureSkipVerify: true}, // #nosec G402
			},
		}
	}

	client := &auth.Client{
		Client: hc,
		Cache:  auth.NewCache(),
	}
	if a.Username != "" {
		cred := auth.Credential{
			Username: a.Username,
			Password: a.Password,
		}
		client.Credential = func(ctx context.Context, s string) (auth.Credential, error) {
			return cred, nil
		}
	}

	r.Client = client

	availableTags, err := registry.Tags(context.TODO(), r)
	if err != nil {
		panic(err)
	}

	// TODO sort tags: https://github.com/Masterminds/semver?tab=readme-ov-file#sorting-semantic-versions

	constraint, err := semver.NewConstraint(location.Version)
	if err != nil {
		return "", fmt.Errorf("failed to compute version constraint from version %q", location.Version)
	}

	var tagToResolve string

	for _, tag := range availableTags {
		// check for exact match before trying something more involved.
		if len(location.Version) > 0 && location.Version == tag {
			tagToResolve = tag
		}

		test, err := semver.NewVersion(tag)
		if err != nil {
			continue
		}

		if constraint.Check(test) {
			tagToResolve = tag
		}
	}

	_, err = r.Resolve(context.TODO(), tagToResolve)
	if err != nil {
		return "", fmt.Errorf("failed to resolve tag %q", tagToResolve)
	}

	return tagToResolve, nil
}
