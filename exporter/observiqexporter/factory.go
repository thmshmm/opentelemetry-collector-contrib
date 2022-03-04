// Copyright  OpenTelemetry Authors
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package observiqexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/observiqexporter"

import (
	"context"
	"errors"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
)

const (
	typeStr              = "observiq"
	defaultHTTPTimeout   = 20 * time.Second
	defaultEndpoint      = "https://nozzle.app.observiq.com/v1/add"
	defaultDialerTimeout = 10 * time.Second
)

// NewFactory creates a factory for observIQ exporter
func NewFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithLogsExporter(createLogsExporter),
	)
}

func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		Endpoint:         defaultEndpoint,
		TimeoutSettings: exporterhelper.TimeoutSettings{
			Timeout: defaultHTTPTimeout,
		},
		RetrySettings: exporterhelper.NewDefaultRetrySettings(),
		QueueSettings: exporterhelper.NewDefaultQueueSettings(),
		AgentID:       defaultAgentID(),
		AgentName:     defaultAgentName(),
		DialerTimeout: defaultDialerTimeout,
	}
}

func createLogsExporter(
	_ context.Context,
	params component.ExporterCreateSettings,
	config config.Exporter,
) (exporter component.LogsExporter, err error) {
	if config == nil {
		return nil, errors.New("nil config")
	}
	exporterConfig := config.(*Config)
	return newObservIQLogExporter(exporterConfig, params)
}
