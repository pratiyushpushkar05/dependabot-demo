package main

import (
	"context"
	"fmt"
	"os"

	"github.com/uipath/service-fabric-utils/admctl-webhook/tests/helpers"
	corev1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/tools/clientcmd"
)

const (
	notWorkTaint = "DONOTWORK"
)

func GetKubeClient() (kubernetes.Interface, error) {
	// Get current mgmt cluster kubeconfig
	kubeConfigFile := os.Getenv("KUBECONFIG")
	fmt.Printf("KUBECONFIG: %s\n", kubeConfigFile)
	kubeconfig := kubeConfigFile
	cfg, err := clientcmd.BuildConfigFromFlags("", kubeconfig)
	if err != nil {
		return nil, err
	}

	clientSet, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		return nil, fmt.Errorf("error creating kubernetes client: %w", err)
	}

	return clientSet, nil
}

func testPod(name, ns string) *corev1.Pod {
	// Use TEST_IMAGE env var from CI pipeline
	testImage := os.Getenv("TEST_IMAGE")
	if testImage == "" {
		fmt.Printf("ERROR: TEST_IMAGE environment variable is not set\n")
		os.Exit(1)
	}
	fmt.Printf("Using test image: %s\n", testImage)

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      name,
			Namespace: ns,
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{
					Name:    name,
					Image:   testImage,
					Command: []string{"sh", "-c", "echo 'Test pod started' && sleep 5 && echo 'Test completed'"},
				},
			},
			RestartPolicy: corev1.RestartPolicyNever,
		},
	}
}

func basicTaintTest(ctx context.Context, name, ns string) error {
	var err error
	clientSet, err := GetKubeClient()
	if err != nil {
		return err
	}

	var nodes *v1.NodeList

	if nodes, err = clientSet.CoreV1().Nodes().List(ctx, metav1.ListOptions{}); err != nil {
		return err
	}

	// Add taint to all nodes
	for _, node := range nodes.Items {
		fmt.Printf("Node: %s\n", node.Name)
		node.Spec.Taints = append(node.Spec.Taints, v1.Taint{
			Key:    notWorkTaint,
			Effect: v1.TaintEffectNoSchedule,
		})

		if _, err = clientSet.CoreV1().Nodes().Update(ctx, &node, metav1.UpdateOptions{}); err != nil {
			return err
		}
	}

	// Create pod
	if _, err = clientSet.CoreV1().Pods(ns).Create(ctx, testPod(name, ns), metav1.CreateOptions{}); err != nil {
		return err
	}

	// Check if it is Pending state
	var pod *v1.Pod
	if pod, err = clientSet.CoreV1().Pods(ns).Get(ctx, name, metav1.GetOptions{}); err != nil {
		return nil
	}

	if pod.Status.Phase != v1.PodPending {
		return fmt.Errorf("pod is not in pending state, expected: %s, actual: %s", v1.PodPending, pod.Status.Phase)
	}

	// Add a taint which we have in our test helm chart to all nodes
	for _, node := range nodes.Items {
		fmt.Printf("Node: %s\n", node.Name)

		node.Spec.Taints = []v1.Taint{
			{
				Key:    "example-key-no-ns",
				Effect: v1.TaintEffectNoSchedule,
			},
		}

		patch := []byte(`{"spec":{"taints":[{"key":"example-key-no-ns","effect":"NoSchedule"}]}}`)
		if _, err = clientSet.CoreV1().Nodes().Patch(ctx, node.Name, types.StrategicMergePatchType, patch, metav1.PatchOptions{}); err != nil {
			return err
		}
	}

	if err := helpers.WaitForPodCompletion(ctx, clientSet, ns, name); err != nil {
		return err
	}

	if err = clientSet.CoreV1().Pods(ns).Delete(ctx, name, metav1.DeleteOptions{}); err != nil {
		return err
	}

	return nil
}

func main() {
	ctx := context.Background()
	if err := basicTaintTest(ctx, "test-pod", "default"); err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
