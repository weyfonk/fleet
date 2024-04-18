//go:generate mockgen --build_flags=--mod=mod -destination=../../mocks/getter_mock.go -package=mocks github.com/rancher/fleet/internal/cmd/cli/apply Getter
//go:generate mockgen --build_flags=--mod=mod -destination=../../mocks/fleet_controller_mock.go -package=mocks -mock_names=Interface=FleetInterface github.com/rancher/fleet/pkg/generated/controllers/fleet.cattle.io/v1alpha1 Interface
//go:generate mockgen --build_flags=--mod=mod -destination=../../mocks/core_controller_mock.go -package=mocks -mock_names=Interface=CoreInterface github.com/rancher/wrangler/v2/pkg/generated/controllers/core/v1 Interface
//go:generate mockgen --build_flags=--mod=mod -destination=../../mocks/rbac_controller_mock.go -package=mocks -mock_names=Interface=RBACInterface github.com/rancher/wrangler/v2/pkg/generated/controllers/rbac/v1 Interface
//go:generate mockgen --build_flags=--mod=mod -destination=../../mocks/apply_mock.go -package=mocks github.com/rancher/wrangler/v2/pkg/apply Apply
package apply

import (
	"github.com/golang/mock/gomock"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	"github.com/rancher/fleet/integrationtests/cli"
	"github.com/rancher/fleet/integrationtests/mocks"
	"github.com/rancher/fleet/internal/client"
	"github.com/rancher/fleet/internal/cmd/cli/apply"
)

var _ = Describe("Fleet apply online", Ordered, func() {
	var (
		dirs    []string
		name    string
		options apply.Options
	)

	JustBeforeEach(func() {
		ctrl := gomock.NewController(GinkgoT())
		getter := mocks.NewMockGetter(ctrl)
		client := client.Client{
			Fleet:     mocks.NewFleetInterface(ctrl),
			Core:      mocks.NewCoreInterface(ctrl),
			RBAC:      mocks.NewRBACInterface(ctrl),
			Apply:     mocks.NewMockApply(ctrl),
			Namespace: "foo",
		}

		getter.EXPECT().GetNamespace().Return("foo")
		getter.EXPECT().Get().Return(&client, nil)

		err := fleetApplyOnline(getter, name, dirs, options)
		Expect(err).NotTo(HaveOccurred())
	})

	When("<TODO>", func() {
		BeforeEach(func() {
			name = "simple"
			dirs = []string{cli.AssetsPath + "simple"}
		})

		It("<ALSO TODO>", func() {
			_, _ = cli.GetBundleFromOutput(buf)
		})
	})

})
