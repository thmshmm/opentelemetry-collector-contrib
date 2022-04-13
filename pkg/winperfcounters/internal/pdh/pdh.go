// Copyright The OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//       http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build windows
// +build windows

package pdh // import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/winperfcounters/internal/pdh"

import "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/winperfcounters/internal/third_party/telegraf/win_perf_counters"

const totalInstanceName = "_Total"

type PerfCounter struct {
	path   string
	query  win_perf_counters.PerformanceQuery
	handle win_perf_counters.PDH_HCOUNTER
}

// NewPerfCounter returns a new performance counter for the specified descriptor.
func NewPerfCounter(counterPath string, collectOnStartup bool) (*PerfCounter, error) {
	query := &win_perf_counters.PerformanceQueryImpl{}
	err := query.Open()
	if err != nil {
		return nil, err
	}

	var handle win_perf_counters.PDH_HCOUNTER
	handle, err = query.AddEnglishCounterToQuery(counterPath)
	if err != nil {
		return nil, err
	}

	// Some perf counters (e.g. cpu) return the usage stats since the last measure.
	// We collect data on startup to avoid an invalid initial reading
	if collectOnStartup {
		err = query.CollectData()
		if err != nil {
			return nil, err
		}
	}

	counter := &PerfCounter{
		path:   counterPath,
		query:  query,
		handle: handle,
	}

	return counter, nil
}

// Close
func (pc *PerfCounter) Close() error {
	return pc.query.Close()
}

// Path
func (pc *PerfCounter) Path() string {
	return pc.path
}

// ScrapeData
func (pc *PerfCounter) ScrapeData() ([]win_perf_counters.CounterValue, error) {
	err := pc.query.CollectData()
	if err != nil {
		return nil, err
	}

	vals, err := pc.query.GetFormattedCounterArrayDouble(pc.handle)
	if err != nil {
		return nil, err
	}

	vals = removeTotalIfMultipleValues(vals)
	return vals, nil
}

func removeTotalIfMultipleValues(vals []win_perf_counters.CounterValue) []win_perf_counters.CounterValue {
	if len(vals) == 0 {
		return vals
	}

	if len(vals) == 1 {
		// if there is only one item & the instance name is "_Total", clear the instance name
		if vals[0].InstanceName == totalInstanceName {
			vals[0].InstanceName = ""
		}
		return vals
	}

	// if there is more than one item, remove an item that has the instance name "_Total"
	for i, val := range vals {
		if val.InstanceName == totalInstanceName {
			return removeItemAt(vals, i)
		}
	}

	return vals
}

func removeItemAt(vals []win_perf_counters.CounterValue, idx int) []win_perf_counters.CounterValue {
	vals[idx] = vals[len(vals)-1]
	vals[len(vals)-1] = win_perf_counters.CounterValue{}
	return vals[:len(vals)-1]
}
