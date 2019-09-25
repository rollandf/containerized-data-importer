package utils

import (
	"fmt"
	"strings"
	"time"

	"github.com/onsi/ginkgo"
	k8sv1 "k8s.io/api/core/v1"
	v1 "k8s.io/api/core/v1"
	k8serrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/client-go/kubernetes"
)

const (
	// PodWaitForTime is the time to wait for Pod operations to complete
	PodWaitForTime = defaultPollPeriod

	podCreateTime = defaultPollPeriod
	podDeleteTime = defaultPollPeriod
)

// CreateExecutorPodWithPVC creates a Pod with the passed in PVC mounted under /pvc. You can then use the executor utilities to
// run commands against the PVC through this Pod.
func CreateExecutorPodWithPVC(clientSet *kubernetes.Clientset, podName, namespace string, pvc *k8sv1.PersistentVolumeClaim) (*k8sv1.Pod, error) {
	return CreatePod(clientSet, namespace, newExecutorPodWithPVC(podName, pvc))
}

// CreatePod calls the Kubernetes API to create a Pod
func CreatePod(clientSet *kubernetes.Clientset, namespace string, podDef *k8sv1.Pod) (*k8sv1.Pod, error) {
	var pod *k8sv1.Pod
	err := wait.PollImmediate(2*time.Second, podCreateTime, func() (bool, error) {
		var err error
		pod, err = clientSet.CoreV1().Pods(namespace).Create(podDef)
		if err != nil {
			return false, err
		}
		return true, nil
	})
	return pod, err
}

// DeletePod deletes the passed in Pod from the passed in Namespace
func DeletePod(clientSet *kubernetes.Clientset, pod *k8sv1.Pod, namespace string) error {
	return wait.PollImmediate(2*time.Second, podDeleteTime, func() (bool, error) {
		err := clientSet.CoreV1().Pods(namespace).Delete(pod.GetName(), &metav1.DeleteOptions{})
		if err != nil {
			return false, nil
		}
		return true, nil
	})
}

// NewPodWithPVC creates a new pod that mounts the given PVC
func NewPodWithPVC(podName, cmd string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	pod := &k8sv1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name: podName,
			Labels: map[string]string{
				"cdi.kubevirt.io/testing": podName,
			},
		},
		Spec: k8sv1.PodSpec{
			RestartPolicy: k8sv1.RestartPolicyNever,
			Containers: []k8sv1.Container{
				{
					Name:    "runner",
					Image:   "registry.fedoraproject.org/fedora-minimal:29",
					Command: []string{"/bin/sh", "-c", cmd},
				},
			},
			Volumes: []k8sv1.Volume{
				{
					Name: pvc.GetName(),
					VolumeSource: k8sv1.VolumeSource{
						PersistentVolumeClaim: &k8sv1.PersistentVolumeClaimVolumeSource{
							ClaimName: pvc.GetName(),
						},
					},
				},
			},
		},
	}

	volumeMode := pvc.Spec.VolumeMode
	if volumeMode != nil && *volumeMode == v1.PersistentVolumeBlock {
		pod.Spec.Containers[0].VolumeDevices = addVolumeDevices(pvc)
	} else {
		pod.Spec.Containers[0].VolumeMounts = addVolumeMounts(pvc)
	}
	return pod
}

func addVolumeDevices(pvc *k8sv1.PersistentVolumeClaim) []v1.VolumeDevice {
	volumeDevices := []v1.VolumeDevice{
		{
			Name:       pvc.GetName(),
			DevicePath: DefaultPvcMountPath,
		},
	}
	return volumeDevices
}

// this is being called for pods using PV with filesystem volume mode
func addVolumeMounts(pvc *k8sv1.PersistentVolumeClaim) []v1.VolumeMount {
	volumeMounts := []v1.VolumeMount{
		{
			Name:      pvc.GetName(),
			MountPath: DefaultPvcMountPath,
		},
	}
	return volumeMounts
}

// FindPodByPrefix finds the first pod which has the passed in prefix. Returns error if multiple pods with the same prefix are found.
func FindPodByPrefix(clientSet *kubernetes.Clientset, namespace, prefix, labelSelector string) (*k8sv1.Pod, error) {
	var result k8sv1.Pod
	var foundPod bool
	err := wait.PollImmediate(2*time.Second, podCreateTime, func() (bool, error) {
		podList, err := clientSet.CoreV1().Pods(namespace).List(metav1.ListOptions{
			LabelSelector: labelSelector,
		})
		if err == nil {
			for _, pod := range podList.Items {
				if strings.HasPrefix(pod.Name, prefix) {
					if !foundPod {
						foundPod = true
						result = pod
					} else {
						fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: First pod name %s in namespace %s\n", result.Name, result.Namespace)
						fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: Second pod name %s in namespace %s\n", pod.Name, pod.Namespace)
						return true, fmt.Errorf("Multiple pods starting with prefix %q in namespace %q", prefix, namespace)
					}
				}
			}
		}
		return foundPod, nil
	})
	if !foundPod {
		return nil, fmt.Errorf("Unable to find pod starting with prefix %s", prefix)
	}
	return &result, err
}

func newExecutorPodWithPVC(podName string, pvc *k8sv1.PersistentVolumeClaim) *k8sv1.Pod {
	return NewPodWithPVC(podName, "sleep 30; echo I am an executor pod;", pvc)
}

// WaitTimeoutForPodReady waits for the given pod to be created and ready
func WaitTimeoutForPodReady(clientSet *kubernetes.Clientset, podName, namespace string, timeout time.Duration) error {
	return WaitTimeoutForPodStatus(clientSet, podName, namespace, k8sv1.PodRunning, timeout)
}

// WaitTimeoutForPodSucceeded waits for pod to succeed
func WaitTimeoutForPodSucceeded(clientSet *kubernetes.Clientset, podName, namespace string, timeout time.Duration) error {
	return WaitTimeoutForPodStatus(clientSet, podName, namespace, k8sv1.PodSucceeded, timeout)
}

// WaitTimeoutForPodFailed waits for pod to fail
func WaitTimeoutForPodFailed(clientSet *kubernetes.Clientset, podName, namespace string, timeout time.Duration) error {
	return WaitTimeoutForPodStatus(clientSet, podName, namespace, k8sv1.PodFailed, timeout)
}

// WaitTimeoutForPodStatus waits for the given pod to be created and have a expected status
func WaitTimeoutForPodStatus(clientSet *kubernetes.Clientset, podName, namespace string, status k8sv1.PodPhase, timeout time.Duration) error {
	return wait.PollImmediate(2*time.Second, timeout, podStatus(clientSet, podName, namespace, status))
}

func podStatus(clientSet *kubernetes.Clientset, podName, namespace string, status k8sv1.PodPhase) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: Checking POD phase: %s\n", string(pod.Status.Phase))
		switch pod.Status.Phase {
		case status:
			return true, nil
		}
		return false, nil
	}
}

// PodGetNode returns the node on which a given pod is executing
func PodGetNode(clientSet *kubernetes.Clientset, podName, namespace string) (string, error) {
	pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
	if err != nil {
		return "", err
	}
	return pod.Spec.NodeName, nil
}

// WaitPodDeleted waits fo a pod to no longer exist
// returns whether the pod is deleted along with any error
func WaitPodDeleted(clientSet *kubernetes.Clientset, podName, namespace string, timeout time.Duration) (bool, error) {
	var result bool
	err := wait.PollImmediate(2*time.Second, timeout, func() (bool, error) {
		_, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				result = true
				return true, nil
			}
			return false, err
		}
		return false, nil
	})
	return result, err
}

// IsExpectedNode waits to check if the specified pod is schedule on the specified node
func IsExpectedNode(clientSet *kubernetes.Clientset, nodeName, podName, namespace string, timeout time.Duration) error {
	return wait.PollImmediate(2*time.Second, timeout, isExpectedNode(clientSet, nodeName, podName, namespace))
}

// returns true is the specified pod running on the specified nodeName. Otherwise returns false
func isExpectedNode(clientSet *kubernetes.Clientset, nodeName, podName, namespace string) wait.ConditionFunc {
	return func() (bool, error) {
		pod, err := clientSet.CoreV1().Pods(namespace).Get(podName, metav1.GetOptions{})
		if err != nil {
			if k8serrors.IsNotFound(err) {
				return false, nil
			}
			return false, err
		}
		fmt.Fprintf(ginkgo.GinkgoWriter, "INFO: Checking Node name: %s\n", string(pod.Spec.NodeName))
		if pod.Spec.NodeName == nodeName {
			return true, nil
		}
		return false, nil
	}
}

// GetSchedulableNode return a schedulable node from a nodes list
func GetSchedulableNode(nodes *v1.NodeList) *string {
	for _, node := range nodes.Items {
		if node.Spec.Taints == nil {
			return &node.Name
		}
		schedulableNode := true
		for _, taint := range node.Spec.Taints {
			if taint.Effect == "NoSchedule" {
				schedulableNode = false
				break
			}
		}
		if schedulableNode {
			return &node.Name
		}
	}
	return nil
}
