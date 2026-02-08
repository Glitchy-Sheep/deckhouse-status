package kube

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (c *Client) fetchRegistryCreds(ctx context.Context) (*RegistryCreds, error) {
	secret, err := c.cs.CoreV1().Secrets(namespace).Get(ctx, secretName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}

	raw := secret.Data[".dockerconfigjson"]
	if len(raw) == 0 {
		return nil, fmt.Errorf("empty .dockerconfigjson in secret %s", secretName)
	}

	var cfg struct {
		Auths map[string]struct {
			Auth string `json:"auth"`
		} `json:"auths"`
	}
	if err := json.Unmarshal(raw, &cfg); err != nil {
		return nil, fmt.Errorf("cannot parse docker config: %w", err)
	}

	for _, entry := range cfg.Auths {
		if entry.Auth != "" {
			return &RegistryCreds{Auth: entry.Auth}, nil
		}
	}

	return nil, fmt.Errorf("no auth entries in docker config")
}
