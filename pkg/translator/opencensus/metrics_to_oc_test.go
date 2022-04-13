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

package opencensus

import (
	"testing"
	"time"

	occommon "github.com/census-instrumentation/opencensus-proto/gen-go/agent/common/v1"
	agentmetricspb "github.com/census-instrumentation/opencensus-proto/gen-go/agent/metrics/v1"
	ocmetrics "github.com/census-instrumentation/opencensus-proto/gen-go/metrics/v1"
	ocresource "github.com/census-instrumentation/opencensus-proto/gen-go/resource/v1"
	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.6.1"
	"google.golang.org/protobuf/types/known/timestamppb"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/occonventions"
	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/coreinternal/testdata"
)

func TestMetricsToOC(t *testing.T) {
	sampleMetricData := testdata.GeneratMetricsAllTypesWithSampleDatapoints()
	attrs := sampleMetricData.ResourceMetrics().At(0).Resource().Attributes()
	attrs.Upsert(conventions.AttributeHostName, pdata.NewValueString("host1"))
	attrs.Upsert(conventions.AttributeProcessPID, pdata.NewValueInt(123))
	attrs.Upsert(occonventions.AttributeProcessStartTime, pdata.NewValueString("2020-02-11T20:26:00Z"))
	attrs.Upsert(conventions.AttributeTelemetrySDKLanguage, pdata.NewValueString("cpp"))
	attrs.Upsert(conventions.AttributeTelemetrySDKVersion, pdata.NewValueString("v2.0.1"))
	attrs.Upsert(occonventions.AttributeExporterVersion, pdata.NewValueString("v1.2.0"))

	tests := []struct {
		name     string
		internal pdata.Metrics
		oc       *agentmetricspb.ExportMetricsServiceRequest
	}{
		{
			name:     "one-empty-resource-metrics",
			internal: testdata.GenerateMetricsOneEmptyResourceMetrics(),
			oc:       &agentmetricspb.ExportMetricsServiceRequest{},
		},

		{
			name:     "no-libraries",
			internal: testdata.GenerateMetricsNoLibraries(),
			oc:       generateOCTestDataNoMetrics(),
		},

		{
			name:     "one-empty-instrumentation-library",
			internal: testdata.GenerateMetricsOneEmptyInstrumentationLibrary(),
			oc:       generateOCTestDataNoMetrics(),
		},

		{
			name:     "one-metric-no-resource",
			internal: testdata.GenerateMetricsOneMetricNoResource(),
			oc: &agentmetricspb.ExportMetricsServiceRequest{
				Metrics: []*ocmetrics.Metric{
					generateOCTestMetricCumulativeInt(),
				},
			},
		},

		{
			name:     "one-metric",
			internal: testdata.GenerateMetricsOneMetric(),
			oc:       generateOCTestDataMetricsOneMetric(),
		},

		{
			name:     "one-metric-no-labels",
			internal: testdata.GenerateMetricsOneMetricNoAttributes(),
			oc:       generateOCTestDataNoLabels(),
		},

		// TODO: Enable this after the testdata.GenerateMetricsAllTypesNoDataPoints is changed
		//  to generate one sum and one gauge (no difference between int/double when no points).
		// {
		//   name:     "all-types-no-data-points",
		//   internal: testdata.GenerateMetricsAllTypesNoDataPoints(),
		//   oc:       generateOCTestDataNoPoints(),
		// },

		{
			name:     "all-types",
			internal: sampleMetricData,
			oc:       generateOCTestData(),
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			gotNode, gotResource, gotMetrics := ResourceMetricsToOC(test.internal.ResourceMetrics().At(0))
			assert.EqualValues(t, test.oc.Node, gotNode)
			assert.EqualValues(t, test.oc.Resource, gotResource)
			assert.EqualValues(t, test.oc.Metrics, gotMetrics)
		})
	}
}

func TestMetricsToOC_InvalidDataType(t *testing.T) {
	internal := testdata.GenerateMetricsMetricTypeInvalid()
	want := &agentmetricspb.ExportMetricsServiceRequest{
		Node: &occommon.Node{},
		Resource: &ocresource.Resource{
			Labels: map[string]string{"resource-attr": "resource-attr-val-1"},
		},
		Metrics: []*ocmetrics.Metric{
			{
				MetricDescriptor: &ocmetrics.MetricDescriptor{
					Name:      testdata.TestSumIntMetricName,
					Unit:      "1",
					Type:      ocmetrics.MetricDescriptor_UNSPECIFIED,
					LabelKeys: nil,
				},
			},
		},
	}
	gotNode, gotResource, gotMetrics := ResourceMetricsToOC(internal.ResourceMetrics().At(0))
	assert.EqualValues(t, want.Node, gotNode)
	assert.EqualValues(t, want.Resource, gotResource)
	assert.EqualValues(t, want.Metrics, gotMetrics)
}

func generateOCTestData() *agentmetricspb.ExportMetricsServiceRequest {
	ts := timestamppb.New(time.Date(2020, 2, 11, 20, 26, 0, 0, time.UTC))

	return &agentmetricspb.ExportMetricsServiceRequest{
		Node: &occommon.Node{
			Identifier: &occommon.ProcessIdentifier{
				HostName:       "host1",
				Pid:            123,
				StartTimestamp: ts,
			},
			LibraryInfo: &occommon.LibraryInfo{
				Language:           occommon.LibraryInfo_CPP,
				ExporterVersion:    "v1.2.0",
				CoreLibraryVersion: "v2.0.1",
			},
		},
		Resource: &ocresource.Resource{
			Labels: map[string]string{
				"resource-attr": "resource-attr-val-1",
			},
		},
		Metrics: []*ocmetrics.Metric{
			generateOCTestMetricGaugeInt(),
			generateOCTestMetricGaugeDouble(),
			generateOCTestMetricCumulativeInt(),
			generateOCTestMetricCumulativeDouble(),
			generateOCTestMetricDoubleHistogram(),
			generateOCTestMetricDoubleSummary(),
		},
	}
}

func TestMetricsType(t *testing.T) {
	tests := []struct {
		name     string
		internal func() pdata.Metric
		descType ocmetrics.MetricDescriptor_Type
	}{
		{
			name: "int-gauge",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeGauge)
				m.Gauge().DataPoints().AppendEmpty().SetIntVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_INT64,
		},
		{
			name: "double-gauge",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeGauge)
				m.Gauge().DataPoints().AppendEmpty().SetDoubleVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
		},
		{
			name: "int-non-monotonic-delta-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(false)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityDelta)
				m.Sum().DataPoints().AppendEmpty().SetIntVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_INT64,
		},
		{
			name: "int-non-monotonic-cumulative-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(false)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)
				m.Sum().DataPoints().AppendEmpty().SetIntVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_INT64,
		},
		{
			name: "int-monotonic-delta-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityDelta)
				m.Sum().DataPoints().AppendEmpty().SetIntVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_INT64,
		},
		{
			name: "int-monotonic-cumulative-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)
				m.Sum().DataPoints().AppendEmpty().SetIntVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_CUMULATIVE_INT64,
		},
		{
			name: "double-non-monotonic-delta-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(false)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityDelta)
				m.Sum().DataPoints().AppendEmpty().SetDoubleVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
		},
		{
			name: "double-non-monotonic-cumulative-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(false)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)
				m.Sum().DataPoints().AppendEmpty().SetDoubleVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
		},
		{
			name: "double-monotonic-delta-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityDelta)
				m.Sum().DataPoints().AppendEmpty().SetDoubleVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_GAUGE_DOUBLE,
		},
		{
			name: "double-monotonic-cumulative-sum",
			internal: func() pdata.Metric {
				m := pdata.NewMetric()
				m.SetDataType(pdata.MetricDataTypeSum)
				m.Sum().SetIsMonotonic(true)
				m.Sum().SetAggregationTemporality(pdata.MetricAggregationTemporalityCumulative)
				m.Sum().DataPoints().AppendEmpty().SetDoubleVal(1)
				return m
			},
			descType: ocmetrics.MetricDescriptor_CUMULATIVE_DOUBLE,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assert.Equal(t, test.descType, metricToOC(test.internal()).MetricDescriptor.Type)
		})
	}
}
