package kube

import "time"

const (
	namespace      = "d8-system"
	deploymentName = "deckhouse"
	secretName     = "deckhouse-registry"
)

type ClusterInfo struct {
	Image         string // full image reference (e.g., "dev-registry.deckhouse.io/sys/deckhouse-oss:pr15160")
	Registry      string // registry host (e.g., "dev-registry.deckhouse.io")
	Repository    string // repository path (e.g., "sys/deckhouse-oss")
	Tag           string // image tag (e.g., "pr15160")
	PodName       string
	PodCreated    time.Time
	PodPhase      string
	RunningDigest string // e.g., "sha256:3778e43a..."
	RegistryCreds *RegistryCreds
}

type RegistryCreds struct {
	Auth string // base64-encoded "user:password"
}
