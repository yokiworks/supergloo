package e2e

import (
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/onsi/gomega/gexec"
	"github.com/solo-io/gloo/pkg/log"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/solo-io/solo-kit/pkg/utils/kubeutils"
	"github.com/solo-io/solo-kit/test/helpers"
	testsetup "github.com/solo-io/solo-kit/test/setup"
	"github.com/solo-io/supergloo/pkg/api/v1"
	"github.com/solo-io/supergloo/pkg/setup"
	"os"
	"os/exec"
	"time"
)

var _ = Describe("istio routing E2e", func() {
	var namespace string
	BeforeEach(func() {
		namespace = helpers.RandString(8)
		err := testsetup.SetupKubeForTest(namespace)
		Expect(err).NotTo(HaveOccurred())
	})
	AfterEach(func() {
		testsetup.TeardownKube(namespace)
	})

	It("works", func() {
		go setup.Main()

		// start discovery
		cmd := exec.Command(PathToUds, "--namespace", namespace)
		cmd.Env = os.Environ()
		_, err := gexec.Start(cmd, GinkgoWriter, GinkgoWriter)
		Expect(err).NotTo(HaveOccurred())

		meshes, routingRules, err := run()
		Expect(err).NotTo(HaveOccurred())
		meta := core.Metadata{Name: "my-istio", Namespace: namespace}
		meshes.Delete(meta.Namespace, meta.Name, clients.DeleteOpts{})
		m1, err := meshes.Write(&v1.Mesh{
			Metadata: meta,
			MeshType: &v1.Mesh_Istio{
				Istio: &v1.Istio{
					WatchNamespaces: []string{namespace},
				},
			},
			Encryption: &v1.Encryption{
				TlsEnabled: true,
			},
			//Encryption: &v1.Encryption{TlsEnabled: true},
		}, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(m1).NotTo(BeNil())
		rrMeta := core.Metadata{Name: "my-istio-rr", Namespace: namespace}
		routingRules.Delete(rrMeta.Namespace, rrMeta.Name, clients.DeleteOpts{})

		ref := m1.Metadata.Ref()
		rr1, err := routingRules.Write(&v1.RoutingRule{
			Metadata:   rrMeta,
			TargetMesh: &ref,
			Destinations: []*core.ResourceRef{{
				Name:      "default-reviews-9080",
				Namespace: "gloo-system",
			}},
			TrafficShifting: &v1.TrafficShifting{
				Destinations: []*v1.WeightedDestination{
					{
						Upstream: &core.ResourceRef{
							Name:      "default-reviews-v1-9080",
							Namespace: "gloo-system",
						},
						Weight: 100,
					},
				},
			},
		}, clients.WriteOpts{})
		Expect(err).NotTo(HaveOccurred())
		Expect(rr1).NotTo(BeNil())
		for {
			select {
			case <-time.After(time.Second):
				log.Printf("waiting 1s")
			}
		}
	})
})

func run() (v1.MeshClient, v1.RoutingRuleClient, error) {
	kubeCache := kube.NewKubeCache()
	restConfig, err := kubeutils.GetConfig("", "")
	if err != nil {
		return nil, nil, err
	}
	meshClient, err := v1.NewMeshClient(&factory.KubeResourceClientFactory{
		Crd:         v1.MeshCrd,
		Cfg:         restConfig,
		SharedCache: kubeCache,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := meshClient.Register(); err != nil {
		return nil, nil, err
	}

	routingRuleClient, err := v1.NewRoutingRuleClient(&factory.KubeResourceClientFactory{
		Crd:         v1.RoutingRuleCrd,
		Cfg:         restConfig,
		SharedCache: kubeCache,
	})
	if err != nil {
		return nil, nil, err
	}
	if err := routingRuleClient.Register(); err != nil {
		return nil, nil, err
	}
	return meshClient, routingRuleClient, nil
}
