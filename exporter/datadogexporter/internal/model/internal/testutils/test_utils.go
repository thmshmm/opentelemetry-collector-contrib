// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package testutils // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/datadogexporter/internal/model/internal/testutils"

import "go.opentelemetry.io/collector/model/pdata"

func fillAttributeMap(attrs pdata.Map, mp map[string]string) {
	attrs.Clear()
	attrs.EnsureCapacity(len(mp))
	for k, v := range mp {
		attrs.Insert(k, pdata.NewValueString(v))
	}
}

// NewAttributeMap creates a new attribute map (string only)
// from a Go map
func NewAttributeMap(mp map[string]string) pdata.Map {
	attrs := pdata.NewMap()
	fillAttributeMap(attrs, mp)
	return attrs
}
