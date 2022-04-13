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

package windowsperfcountersreceiver

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component/componenterror"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config/configtest"
	"go.opentelemetry.io/collector/consumer/consumertest"

	winperfcounters "github.com/open-telemetry/opentelemetry-collector-contrib/pkg/winperfcounters"
)

var creationParams = componenttest.NewNopReceiverCreateSettings()

func TestCreateDefaultConfig(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()

	assert.NotNil(t, cfg, "failed to create default config")
	assert.NoError(t, configtest.CheckConfigStruct(cfg))

	cfg.(*Config).PerfCounters = []winperfcounters.ObjectConfig{
		{
			Object:   "object",
			Counters: []winperfcounters.CounterConfig{{Name: "counter", MetricRep: winperfcounters.MetricRep{Name: "metric"}}},
		},
	}

	cfg.(*Config).MetricMetaData = map[string]MetricConfig{
		"metric": {
			Description: "desc",
			Unit:        "1",
			Gauge:       GaugeMetric{},
		},
	}

	assert.NoError(t, cfg.Validate())
}

func TestCreateTracesReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).PerfCounters = []winperfcounters.ObjectConfig{
		{
			Object:   "object",
			Counters: []winperfcounters.CounterConfig{{Name: "counter", MetricRep: winperfcounters.MetricRep{Name: "metric"}}},
		},
	}

	cfg.(*Config).MetricMetaData = map[string]MetricConfig{
		"metric": {
			Description: "desc",
			Unit:        "1",
			Gauge:       GaugeMetric{},
		},
	}
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), creationParams, cfg, consumertest.NewNop())

	assert.ErrorIs(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, tReceiver)
}

func TestCreateTracesReceiverNoMetrics(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).PerfCounters = []winperfcounters.ObjectConfig{
		{
			Object:   "object",
			Counters: []winperfcounters.CounterConfig{{Name: "counter"}},
		},
	}
	tReceiver, err := factory.CreateTracesReceiver(context.Background(), creationParams, cfg, consumertest.NewNop())

	assert.ErrorIs(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, tReceiver)
}

func TestCreateLogsReceiver(t *testing.T) {
	factory := NewFactory()
	cfg := factory.CreateDefaultConfig()
	cfg.(*Config).PerfCounters = []winperfcounters.ObjectConfig{
		{
			Object:   "object",
			Counters: []winperfcounters.CounterConfig{{Name: "counter", MetricRep: winperfcounters.MetricRep{Name: "metric"}}},
		},
	}

	cfg.(*Config).MetricMetaData = map[string]MetricConfig{
		"metric": {
			Description: "desc",
			Unit:        "1",
			Gauge:       GaugeMetric{},
		},
	}

	tReceiver, err := factory.CreateLogsReceiver(context.Background(), creationParams, cfg, consumertest.NewNop())

	assert.ErrorIs(t, err, componenterror.ErrDataTypeIsNotSupported)
	assert.Nil(t, tReceiver)
}
