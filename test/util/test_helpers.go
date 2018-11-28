package util

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"k8s.io/helm/pkg/proto/hapi/release"

	"github.com/solo-io/supergloo/pkg/install/helm"

	"github.com/hashicorp/consul/api"
	. "github.com/onsi/gomega"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients"
	"github.com/solo-io/solo-kit/pkg/api/v1/clients/kube"
	"github.com/solo-io/solo-kit/pkg/api/v1/resources/core"
	"github.com/solo-io/supergloo/pkg/api/v1"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"

	"github.com/solo-io/solo-kit/pkg/api/v1/clients/factory"
	gloo "github.com/solo-io/supergloo/pkg/api/external/gloo/v1"
	istiosecret "github.com/solo-io/supergloo/pkg/api/external/istio/encryption/v1"
	istioSync "github.com/solo-io/supergloo/pkg/translator/istio"
	"k8s.io/client-go/kubernetes"

	kubecore "k8s.io/api/core/v1"
	apiexts "k8s.io/apiextensions-apiserver/pkg/client/clientset/clientset"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	kubemeta "k8s.io/apimachinery/pkg/apis/meta/v1"
	helmlib "k8s.io/helm/pkg/helm"
	helmkube "k8s.io/helm/pkg/kube"

	security "github.com/openshift/client-go/security/clientset/versioned"
	// love me google.
	_ "k8s.io/client-go/plugin/pkg/client/auth/gcp"
)

var kubeConfig *rest.Config
var kubeClient *kubernetes.Clientset
var apiExtsClient apiexts.Interface
var upstreamClient gloo.UpstreamClient

var testKey = "-----BEGIN PRIVATE KEY-----\nMIG2AgEAMBAGByqGSM49AgEGBSuBBAAiBIGeMIGbAgEBBDBoI1sMdiOTvBBdjWlS\nZ8qwNuK9xV4yKuboLZ4Sx/OBfy1eKZocxTKvnjLrHUe139uhZANiAAQMTIR56O8U\nTIqf6uUHM4i9mZYLj152up7elS06Gi6lk7IeUQDHxP0NnOnbhC7rmtOV6myLNApL\nQ92kZKg7qa8q7OY/4w1QfC4ch7zZKxjNkSIiuAx7V/lzF6FYDcqT3js=\n-----END PRIVATE KEY-----"
var TestRoot = "-----BEGIN CERTIFICATE-----\nMIIB7jCCAXUCCQC2t6Lqc2xnXDAKBggqhkjOPQQDAjBhMQswCQYDVQQGEwJVUzEW\nMBQGA1UECAwNTWFzc2FjaHVzZXR0czESMBAGA1UEBwwJQ2FtYnJpZGdlMQwwCgYD\nVQQKDANPcmcxGDAWBgNVBAMMD3d3dy5leGFtcGxlLmNvbTAeFw0xODExMTgxMzQz\nMDJaFw0xOTExMTgxMzQzMDJaMGExCzAJBgNVBAYTAlVTMRYwFAYDVQQIDA1NYXNz\nYWNodXNldHRzMRIwEAYDVQQHDAlDYW1icmlkZ2UxDDAKBgNVBAoMA09yZzEYMBYG\nA1UEAwwPd3d3LmV4YW1wbGUuY29tMHYwEAYHKoZIzj0CAQYFK4EEACIDYgAEDEyE\neejvFEyKn+rlBzOIvZmWC49edrqe3pUtOhoupZOyHlEAx8T9DZzp24Qu65rTleps\nizQKS0PdpGSoO6mvKuzmP+MNUHwuHIe82SsYzZEiIrgMe1f5cxehWA3Kk947MAoG\nCCqGSM49BAMCA2cAMGQCMCytVFc8sBdbM7DaBCz0N2ptdb0T7LFFfxDTzn4gjiDq\nVCd/3dct21TUWsthKXF2VgIwXEMI5EQiJ5kjR/y1KNBC9b4wfDiKRvG33jYe9gn6\ntzXUS00SoqG9D27/7aK71/xv\n-----END CERTIFICATE-----"
var testCertChain = ""

func GetKubeConfig() *rest.Config {
	if kubeConfig != nil {
		return kubeConfig
	}
	kubeconfigPath := filepath.Join(os.Getenv("HOME"), ".kube", "config")
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfigPath)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	kubeConfig = cfg
	return cfg
}

func GetKubeClient() *kubernetes.Clientset {
	if kubeClient != nil {
		return kubeClient
	}
	cfg := GetKubeConfig()
	client, err := kubernetes.NewForConfig(cfg)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	kubeClient = client
	return client
}

func GetApiExtsClient() apiexts.Interface {
	if apiExtsClient != nil {
		return apiExtsClient
	}
	cfg := GetKubeConfig()
	client, err := apiexts.NewForConfig(cfg)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	apiExtsClient = client
	return client
}

func GetSecurityClient() *security.Clientset {
	securityClient, err := security.NewForConfig(GetKubeConfig())
	ExpectWithOffset(1, err).To(BeNil())
	return securityClient
}

func GetSecretClient() istiosecret.IstioCacertsSecretClient {
	secretClient, err := istiosecret.NewIstioCacertsSecretClient(&factory.KubeSecretClientFactory{
		Clientset: GetKubeClient(),
	})
	ExpectWithOffset(1, err).Should(BeNil())
	err = secretClient.Register()
	ExpectWithOffset(1, err).Should(BeNil())
	return secretClient
}

func TryCreateNamespace(namespace string) {
	client := GetKubeClient()
	resource := &kubecore.Namespace{
		ObjectMeta: kubemeta.ObjectMeta{
			Name: namespace,
		},
	}
	_, err := client.CoreV1().Namespaces().Create(resource)
	if err != nil {
		ExpectWithOffset(1, apierrors.IsAlreadyExists(err)).To(BeTrue())
	}
}

func TerminateNamespace(namespace string) {
	client := GetKubeClient()
	gracePeriod := int64(0)
	deleteOptions := &kubemeta.DeleteOptions{
		GracePeriodSeconds: &gracePeriod,
	}
	client.CoreV1().Pods(namespace).DeleteCollection(deleteOptions, kubemeta.ListOptions{})
	client.CoreV1().Namespaces().Delete(namespace, deleteOptions)
}

func TerminateNamespaceBlocking(namespace string) {
	TerminateNamespace(namespace)
	WaitForTerminatedNamespace(namespace)
}

func WaitForTerminatedNamespace(namespace string) {
	client := GetKubeClient()
	EventuallyWithOffset(1, func() error {
		_, err := client.CoreV1().Namespaces().Get(namespace, kubemeta.GetOptions{})
		return err
	}, "120s", "1s").ShouldNot(BeNil()) // will be non-nil when NS is gone
}

func WaitForAvailablePodsWithTimeout(namespace string, timeout string) int {
	// use helper function so that stack offset is consistent
	return waitForAvailablePodsWithTimeout(namespace, timeout)
}

func waitForAvailablePodsWithTimeout(namespace, timeout string) int {
	client := GetKubeClient()
	var podNum int

	EventuallyWithOffset(2, func() (bool, error) {
		podList, err := client.CoreV1().Pods(namespace).List(kubemeta.ListOptions{})
		if err != nil {
			return false, err
		}
		podNum = len(podList.Items)
		done := true
		for _, pod := range podList.Items {
			for _, condition := range pod.Status.Conditions {
				if pod.Status.Phase == kubecore.PodSucceeded {
					continue
				}
				if condition.Type == kubecore.PodReady && condition.Status != kubecore.ConditionTrue {
					done = false
				}
			}
		}
		return done, nil
	}, timeout, "1s").Should(BeTrue())
	return podNum
}

func WaitForDeletedPodsWithTimeout(namespace string, timeout string) {
	client := GetKubeClient()
	Eventually(func() bool {
		podList, err := client.CoreV1().Pods(namespace).List(kubemeta.ListOptions{})
		Expect(err).To(BeNil())
		return len(podList.Items) == 0
	}, timeout, "1s").Should(BeTrue())
}

func WaitForAvailablePods(namespace string) int {
	return waitForAvailablePodsWithTimeout(namespace, "120s")
}

func WaitForDeletedPods(namespace string) {
	WaitForDeletedPodsWithTimeout(namespace, "120s")
}

func GetMeshClient(kubeCache *kube.KubeCache) v1.MeshClient {
	meshClient, err := v1.NewMeshClient(&factory.KubeResourceClientFactory{
		Crd:         v1.MeshCrd,
		Cfg:         GetKubeConfig(),
		SharedCache: kubeCache,
	})
	ExpectWithOffset(1, err).Should(BeNil())
	err = meshClient.Register()
	ExpectWithOffset(1, err).Should(BeNil())
	return meshClient
}

func GetUpstreamClient(kubeCache *kube.KubeCache) gloo.UpstreamClient {
	if upstreamClient != nil {
		return upstreamClient
	}
	client, err := gloo.NewUpstreamClient(&factory.KubeResourceClientFactory{
		Crd:         gloo.UpstreamCrd,
		Cfg:         GetKubeConfig(),
		SharedCache: kubeCache,
	})
	ExpectWithOffset(1, err).Should(BeNil())
	err = client.Register()
	ExpectWithOffset(1, err).Should(BeNil())
	upstreamClient = client
	return upstreamClient
}

func DeleteCrb(crbName string) {
	client := GetKubeClient()
	client.RbacV1().ClusterRoleBindings().Delete(crbName, &kubemeta.DeleteOptions{})
}

func CrbDoesntExist(crbName string) bool {
	client := GetKubeClient()
	_, err := client.RbacV1().ClusterRoleBindings().Get(crbName, kubemeta.GetOptions{})
	return apierrors.IsNotFound(err)
}

func DeleteWebhookConfigIfExists(webhookName string) {
	client := GetKubeClient()
	hooks, err := client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().List(kubemeta.ListOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	for _, hook := range hooks.Items {
		if strings.HasSuffix(hook.Name, webhookName) {
			client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Delete(hook.Name, &kubemeta.DeleteOptions{})
		}
	}
}

func WebhookConfigNotFound(webhookName string) bool {
	client := GetKubeClient()
	_, err := client.AdmissionregistrationV1beta1().MutatingWebhookConfigurations().Get(webhookName, kubemeta.GetOptions{})
	return apierrors.IsNotFound(err)
}

func GetConsulServerPodName(namespace string) string {
	client := GetKubeClient()
	podList, err := client.CoreV1().Pods(namespace).List(kubemeta.ListOptions{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	for _, pod := range podList.Items {
		if strings.Contains(pod.Name, "consul-mesh-server-0") {
			return pod.Name
		}
	}
	// Should not have happened
	ExpectWithOffset(1, false).To(BeTrue())
	return ""
}

// New creates a new and initialized tunnel.
func CreateConsulTunnel(namespace string, port int) (*helmkube.Tunnel, error) {
	podName := GetConsulServerPodName(namespace)
	t := helmkube.NewTunnel(GetKubeClient().CoreV1().RESTClient(), GetKubeConfig(), namespace, podName, port)
	return t, t.ForwardPort()
}

func CreateTestSecret(namespace string, name string) (*istiosecret.IstioCacertsSecret, *core.ResourceRef) {
	secret := &istiosecret.IstioCacertsSecret{
		Metadata: core.Metadata{
			Namespace: namespace,
			Name:      name,
		},
		CaCert:    TestRoot,
		CaKey:     testKey,
		RootCert:  TestRoot,
		CertChain: testCertChain,
	}
	GetSecretClient().Delete(namespace, name, clients.DeleteOpts{})
	_, err := GetSecretClient().Write(secret, clients.WriteOpts{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ref := &core.ResourceRef{
		Namespace: namespace,
		Name:      name,
	}
	return secret, ref
}

func CheckCertMatchesConsul(consulTunnelPort int, rootCert string) {
	config := &api.Config{
		Address: fmt.Sprintf("127.0.0.1:%d", consulTunnelPort),
	}
	client, err := api.NewClient(config)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	var queryOpts api.QueryOptions
	currentConfig, _, err := client.Connect().CAGetConfig(&queryOpts)
	ExpectWithOffset(1, err).NotTo(HaveOccurred())

	currentRoot := currentConfig.Config["RootCert"]
	ExpectWithOffset(1, currentRoot).To(BeEquivalentTo(rootCert))
}

func CheckCertMatchesIstio(installNamespace string) {
	actual, err := GetSecretClient().Read(installNamespace, istioSync.CustomRootCertificateSecretName, clients.ReadOpts{})
	ExpectWithOffset(1, err).NotTo(HaveOccurred())
	ExpectWithOffset(1, actual.RootCert).Should(BeEquivalentTo(TestRoot))
	ExpectWithOffset(1, actual.CaCert).Should(BeEquivalentTo(TestRoot))
	ExpectWithOffset(1, actual.CaKey).Should(BeEquivalentTo(testKey))
	ExpectWithOffset(1, actual.CertChain).Should(BeEquivalentTo(testCertChain))
}

func UninstallHelmRelease(releaseName string) error {
	// helm install
	helmClient, err := helm.GetHelmClient(context.TODO())
	if err != nil {
		return err
	}
	_, err = helmClient.DeleteRelease(releaseName, helmlib.DeletePurge(true))
	helm.Teardown()
	return err
}

func HelmReleaseDoesntExist(releaseName string) bool {
	helmClient, err := helm.GetHelmClient(context.TODO())
	if err != nil {
		return false
	}
	statuses := []release.Status_Code{
		release.Status_UNKNOWN,
		release.Status_DEPLOYED,
		release.Status_DELETED,
		release.Status_SUPERSEDED,
		release.Status_FAILED,
		release.Status_DELETING,
		release.Status_PENDING_INSTALL,
		release.Status_PENDING_UPGRADE,
		release.Status_PENDING_ROLLBACK,
	}
	// equivalent to "--all" option
	list, err := helmClient.ListReleases(helmlib.ReleaseListStatuses(statuses))
	if err != nil {
		return false
	}
	// No releases == successfully deleted
	if list == nil {
		return true
	}
	for _, item := range list.Releases {
		if item.Name == releaseName {
			return false
		}
	}
	return true
}

func TryDeleteIstioCrds() {
	crdClient := GetApiExtsClient()
	crdList, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(kubemeta.ListOptions{})
	if err != nil {
		return
	}
	for _, crd := range crdList.Items {
		//TODO: use labels
		if strings.Contains(crd.Name, "istio.io") {
			crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().Delete(crd.Name, &kubemeta.DeleteOptions{})
		}
	}
}

func IstioCrdsDontExist() bool {
	crdClient := GetApiExtsClient()
	crdList, err := crdClient.ApiextensionsV1beta1().CustomResourceDefinitions().List(kubemeta.ListOptions{})
	if err != nil {
		return false
	}
	for _, crd := range crdList.Items {
		if strings.Contains(crd.Name, "istio.io") {
			return false
		}
	}
	return true
}

func GetUpstreamNames(upstreamClient gloo.UpstreamClient) ([]string, error) {
	ul, err := upstreamClient.List("gloo-system", clients.ListOpts{})
	if err != nil {
		return nil, err
	}
	ups := []string{}
	for _, up := range ul {
		ups = append(ups, up.Metadata.Name)
	}
	return ups, nil
}
