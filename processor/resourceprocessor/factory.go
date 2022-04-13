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

package resourceprocessor // import "github.com/open-telemetry/opentelemetry-collector-contrib/processor/resourceprocessor"

import (
	"context"
	"fmt"

	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/consumer"
	"go.opentelemetry.io/collector/processor/processorhelper"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/attraction"
)

const (
	// The value of "type" key in configuration.
	typeStr = "resource"
)

var processorCapabilities = consumer.Capabilities{MutatesData: true}

// NewFactory returns a new factory for the Resource processor.
func NewFactory() component.ProcessorFactory {
	return component.NewProcessorFactory(
		typeStr,
		createDefaultConfig,
		component.WithTracesProcessor(createTracesProcessor),
		component.WithMetricsProcessor(createMetricsProcessor),
		component.WithLogsProcessor(createLogsProcessor))
}

// Note: This isn't a valid configuration because the processor would do no work.
func createDefaultConfig() config.Processor {
	return &Config{
		ProcessorSettings: config.NewProcessorSettings(config.NewComponentID(typeStr)),
	}
}

func createTracesProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Traces) (component.TracesProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	proc := &resourceProcessor{logger: params.Logger, attrProc: attrProc}
	return processorhelper.NewTracesProcessor(
		cfg,
		nextConsumer,
		proc.processTraces,
		processorhelper.WithCapabilities(processorCapabilities))
}

func createMetricsProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Metrics) (component.MetricsProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	proc := &resourceProcessor{logger: params.Logger, attrProc: attrProc}
	return processorhelper.NewMetricsProcessor(
		cfg,
		nextConsumer,
		proc.processMetrics,
		processorhelper.WithCapabilities(processorCapabilities))
}

func createLogsProcessor(
	_ context.Context,
	params component.ProcessorCreateSettings,
	cfg config.Processor,
	nextConsumer consumer.Logs) (component.LogsProcessor, error) {
	attrProc, err := createAttrProcessor(cfg.(*Config))
	if err != nil {
		return nil, err
	}
	proc := &resourceProcessor{logger: params.Logger, attrProc: attrProc}
	return processorhelper.NewLogsProcessor(
		cfg,
		nextConsumer,
		proc.processLogs,
		processorhelper.WithCapabilities(processorCapabilities))
}

func createAttrProcessor(cfg *Config) (*attraction.AttrProc, error) {
	if len(cfg.AttributesActions) == 0 {
		return nil, fmt.Errorf("error creating \"%v\" processor due to missing required field \"attributes\"", cfg.ID())
	}
	attrProc, err := attraction.NewAttrProc(&attraction.Settings{Actions: cfg.AttributesActions})
	if err != nil {
		return nil, fmt.Errorf("error creating \"%v\" processor: %w", cfg.ID(), err)
	}
	return attrProc, nil
}
