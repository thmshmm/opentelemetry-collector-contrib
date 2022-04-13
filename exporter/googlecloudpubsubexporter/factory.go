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

package googlecloudpubsubexporter // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/googlecloudpubsubexporter"

import (
	"context"
	"fmt"
	"strings"
	"time"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/exporter/exporterhelper"
	"go.opentelemetry.io/collector/model/otlp"
)

const (
	// The value of "type" key in configuration.
	typeStr        = "googlecloudpubsub"
	defaultTimeout = 12 * time.Second
)

// NewFactory creates a factory for Google Cloud Pub/Sub exporter.
func NewFactory() component.ExporterFactory {
	return component.NewExporterFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesExporter(createTracesExporter),
		component.WithMetricsExporter(createMetricsExporter),
		component.WithLogsExporter(createLogsExporter))
}

var exporters = map[*Config]*pubsubExporter{}

func ensureExporter(params component.ExporterCreateSettings, pCfg *Config) *pubsubExporter {
	receiver := exporters[pCfg]
	if receiver != nil {
		return receiver
	}
	receiver = &pubsubExporter{
		logger:           params.Logger,
		userAgent:        strings.ReplaceAll(pCfg.UserAgent, "{{version}}", params.BuildInfo.Version),
		ceSource:         fmt.Sprintf("/opentelemetry/collector/%s/%s", name, params.BuildInfo.Version),
		config:           pCfg,
		tracesMarshaler:  otlp.NewProtobufTracesMarshaler(),
		metricsMarshaler: otlp.NewProtobufMetricsMarshaler(),
		logsMarshaler:    otlp.NewProtobufLogsMarshaler(),
	}
	// we ignore the error here as the config is already validated with the same method
	receiver.ceCompression, _ = pCfg.parseCompression()
	watermarkBehavior, _ := pCfg.Watermark.parseWatermarkBehavior()
	switch watermarkBehavior {
	case earliest:
		receiver.tracesWatermarkFunc = earliestTracesWatermark
		receiver.metricsWatermarkFunc = earliestMetricsWatermark
		receiver.logsWatermarkFunc = earliestLogsWatermark
	case current:
		receiver.tracesWatermarkFunc = currentTracesWatermark
		receiver.metricsWatermarkFunc = currentMetricsWatermark
		receiver.logsWatermarkFunc = currentLogsWatermark
	}
	exporters[pCfg] = receiver
	return receiver
}

// createDefaultConfig creates the default configuration for exporter.
func createDefaultConfig() config.Exporter {
	return &Config{
		ExporterSettings: config.NewExporterSettings(config.NewComponentID(typeStr)),
		UserAgent:        "opentelemetry-collector-contrib {{version}}",
		TimeoutSettings:  exporterhelper.TimeoutSettings{Timeout: defaultTimeout},
		Watermark: WatermarkConfig{
			Behavior:     "current",
			AllowedDrift: 0,
		},
	}
}

func createTracesExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter) (component.TracesExporter, error) {

	pCfg := cfg.(*Config)
	pubsubExporter := ensureExporter(set, pCfg)

	return exporterhelper.NewTracesExporter(
		cfg,
		set,
		pubsubExporter.consumeTraces,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutSettings),
		exporterhelper.WithRetry(pCfg.RetrySettings),
		exporterhelper.WithQueue(pCfg.QueueSettings),
		exporterhelper.WithStart(pubsubExporter.start),
		exporterhelper.WithShutdown(pubsubExporter.shutdown),
	)
}

func createMetricsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter) (component.MetricsExporter, error) {

	pCfg := cfg.(*Config)
	pubsubExporter := ensureExporter(set, pCfg)
	return exporterhelper.NewMetricsExporter(
		cfg,
		set,
		pubsubExporter.consumeMetrics,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutSettings),
		exporterhelper.WithRetry(pCfg.RetrySettings),
		exporterhelper.WithQueue(pCfg.QueueSettings),
		exporterhelper.WithStart(pubsubExporter.start),
		exporterhelper.WithShutdown(pubsubExporter.shutdown),
	)
}

func createLogsExporter(
	_ context.Context,
	set component.ExporterCreateSettings,
	cfg config.Exporter) (component.LogsExporter, error) {

	pCfg := cfg.(*Config)
	pubsubExporter := ensureExporter(set, pCfg)

	return exporterhelper.NewLogsExporter(
		cfg,
		set,
		pubsubExporter.consumeLogs,
		exporterhelper.WithCapabilities(consumer.Capabilities{MutatesData: false}),
		exporterhelper.WithTimeout(pCfg.TimeoutSettings),
		exporterhelper.WithRetry(pCfg.RetrySettings),
		exporterhelper.WithQueue(pCfg.QueueSettings),
		exporterhelper.WithStart(pubsubExporter.start),
		exporterhelper.WithShutdown(pubsubExporter.shutdown),
	)
}
