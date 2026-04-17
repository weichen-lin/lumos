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
	"sort"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

// EntryMapping records which ConfigMap keys were produced by one data entry.
type EntryMapping struct {
	// Source is the source path / KV key (from ExternalConfigData.source).
	Source string
	// Keys are the ConfigMap keys that were written for this entry.
	Keys []string
}

// FetchResult holds the data returned by a provider after a successful fetch.
type FetchResult struct {
	// Data maps key → value, ready to write into a ConfigMap.
	Data map[string]string

	// Version is a provider-specific version identifier
	// (commit SHA for Git).
	Version string

	// Mappings records which source path/key produced which local ConfigMap keys.
	Mappings []EntryMapping
}

// mergeEntries merges the entries produced by one data entry into the
// accumulated result, returning an error if any key already exists (conflict).
func mergeEntries(source string, entries map[string]string, result map[string]string, mappings *[]EntryMapping) error {
	keys := make([]string, 0, len(entries))
	for k := range entries {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		v := entries[k]
		if _, exists := result[k]; exists {
			return fmt.Errorf("key conflict: %q is produced by multiple data entries", k)
		}
		result[k] = v
	}

	*mappings = append(*mappings, EntryMapping{
		Source: source,
		Keys:   keys,
	})
	return nil
}

// Provider is the interface every backend must implement.
// Fetch reads the listed keys from the remote source and returns
// a map of localKey → value plus metadata.
type Provider interface {
	Fetch(ctx context.Context, data []syncv1alpha1.ExternalConfigData) (FetchResult, error)
}
