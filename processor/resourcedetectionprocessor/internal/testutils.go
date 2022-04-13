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

package internal // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourcedetectionprocessor/internal"

import "go.opentelemetry.io/collector/model/pdata"

func NewResource(mp map[string]interface{}) pdata.Resource {
	res := pdata.NewResource()
	attr := res.Attributes()
	fillAttributeMap(mp, attr)
	return res
}

func NewAttributeMap(mp map[string]interface{}) pdata.Map {
	attr := pdata.NewMap()
	fillAttributeMap(mp, attr)
	return attr
}

func fillAttributeMap(mp map[string]interface{}, attr pdata.Map) {
	attr.Clear()
	attr.EnsureCapacity(len(mp))
	for k, v := range mp {
		switch t := v.(type) {
		case bool:
			attr.Insert(k, pdata.NewValueBool(t))
		case int64:
			attr.Insert(k, pdata.NewValueInt(t))
		case float64:
			attr.Insert(k, pdata.NewValueDouble(t))
		case string:
			attr.Insert(k, pdata.NewValueString(t))
		}
	}
}
