/*
Copyright 2026.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controller

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/types"
	"sigs.k8s.io/controller-runtime/pkg/client"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	"github.com/weichen-lin/lumos/internal/provider"
)

// resolveGitAuthFromClient reads credentials from the Secret referenced by cfg.SecretRef
// and returns a GitAuth. Returns nil if no SecretRef is set (public repo).
func resolveGitAuthFromClient(
	ctx context.Context,
	c client.Client,
	namespace string,
	cfg *syncv1alpha1.GitProvider,
) (*provider.GitAuth, error) {
	if cfg.SecretRef == nil {
		return nil, nil
	}

	var secret corev1.Secret
	if err := c.Get(ctx, types.NamespacedName{
		Name:      cfg.SecretRef.Name,
		Namespace: namespace,
	}, &secret); err != nil {
		return nil, fmt.Errorf("reading git secret %q: %w", cfg.SecretRef.Name, err)
	}

	auth := &provider.GitAuth{}

	if key, ok := secret.Data["sshPrivateKey"]; ok {
		auth.SSHPrivateKey = key
		return auth, nil
	}
	if token, ok := secret.Data["token"]; ok {
		auth.Username = "token"
		auth.Password = string(token)
		return auth, nil
	}
	if pass, ok := secret.Data["password"]; ok {
		if username, ok := secret.Data["username"]; ok {
			auth.Username = string(username)
		}
		auth.Password = string(pass)
		return auth, nil
	}

	return nil, fmt.Errorf("secret %q has no recognized key (sshPrivateKey / token / password)", cfg.SecretRef.Name)
}
