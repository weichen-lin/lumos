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

package transform

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
	"unicode"

	syncv1alpha1 "github.com/weichen-lin/lumos/api/v1alpha1"
)

// Apply processes rawValue according to format and returns a map of ConfigMap entries.
func Apply(format syncv1alpha1.DataFormat, key, rawValue string) (map[string]string, error) {
	if format == syncv1alpha1.FormatEnv {
		if key != "" {
			return nil, fmt.Errorf("key must be empty when format is Env")
		}
		return parseEnv(rawValue)
	}

	// FormatRaw (default)
	if key == "" {
		return nil, fmt.Errorf("key is required when format is Raw")
	}
	return map[string]string{key: rawValue}, nil
}

func parseEnv(rawValue string) (map[string]string, error) {
	trimmed := strings.TrimSpace(rawValue)
	if trimmed == "" {
		return make(map[string]string), nil
	}

	// Try JSON
	if strings.HasPrefix(trimmed, "{") || strings.HasPrefix(trimmed, "[") {
		var obj interface{}
		if err := json.Unmarshal([]byte(trimmed), &obj); err == nil {
			if _, ok := obj.(map[string]interface{}); !ok {
				return nil, fmt.Errorf("Env format requires a JSON object at root, got array or scalar")
			}
			result := make(map[string]string)
			flattenValue("", obj, result, true) // Use UPPER_SNAKE_CASE for Env
			return result, nil
		}
	}

	// Fallback to .env (already flat, but ensure valid keys)
	return parseDotEnv(rawValue)
}

// flattenValue recursively walks the parsed JSON tree.
func flattenValue(prefix string, value interface{}, result map[string]string, isEnv bool) {
	switch v := value.(type) {
	case map[string]interface{}:
		for k, child := range v {
			childKey := k
			if isEnv {
				childKey = toEnvVarName(k)
				if prefix != "" {
					childKey = prefix + "_" + childKey
				}
			} else {
				if prefix != "" {
					childKey = prefix + "." + k
				}
			}
			flattenValue(childKey, child, result, isEnv)
		}

	case []interface{}:
		b, _ := json.Marshal(v)
		result[prefix] = string(b)

	case bool:
		result[prefix] = strconv.FormatBool(v)

	case string:
		result[prefix] = v

	case nil:
		result[prefix] = ""

	default:
		result[prefix] = fmt.Sprintf("%v", v)
	}
}

// toEnvVarName converts a string to UPPER_SNAKE_CASE part.
func toEnvVarName(s string) string {
	var buf strings.Builder
	prevWasLower := false
	for _, r := range s {
		if unicode.IsUpper(r) && prevWasLower {
			buf.WriteRune('_')
		}
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			buf.WriteRune(unicode.ToUpper(r))
			prevWasLower = unicode.IsLower(r)
		} else {
			buf.WriteRune('_')
			prevWasLower = false
		}
	}
	return buf.String()
}

func parseDotEnv(rawValue string) (map[string]string, error) {
	res := make(map[string]string)
	lines := strings.Split(rawValue, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		k := strings.TrimSpace(parts[0])
		v := strings.TrimSpace(parts[1])
		if len(v) >= 2 && ((v[0] == '"' && v[len(v)-1] == '"') || (v[0] == '\'' && v[len(v)-1] == '\'')) {
			v = v[1 : len(v)-1]
		}
		res[k] = v
	}
	return res, nil
}
