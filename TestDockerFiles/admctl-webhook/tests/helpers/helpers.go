package helpers

import (
	"context"
	"fmt"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
)

func WaitForPodCompletion(ctx context.Context, clientset kubernetes.Interface, namespace, podName string) error {

	attempts := 10
	for i := 0; i < attempts; i++ {
		pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
		if err != nil {
			return err
		}

		if pod.Status.Phase == corev1.PodSucceeded {
			fmt.Printf("Pod %s completed with status: %s\n", podName, pod.Status.Phase)
			return nil
		}

		if pod.Status.Phase == corev1.PodFailed {
			printPodDebugInfo(ctx, clientset, namespace, podName, pod)
			return fmt.Errorf("Pod %s failed\n", podName)
		}

		fmt.Printf("Pod %s status: %s. Waiting...\n", podName, pod.Status.Phase)
		time.Sleep(5 * time.Second)
	}

	// Timeout - print debug info
	pod, err := clientset.CoreV1().Pods(namespace).Get(ctx, podName, metav1.GetOptions{})
	if err == nil {
		printPodDebugInfo(ctx, clientset, namespace, podName, pod)
	}

	return fmt.Errorf("Pod %s did not complete in time", podName)
}

func printPodDebugInfo(ctx context.Context, clientset kubernetes.Interface, namespace, podName string, pod *corev1.Pod) {
	fmt.Printf("\n=== Debug Info for Pod %s ===\n", podName)

	// Print pod conditions
	fmt.Printf("\nPod Conditions:\n")
	for _, cond := range pod.Status.Conditions {
		fmt.Printf("  - Type: %s, Status: %s, Reason: %s, Message: %s\n",
			cond.Type, cond.Status, cond.Reason, cond.Message)
	}

	// Print init container statuses
	if len(pod.Status.InitContainerStatuses) > 0 {
		fmt.Printf("\nInit Container Statuses:\n")
		printContainerStatuses(pod.Status.InitContainerStatuses)
	}

	// Print container statuses
	fmt.Printf("\nContainer Statuses:\n")
	printContainerStatuses(pod.Status.ContainerStatuses)

	// Print pod events
	fmt.Printf("\nPod Events:\n")
	events, err := clientset.CoreV1().Events(namespace).List(ctx, metav1.ListOptions{
		FieldSelector: "involvedObject.name=" + podName,
	})
	if err == nil {
		for _, event := range events.Items {
			fmt.Printf("  - [%s] %s: %s\n", event.Type, event.Reason, event.Message)
		}
	}

	// Print pod logs
	fmt.Printf("\nPod Logs:\n")
	printContainerLogs(ctx, clientset, namespace, podName, pod.Status.InitContainerStatuses, "Init Container")
	printContainerLogs(ctx, clientset, namespace, podName, pod.Status.ContainerStatuses, "Container")

	fmt.Printf("\n=== End Debug Info ===\n\n")
}

func printContainerStatuses(statuses []corev1.ContainerStatus) {
	for _, cs := range statuses {
		fmt.Printf("  - Container: %s, Ready: %v, RestartCount: %d\n", cs.Name, cs.Ready, cs.RestartCount)
		if cs.State.Waiting != nil {
			fmt.Printf("    Waiting: Reason=%s, Message=%s\n", cs.State.Waiting.Reason, cs.State.Waiting.Message)
		}
		if cs.State.Terminated != nil {
			fmt.Printf("    Terminated: ExitCode=%d, Reason=%s, Message=%s\n",
				cs.State.Terminated.ExitCode, cs.State.Terminated.Reason, cs.State.Terminated.Message)
		}
	}
}

func printContainerLogs(ctx context.Context, clientset kubernetes.Interface, namespace, podName string, statuses []corev1.ContainerStatus, containerType string) {
	for _, cs := range statuses {
		fmt.Printf("\n--- %s: %s ---\n", containerType, cs.Name)
		logOptions := &corev1.PodLogOptions{Container: cs.Name}
		req := clientset.CoreV1().Pods(namespace).GetLogs(podName, logOptions)
		logs, err := req.DoRaw(ctx)
		if err == nil && len(logs) > 0 {
			fmt.Printf("%s\n", string(logs))
		} else {
			fmt.Printf("  No logs available or error fetching logs: %v\n", err)
		}
	}
}
