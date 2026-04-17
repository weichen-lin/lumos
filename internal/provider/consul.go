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

package provider

import (
	"context"
	"fmt"
	"strconv"

	consul "github.com/hashicorp/consul/api"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
	"github.com/weichen-lin/lumos/internal/transform"
)

// ConsulProvider fetches config from a Consul KV store.
type ConsulProvider struct {
	address string
	prefix  string
	token   string
}

// NewConsul creates a ConsulProvider. token may be empty for unauthenticated agents.
func NewConsul(address, prefix, token string) *ConsulProvider {
	return &ConsulProvider{address: address, prefix: prefix, token: token}
}

// Fetch reads the requested keys from Consul KV.
// Each RemoteKey is resolved relative to the store prefix.
func (c *ConsulProvider) Fetch(_ context.Context, data []syncv1alpha1.ExternalConfigData) (FetchResult, error) {
	cfg := consul.DefaultConfig()
	cfg.Address = c.address
	if c.token != "" {
		cfg.Token = c.token
	}

	client, err := consul.NewClient(cfg)
	if err != nil {
		return FetchResult{}, fmt.Errorf("creating consul client: %w", err)
	}

	kv := client.KV()
	result := make(map[string]string)
	var mappings []EntryMapping

	// Track the highest ModifyIndex across all keys as a version signal.
	var maxIndex uint64

	for _, d := range data {
		fullKey := c.prefix + "/" + d.Source
		pair, _, err := kv.Get(fullKey, nil)
		if err != nil {
			return FetchResult{}, fmt.Errorf("reading consul key %q: %w", fullKey, err)
		}
		if pair == nil {
			return FetchResult{}, fmt.Errorf("consul key %q not found", fullKey)
		}
		entries, err := transform.Apply(d.Format, d.Key, string(pair.Value))
		if err != nil {
			return FetchResult{}, fmt.Errorf("transforming %q: %w", fullKey, err)
		}
		if err := mergeEntries(d.Source, entries, result, &mappings); err != nil {
			return FetchResult{}, err
		}
		if pair.ModifyIndex > maxIndex {
			maxIndex = pair.ModifyIndex
		}
	}

	return FetchResult{
		Data:     result,
		Version:  strconv.FormatUint(maxIndex, 10),
		Mappings: mappings,
	}, nil
}
