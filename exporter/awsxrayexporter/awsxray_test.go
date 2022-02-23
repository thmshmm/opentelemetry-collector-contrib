// Copyright 2019, OpenTelemetry Authors
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

package awsxrayexporter

import (
	"context"
	"encoding/binary"
	"fmt"
	"math/rand"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/component"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/config"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.6.1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/awsutil"
)

func TestTraceExport(t *testing.T) {
	traceExporter := initializeTracesExporter()
	ctx := context.Background()
	td := constructSpanData()
	err := traceExporter.ConsumeTraces(ctx, td)
	assert.NotNil(t, err)
	err = traceExporter.Shutdown(ctx)
	assert.Nil(t, err)
}

func TestXraySpanTraceResourceExtraction(t *testing.T) {
	td := constructSpanData()
	logger, _ := zap.NewProduction()
	assert.Len(t, extractResourceSpans(generateConfig(), logger, td), 2, "2 spans have xay trace id")
}

func TestXrayAndW3CSpanTraceExport(t *testing.T) {
	traceExporter := initializeTracesExporter()
	ctx := context.Background()
	td := constructXrayAndW3CSpanData()
	err := traceExporter.ConsumeTraces(ctx, td)
	assert.NotNil(t, err)
	err = traceExporter.Shutdown(ctx)
	assert.Nil(t, err)
}

func TestXrayAndW3CSpanTraceResourceExtraction(t *testing.T) {
	td := constructXrayAndW3CSpanData()
	logger, _ := zap.NewProduction()
	assert.Len(t, extractResourceSpans(generateConfig(), logger, td), 2, "2 spans have xay trace id")
}

func TestW3CSpanTraceResourceExtraction(t *testing.T) {
	td := constructW3CSpanData()
	logger, _ := zap.NewProduction()
	assert.Len(t, extractResourceSpans(generateConfig(), logger, td), 0, "0 spans have xray trace id")
}

func BenchmarkForTracesExporter(b *testing.B) {
	traceExporter := initializeTracesExporter()
	for i := 0; i < b.N; i++ {
		b.StopTimer()
		ctx := context.Background()
		td := constructSpanData()
		b.StartTimer()
		traceExporter.ConsumeTraces(ctx, td)
	}
}

func initializeTracesExporter() component.TracesExporter {
	exporterConfig := generateConfig()
	mconn := new(awsutil.Conn)
	traceExporter, err := newTracesExporter(exporterConfig, componenttest.NewNopExporterCreateSettings(), mconn)
	if err != nil {
		panic(err)
	}
	return traceExporter
}

func generateConfig() config.Exporter {
	os.Setenv("AWS_ACCESS_KEY_ID", "AKIASSWVJUY4PZXXXXXX")
	os.Setenv("AWS_SECRET_ACCESS_KEY", "XYrudg2H87u+ADAAq19Wqx3D41a09RsTXXXXXXXX")
	os.Setenv("AWS_DEFAULT_REGION", "us-east-1")
	os.Setenv("AWS_REGION", "us-east-1")
	factory := NewFactory()
	exporterConfig := factory.CreateDefaultConfig()
	exporterConfig.(*Config).Region = "us-east-1"
	exporterConfig.(*Config).LocalMode = true
	return exporterConfig
}

func constructSpanData() pdata.Traces {
	resource := constructResource()

	traces := pdata.NewTraces()
	rspans := traces.ResourceSpans().AppendEmpty()
	resource.CopyTo(rspans.Resource())
	ispans := rspans.InstrumentationLibrarySpans().AppendEmpty()
	constructXrayTraceSpanData(ispans)
	return traces
}

func constructW3CSpanData() pdata.Traces {
	resource := constructResource()
	traces := pdata.NewTraces()
	rspans := traces.ResourceSpans().AppendEmpty()
	resource.CopyTo(rspans.Resource())
	ispans := rspans.InstrumentationLibrarySpans().AppendEmpty()
	constructW3CFormatTraceSpanData(ispans)
	return traces
}

func constructXrayAndW3CSpanData() pdata.Traces {
	resource := constructResource()
	traces := pdata.NewTraces()
	rspans := traces.ResourceSpans().AppendEmpty()
	resource.CopyTo(rspans.Resource())
	ispans := rspans.InstrumentationLibrarySpans().AppendEmpty()
	constructXrayTraceSpanData(ispans)
	constructW3CFormatTraceSpanData(ispans)
	return traces
}

func constructXrayTraceSpanData(ispans pdata.InstrumentationLibrarySpans) {
	constructHTTPClientSpan(newTraceID()).CopyTo(ispans.Spans().AppendEmpty())
	constructHTTPServerSpan(newTraceID()).CopyTo(ispans.Spans().AppendEmpty())
}

func constructW3CFormatTraceSpanData(ispans pdata.InstrumentationLibrarySpans) {
	constructHTTPClientSpan(constructW3CTraceID()).CopyTo(ispans.Spans().AppendEmpty())
	constructHTTPServerSpan(constructW3CTraceID()).CopyTo(ispans.Spans().AppendEmpty())
}

func constructResource() pdata.Resource {
	resource := pdata.NewResource()
	attrs := pdata.NewAttributeMap()
	attrs.InsertString(conventions.AttributeServiceName, "signup_aggregator")
	attrs.InsertString(conventions.AttributeContainerName, "signup_aggregator")
	attrs.InsertString(conventions.AttributeContainerImageName, "otel/signupaggregator")
	attrs.InsertString(conventions.AttributeContainerImageTag, "v1")
	attrs.InsertString(conventions.AttributeCloudProvider, conventions.AttributeCloudProviderAWS)
	attrs.InsertString(conventions.AttributeCloudAccountID, "999999998")
	attrs.InsertString(conventions.AttributeCloudRegion, "us-west-2")
	attrs.InsertString(conventions.AttributeCloudAvailabilityZone, "us-west-1b")
	attrs.CopyTo(resource.Attributes())
	return resource
}

func constructHTTPClientSpan(traceID pdata.TraceID) pdata.Span {
	attributes := make(map[string]interface{})
	attributes[conventions.AttributeHTTPMethod] = "GET"
	attributes[conventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	attributes[conventions.AttributeHTTPStatusCode] = 200
	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)
	spanAttributes := constructSpanAttributes(attributes)

	span := pdata.NewSpan()
	span.SetTraceID(traceID)
	span.SetSpanID(newSegmentID())
	span.SetParentSpanID(newSegmentID())
	span.SetName("/users/junit")
	span.SetKind(pdata.SpanKindClient)
	span.SetStartTimestamp(pdata.NewTimestampFromTime(startTime))
	span.SetEndTimestamp(pdata.NewTimestampFromTime(endTime))

	status := pdata.NewSpanStatus()
	status.SetCode(0)
	status.SetMessage("OK")
	status.CopyTo(span.Status())

	spanAttributes.CopyTo(span.Attributes())
	return span
}

func constructHTTPServerSpan(traceID pdata.TraceID) pdata.Span {
	attributes := make(map[string]interface{})
	attributes[conventions.AttributeHTTPMethod] = "GET"
	attributes[conventions.AttributeHTTPURL] = "https://api.example.com/users/junit"
	attributes[conventions.AttributeHTTPClientIP] = "192.168.15.32"
	attributes[conventions.AttributeHTTPStatusCode] = 200
	endTime := time.Now().Round(time.Second)
	startTime := endTime.Add(-90 * time.Second)
	spanAttributes := constructSpanAttributes(attributes)

	span := pdata.NewSpan()
	span.SetTraceID(traceID)
	span.SetSpanID(newSegmentID())
	span.SetParentSpanID(newSegmentID())
	span.SetName("/users/junit")
	span.SetKind(pdata.SpanKindServer)
	span.SetStartTimestamp(pdata.NewTimestampFromTime(startTime))
	span.SetEndTimestamp(pdata.NewTimestampFromTime(endTime))

	status := pdata.NewSpanStatus()
	status.SetCode(0)
	status.SetMessage("OK")
	status.CopyTo(span.Status())

	spanAttributes.CopyTo(span.Attributes())
	return span
}

func constructSpanAttributes(attributes map[string]interface{}) pdata.AttributeMap {
	attrs := pdata.NewAttributeMap()
	for key, value := range attributes {
		if cast, ok := value.(int); ok {
			attrs.InsertInt(key, int64(cast))
		} else if cast, ok := value.(int64); ok {
			attrs.InsertInt(key, cast)
		} else {
			attrs.InsertString(key, fmt.Sprintf("%v", value))
		}
	}
	return attrs
}

func newTraceID() pdata.TraceID {
	var r [16]byte
	epoch := time.Now().Unix()
	binary.BigEndian.PutUint32(r[0:4], uint32(epoch))
	_, err := rand.Read(r[4:])
	if err != nil {
		panic(err)
	}
	return pdata.NewTraceID(r)
}

func constructW3CTraceID() pdata.TraceID {
	var r [16]byte
	for i := range r {
		r[i] = byte(rand.Intn(128))
	}
	return pdata.NewTraceID(r)
}

func newSegmentID() pdata.SpanID {
	var r [8]byte
	_, err := rand.Read(r[:])
	if err != nil {
		panic(err)
	}
	return pdata.NewSpanID(r)
}
