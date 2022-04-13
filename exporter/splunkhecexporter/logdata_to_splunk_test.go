// Copyright 2020, OpenTelemetry Authors
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

package splunkhecexporter

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.6.1"
	"go.uber.org/zap"

	"github.com/open-telemetry/opentelemetry-collector-contrib/internal/splunk"
)

func Test_mapLogRecordToSplunkEvent(t *testing.T) {
	logger := zap.NewNop()
	ts := pdata.Timestamp(123)

	tests := []struct {
		name             string
		logRecordFn      func() pdata.LogRecord
		logResourceFn    func() pdata.Resource
		configDataFn     func() *Config
		wantSplunkEvents []*splunk.Event
	}{
		{
			name: "valid",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{"custom": "custom"},
					"myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with_name",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{"custom": "custom"},
					"myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with_hec_token",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString(splunk.HecTokenLabel, "mytoken")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{},
					"unknown", "source", "sourcetype"),
			},
		},
		{
			name: "non-string attribute",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertDouble("foo", 123)
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{"foo": float64(123)}, "myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with_config",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{"custom": "custom"}, "unknown", "source", "sourcetype"),
			},
		},
		{
			name: "with_custom_mapping",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.Attributes().InsertString("mysource", "mysource")
				logRecord.Attributes().InsertString("mysourcetype", "mysourcetype")
				logRecord.Attributes().InsertString("myindex", "myindex")
				logRecord.Attributes().InsertString("myhost", "myhost")
				logRecord.SetSeverityText("DEBUG")
				logRecord.SetSeverityNumber(pdata.SeverityNumberDEBUG)
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				return &Config{
					HecToOtelAttrs: splunk.HecToOtelAttrs{
						Source:     "mysource",
						SourceType: "mysourcetype",
						Index:      "myindex",
						Host:       "myhost",
					},
					HecFields: OtelToHecFields{
						SeverityNumber: "myseveritynum",
						SeverityText:   "myseverity",
					},
				}
			},
			wantSplunkEvents: []*splunk.Event{
				func() *splunk.Event {
					event := commonLogSplunkEvent("mylog", ts, map[string]interface{}{"custom": "custom", "myseverity": "DEBUG", "myseveritynum": pdata.SeverityNumber(5)}, "myhost", "mysource", "mysourcetype")
					event.Index = "myindex"
					return event
				}(),
			},
		},
		{
			name: "log_is_empty",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(nil, 0, map[string]interface{}{}, "unknown", "source", "sourcetype"),
			},
		},
		{
			name: "with span and trace id",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.SetSpanID(pdata.NewSpanID([8]byte{0, 0, 0, 0, 0, 0, 0, 50}))
				logRecord.SetTraceID(pdata.NewTraceID([16]byte{0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 0, 100}))
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: func() []*splunk.Event {
				event := commonLogSplunkEvent(nil, 0, map[string]interface{}{}, "unknown", "source", "sourcetype")
				event.Fields["span_id"] = "0000000000000032"
				event.Fields["trace_id"] = "00000000000000000000000000000064"
				return []*splunk.Event{event}
			}(),
		},
		{
			name: "with double body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetDoubleVal(42)
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(float64(42), ts, map[string]interface{}{"custom": "custom"}, "myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with int body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetIntVal(42)
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(int64(42), ts, map[string]interface{}{"custom": "custom"}, "myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with bool body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetBoolVal(true)
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(true, ts, map[string]interface{}{"custom": "custom"}, "myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with map body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				attVal := pdata.NewValueMap()
				attMap := attVal.MapVal()
				attMap.InsertDouble("23", 45)
				attMap.InsertString("foo", "bar")
				attVal.CopyTo(logRecord.Body())
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(map[string]interface{}{"23": float64(45), "foo": "bar"}, ts,
					map[string]interface{}{"custom": "custom"},
					"myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with nil body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent(nil, ts, map[string]interface{}{"custom": "custom"},
					"myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "with array body",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				attVal := pdata.NewValueSlice()
				attArray := attVal.SliceVal()
				attArray.AppendEmpty().SetStringVal("foo")
				attVal.CopyTo(logRecord.Body())
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent([]interface{}{"foo"}, ts, map[string]interface{}{"custom": "custom"},
					"myhost", "myapp", "myapp-type"),
			},
		},
		{
			name: "log resource attribute",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: func() pdata.Resource {
				resource := pdata.NewResource()
				resource.Attributes().InsertString("resourceAttr1", "some_string")
				resource.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type-from-resource-attr")
				resource.Attributes().InsertString(splunk.DefaultIndexLabel, "index-resource")
				resource.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp-resource")
				resource.Attributes().InsertString(conventions.AttributeHostName, "myhost-resource")
				return resource
			},
			configDataFn: func() *Config {
				return createDefaultConfig().(*Config)
			},
			wantSplunkEvents: func() []*splunk.Event {
				event := commonLogSplunkEvent("mylog", ts, map[string]interface{}{
					"resourceAttr1": "some_string",
				}, "myhost-resource", "myapp-resource", "myapp-type-from-resource-attr")
				event.Index = "index-resource"
				return []*splunk.Event{
					event,
				}
			}(),
		},
		{
			name: "with severity",
			logRecordFn: func() pdata.LogRecord {
				logRecord := pdata.NewLogRecord()
				logRecord.Body().SetStringVal("mylog")
				logRecord.Attributes().InsertString(splunk.DefaultSourceLabel, "myapp")
				logRecord.Attributes().InsertString(splunk.DefaultSourceTypeLabel, "myapp-type")
				logRecord.Attributes().InsertString(conventions.AttributeHostName, "myhost")
				logRecord.Attributes().InsertString("custom", "custom")
				logRecord.SetSeverityText("DEBUG")
				logRecord.SetSeverityNumber(pdata.SeverityNumberDEBUG)
				logRecord.SetTimestamp(ts)
				return logRecord
			},
			logResourceFn: pdata.NewResource,
			configDataFn: func() *Config {
				config := createDefaultConfig().(*Config)
				config.Source = "source"
				config.SourceType = "sourcetype"
				return config
			},
			wantSplunkEvents: []*splunk.Event{
				commonLogSplunkEvent("mylog", ts, map[string]interface{}{"custom": "custom", "otel.log.severity.number": pdata.SeverityNumberDEBUG, "otel.log.severity.text": "DEBUG"},
					"myhost", "myapp", "myapp-type"),
			},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			for _, want := range tt.wantSplunkEvents {
				config := tt.configDataFn()
				got := mapLogRecordToSplunkEvent(tt.logResourceFn(), tt.logRecordFn(), config, logger)
				assert.EqualValues(t, want, got)
			}
		})
	}
}

func commonLogSplunkEvent(
	event interface{},
	ts pdata.Timestamp,
	fields map[string]interface{},
	host string,
	source string,
	sourcetype string,
) *splunk.Event {
	return &splunk.Event{
		Time:       nanoTimestampToEpochMilliseconds(ts),
		Host:       host,
		Event:      event,
		Source:     source,
		SourceType: sourcetype,
		Fields:     fields,
	}
}

func Test_emptyLogRecord(t *testing.T) {
	event := mapLogRecordToSplunkEvent(pdata.NewResource(), pdata.NewLogRecord(), &Config{}, zap.NewNop())
	assert.Nil(t, event.Time)
	assert.Equal(t, event.Host, "unknown")
	assert.Zero(t, event.Source)
	assert.Zero(t, event.SourceType)
	assert.Zero(t, event.Index)
	assert.Nil(t, event.Event)
	assert.Empty(t, event.Fields)
}

func Test_nanoTimestampToEpochMilliseconds(t *testing.T) {
	splunkTs := nanoTimestampToEpochMilliseconds(1001000000)
	assert.Equal(t, 1.001, *splunkTs)
	splunkTs = nanoTimestampToEpochMilliseconds(1001990000)
	assert.Equal(t, 1.002, *splunkTs)
	splunkTs = nanoTimestampToEpochMilliseconds(0)
	assert.True(t, nil == splunkTs)
}
