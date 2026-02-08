package kube

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
)

type Client struct {
	cs kubernetes.Interface
}

func NewClient() (*Client, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		kubeconfig := filepath.Join(homedir.HomeDir(), ".kube", "config")
		config, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
		if err != nil {
			return nil, fmt.Errorf("cannot create k8s config: %w", err)
		}
	}

	cs, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("cannot create k8s client: %w", err)
	}

	return &Client{cs: cs}, nil
}

func (c *Client) FetchClusterInfo(ctx context.Context) (*ClusterInfo, error) {
	pods, err := c.cs.CoreV1().Pods(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: "app=deckhouse",
	})
	if err != nil {
		return nil, fmt.Errorf("cannot list pods: %w", err)
	}
	if len(pods.Items) == 0 {
		return nil, fmt.Errorf("no deckhouse pods found in %s", namespace)
	}

	// Pick the first running pod, or just the first one
	pod := pods.Items[0]
	for _, p := range pods.Items {
		if p.Status.Phase == "Running" {
			pod = p
			break
		}
	}

	info := &ClusterInfo{
		PodName:    pod.Name,
		PodCreated: pod.CreationTimestamp.Time,
		PodPhase:   string(pod.Status.Phase),
	}

	if len(pod.Spec.Containers) > 0 {
		info.Image = pod.Spec.Containers[0].Image
		info.Registry, info.Repository, info.Tag = parseImage(info.Image)
	}

	if len(pod.Status.ContainerStatuses) > 0 {
		imageID := pod.Status.ContainerStatuses[0].ImageID
		if idx := strings.Index(imageID, "@"); idx != -1 {
			info.RunningDigest = imageID[idx+1:]
		}
	}

	info.RegistryCreds, _ = c.fetchRegistryCreds(ctx)

	return info, nil
}

func parseImage(image string) (registry, repo, tag string) {
	if idx := strings.LastIndex(image, ":"); idx != -1 {
		tag = image[idx+1:]
		image = image[:idx]
	}
	if idx := strings.Index(image, "/"); idx != -1 {
		registry = image[:idx]
		repo = image[idx+1:]
	}
	return
}
