/*
ICOS Telemetruum Leaf Exporter
Copyright © 2022 - 2025 Engineering Ingegneria Informatica S.p.A.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.

This work has received funding from the European Union's HORIZON research
and innovation programme under grant agreement No. 101070177.
*/

package modules

import (
	"context"
	"crypto/x509"
	"encoding/pem"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"telemetruum/leaf-exporter/cli"
	"time"

	"github.com/rs/zerolog"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/tools/leaderelection"
	"k8s.io/client-go/tools/leaderelection/resourcelock"
)

var (
	kubeConfig            = cli.Serve.Flag("kube-config", "Kubernetes Configuration file").String()
	ocmWorkAgentPodPrefix = cli.Serve.Flag("ocm-pod-prefix", "Prefix to match the OCM Work Agent pod").Default("klusterlet-work-agent-").String()

	m1 = regexp.MustCompile(`(.+).icos.eu/(.+)`)
	m2 = regexp.MustCompile(`icos.eu/(.+)`)
)

type KubernetesProvider struct {
	BaseProvider
	KubernetesClient        *kubernetes.Clientset
	iAmTheLeader            bool
	Id                      string
	nodeName                string
	namespace               string
	myPodName               string
	leaderElectionListeners []func(bool)
}

func InizializeKubernetesProvider(logger zerolog.Logger) (*KubernetesProvider, error) {

	var clientset *kubernetes.Clientset

	if *kubeConfig != "" {

		cfg, err := clientcmd.BuildConfigFromFlags("", *kubeConfig)
		if err != nil {
			log.Fatalf("kubeConfig specified but an error occurred during initialization: '%s'. Fix the error or disable Kuberentes provider with '--no-kubernetes'", err)
		}

		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Fatalf("kubeConfig specified but an error occurred during initialization: '%s'. Fix the error or disable Kuberentes provider with '--no-kubernetes'", err)
		}

	} else {

		cfg, err := rest.InClusterConfig()
		if err == rest.ErrNotInCluster {
			// Returning nil if we did not found a Kubernetes configuration. The provider will be disabled
			// For all other execution paths (where a kubernetes configuration has been found) we fail if somegthing
			// goes wrong
			return nil, err
		}

		if err != nil {
			log.Fatalf("InCluster Kubernetes config found but an error occurred during initialization: '%s'. Fix the error or disable Kuberentes provider with '--no-kubernetes'", err)
		}
		clientset, err = kubernetes.NewForConfig(cfg)
		if err != nil {
			log.Fatalf("InCluster Kubernetes config found but an error occurred during initialization: '%s'. Fix the error or disable Kuberentes provider with '--no-kubernetes'", err)
		}
	}

	nodeName := os.Getenv("NODE_NAME")
	if nodeName == "" {
		log.Fatalf("NODE_NAME env variable must be provided when Kubernetes provider is used. Fix the error or disable Kuberentes provider with '--no-kubernetes'")
	}

	namespace := os.Getenv("NAMESPACE")
	if namespace == "" {
		log.Fatalf("NAMESPACE env variable must be provided when Kubernetes provider is used. Fix the error or disable Kuberentes provider with '--no-kubernetes'")
	}

	podName := os.Getenv("POD_NAME")
	if podName == "" {
		log.Fatalf("POD_NAME env variable must be provided when Kubernetes provider is used. Fix the error or disable Kuberentes provider with '--no-kubernetes'")
	}

	kubeSystemNS, err := clientset.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})
	if err != nil {
		log.Fatalf("Error testing Kubernetes configuration. Error reading cluster id from 'kube-system' namespace: %s. Fix the error or disable Kuberentes provider with '--no-kubernetes'", err)
	}

	logger.Debug().Msgf("Initialized Kubernetes Provider for cluster with Id: %s", string(kubeSystemNS.UID))

	return &KubernetesProvider{Id: string(kubeSystemNS.UID), KubernetesClient: clientset, BaseProvider: BaseProvider{Logger: logger},
		nodeName: nodeName, namespace: namespace, myPodName: podName}, nil
}

func (kp *KubernetesProvider) Start(ctx context.Context, wg *sync.WaitGroup) {
	kp.leaderElectionControlLoop(ctx)
}

func (kp *KubernetesProvider) ProvideLabels(ctx context.Context, hlc *HostLabelsCollector) {
	nodes, _ := kp.KubernetesClient.
		CoreV1().
		Nodes().
		List(ctx, metav1.ListOptions{
			FieldSelector: "metadata.name=" + kp.nodeName,
		})
	if len(nodes.Items) != 1 {
		kp.Logger.Error().Msgf("Error getting curret node. Expected exactly one result, got %d", len(nodes.Items))
	} else {
		node := nodes.Items[0]
		for k, v := range node.ObjectMeta.Labels {
			if m2.MatchString(k) {
				newK := m2.ReplaceAllString(k, "$1")
				hlc.Labels[newK] = v
			}
		}
	}
}

func (kp *KubernetesProvider) ProvideRuntimeOrchestartorInfo(ctx context.Context, ric *RuntimeInfoCollector) {
	ric.Type = "Kubernetes"
	version, err := kp.KubernetesClient.ServerVersion()
	if err != nil {
		kp.Logger.Error().Msg("Error getting Kubernetes version")
	}
	ric.Version = version.String()
	ric.NodeName = os.Getenv("NODE_NAME")

	// using kube-system namespace since it is the same used by the OTEL Collector for the k8s.cluster.id attribute
	kubesystemNS, _ := kp.KubernetesClient.CoreV1().Namespaces().Get(context.TODO(), "kube-system", metav1.GetOptions{})

	ric.ClusterId = string(kubesystemNS.GetUID())
}

func k8sStatus2WIStatus(k8sStatus string) WorkloadStatus {
	switch k8sStatus {
	case "Pending":
		return Pending
	case "Running":
		return Running
	case "Succeeded":
		return Exited
	case "Failed":
		return Failed
	case "Unknown":
		return Unknown
	}
	return Unknown
}

func (kp *KubernetesProvider) ProvideWorkloadInfo(ctx context.Context, c *WorkloadInfoCollector) {
	kp.Logger.Debug().Msgf("Listing pods in node \"%s\"", kp.nodeName)

	pods, _ := kp.KubernetesClient.
		CoreV1().
		Pods("").
		List(ctx, metav1.ListOptions{
			FieldSelector: "spec.nodeName=" + kp.nodeName,
		})

	kp.Logger.Debug().Msgf("Found %d pods", len(pods.Items))
	res := []*WorkloadInfo{}
	for _, p := range pods.Items {
		wi := &WorkloadInfo{Name: p.ObjectMeta.Namespace + "__" + p.ObjectMeta.Name, Id: string(p.ObjectMeta.UID), Annotations: map[string]string{}, Status: k8sStatus2WIStatus(string(p.Status.Phase))}
		res = append(res, wi)

		for k, v := range p.ObjectMeta.Annotations {
			if m1.MatchString(k) {
				newK := m1.ReplaceAllString(k, "icos.$1.$2")
				wi.Annotations[newK] = v
			}
		}
	}

	c.RunningWorkloads = res
	//c.ClusterId = kp.Id
}

func (kp *KubernetesProvider) ProvideNuvlaOrchestratorInfo(ctx context.Context, oic *OrchInfoCollector) {

	nodeName := os.Getenv("NODE_NAME")

	nuvlaEdgePodList, _ := kp.KubernetesClient.CoreV1().Pods("").List(context.TODO(),
		metav1.ListOptions{
			LabelSelector: "app.kubernetes.io/name=nuvlaedge,component=agent"})

	if len(nuvlaEdgePodList.Items) == 0 {
		kp.Logger.Warn().Msgf("No NuvlaEdge pod found.\n")
		return
	}

	if len(nuvlaEdgePodList.Items) > 1 {
		kp.Logger.Warn().Msgf("More than on pod found for NuvlaEdge. This should never happen. Do not extract Nuvla info.\n")
		return
	}

	nuvlaEdgePod := nuvlaEdgePodList.Items[0]
	kp.Logger.Debug().Msgf("NuvlaEdge pod found: %s", nuvlaEdgePod.ObjectMeta.Name)

	if nodeName != nuvlaEdgePod.Spec.NodeName {
		kp.Logger.Warn().Msg("We are in a different node from the one NuvlaEdge is running. Not extracting info from Nuvla context file\n")
		return
	}

	nuvlaContextFile := filepath.Join(*cli.PathRootFs, fmt.Sprintf("/var/lib/nuvlaedge/%s/data/nuvlaedge_session.json", nuvlaEdgePod.ObjectMeta.Namespace))

	if _, err := os.Stat(nuvlaContextFile); !os.IsNotExist(err) {
		kp.Logger.Debug().Msgf("Nuvla Context file found at %s", nuvlaContextFile)
		CommonProvideNuvlaOrchestratorInfo(ctx, nuvlaContextFile, oic, kp.Logger)
	} else {
		kp.Logger.Warn().Msgf("Nuvla Context file not found at %s", nuvlaContextFile)
	}
}

func (kp *KubernetesProvider) ProvideOCMOrchInfo(ctx context.Context, c *OrchInfoCollector) {

	if !kp.iAmTheLeader {
		return
	}

	kp.Logger.Debug().Msgf("Searching OCM Agent pod (with prefix '%s')...", *ocmWorkAgentPodPrefix)

	pods, _ := kp.KubernetesClient.CoreV1().Pods("").List(context.TODO(), metav1.ListOptions{})
	for _, p := range pods.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, *ocmWorkAgentPodPrefix) {
			kp.Logger.Debug().Msgf("OCM Agent pod found: %s", p.ObjectMeta.Name)

			ocm_agent_name := ""
			ocm_agent_id := ""
			for _, a := range p.Spec.Containers[0].Args {
				if strings.HasPrefix(a, "--spoke-cluster-name") {
					ocm_agent_name = a[21:]
				}
				if strings.HasPrefix(a, "--agent-id") {
					ocm_agent_id = a[11:]
				}

			}

			if ocm_agent_name != "" && ocm_agent_id != "" {
				c.Type = "ocm"
				c.AgentName = ocm_agent_name
				c.AgentId = ocm_agent_id
			} else {
				kp.Logger.Warn().Msgf("Couldn't determine OCM Agent Id and/or Name from the container args: \"%s\"", p.Spec.Containers[0].Args)
			}
		}
	}
}

func (kp *KubernetesProvider) ProvideVNetInfo(ctx context.Context, c *VNetInfoCollector) {

	if !kp.iAmTheLeader {
		return
	}

	clusterLinkNamespace := "clusterlink-system"

	kp.Logger.Debug().Msg("Getting Virtual Network Provider")

	_, err := kp.KubernetesClient.CoreV1().Namespaces().Get(context.TODO(), clusterLinkNamespace, metav1.GetOptions{})

	if err != nil {
		kp.Logger.Debug().Msgf("ClusterLink namespace not found: %s", err.Error())
		return
	}

	pods, _ := kp.KubernetesClient.CoreV1().Pods(clusterLinkNamespace).List(context.TODO(), metav1.ListOptions{})
	clPodRunning := false
	for _, p := range pods.Items {
		if strings.HasPrefix(p.ObjectMeta.Name, "cl-controlplane-") && p.Status.Phase == "Running" {
			clPodRunning = true
			kp.Logger.Debug().Msgf("ClusterLink pods running: %s", p.ObjectMeta.Name)
		}
	}

	if !clPodRunning {
		kp.Logger.Debug().Msg("ClusterLink pods not running")
		return
	}

	secret, err := kp.KubernetesClient.CoreV1().Secrets(clusterLinkNamespace).Get(context.TODO(), "cl-peer", metav1.GetOptions{})

	if err != nil {
		kp.Logger.Debug().Msgf("ClusterLink secret \"cl-peer\" not found: %s", err.Error())
		return
	}

	block, _ := pem.Decode(secret.Data["cert.pem"])
	cert, err := x509.ParseCertificate(block.Bytes)
	if err != nil {
		kp.Logger.Debug().Msgf("Error parsing cert data: %s", err.Error())
		return
	}

	c.Provider = "ClusterLink"
	c.Name = cert.Issuer.CommonName
	c.Node = cert.Subject.CommonName

	kp.Logger.Info().Msgf("ClusterLink node name: %s", cert.Subject.CommonName)
}

func (kp *KubernetesProvider) OnLeaderElected(dp func(bool)) {
	kp.leaderElectionListeners = append(kp.leaderElectionListeners, dp)
}

func (kp *KubernetesProvider) leaderElectionControlLoop(ctx context.Context) {

	leaseName := "telemetruum-leaf-exporter"
	leasePrefix := os.Getenv("TLUM_K8S_LEASE_NAME_PREFIX")
	if leasePrefix != "" {
		kp.Logger.Debug().Msgf("[kubernetes] Using prefix \"%s\" in kubernetes lease name set by the TLUM_K8S_LEASE_NAME_PREFIX env variable", leasePrefix)
		leaseName = leasePrefix + "-" + leaseName
	}

	go func() {

		lock := &resourcelock.LeaseLock{
			LeaseMeta: metav1.ObjectMeta{
				Name:      leaseName,
				Namespace: kp.namespace,
			},
			Client: kp.KubernetesClient.CoordinationV1(),
			LockConfig: resourcelock.ResourceLockConfig{
				Identity: kp.myPodName,
			},
		}

		leaderelection.RunOrDie(ctx, leaderelection.LeaderElectionConfig{
			Lock:            lock,
			ReleaseOnCancel: true,
			LeaseDuration:   60 * time.Second,
			RenewDeadline:   15 * time.Second,
			RetryPeriod:     5 * time.Second,
			Callbacks: leaderelection.LeaderCallbacks{
				OnStartedLeading: func(ctx context.Context) {
					kp.Logger.Info().Msgf("[kubernetes] We are the new leader. Starting collecting metrics")
					kp.iAmTheLeader = true
					for _, listener := range kp.leaderElectionListeners {
						listener(true)
					}
				},
				OnStoppedLeading: func() {
					kp.Logger.Info().Msg("[kubernetes] Stopping leading. Stopping collecting metrics and exiting")
					kp.iAmTheLeader = false
					for _, listener := range kp.leaderElectionListeners {
						listener(false)
					}
					os.Exit(0)
				},
				OnNewLeader: func(identity string) {
					kp.Logger.Info().Msgf("[kubernetes] New leader is \"%s\"", identity)
				},
			},
		})

	}()
}
