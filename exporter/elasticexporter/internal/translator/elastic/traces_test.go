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

package elastic_test

import (
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.elastic.co/apm/model"
	"go.elastic.co/apm/transport/transporttest"
	"go.elastic.co/fastjson"
	"go.opentelemetry.io/collector/model/pdata"

	"github.com/open-telemetry/opentelemetry-collector-contrib/exporter/elasticexporter/internal/translator/elastic"
)

func TestEncodeSpan(t *testing.T) {
	var w fastjson.Writer
	var recorder transporttest.RecorderTransport
	elastic.EncodeResourceMetadata(pdata.NewResource(), &w)

	traceID := model.TraceID{1, 2, 3, 4, 5, 6, 7, 8, 9, 10, 11, 12, 13, 14, 15, 16}
	rootTransactionID := model.SpanID{1, 1, 1, 1, 1, 1, 1, 1}
	clientSpanID := model.SpanID{2, 2, 2, 2, 2, 2, 2, 2}
	serverTransactionID := model.SpanID{3, 3, 3, 3, 3, 3, 3, 3}

	startTime := time.Unix(123, 0).UTC()
	endTime := startTime.Add(time.Millisecond * 5)

	rootSpan := pdata.NewSpan()
	rootSpan.SetSpanID(pdata.NewSpanID(rootTransactionID))
	rootSpan.SetName("root_span")
	rootSpan.Attributes().InsertString("string.attr", "string_value")
	rootSpan.Attributes().InsertInt("int.attr", 123)
	rootSpan.Attributes().InsertDouble("double.attr", 123.456)
	rootSpan.Attributes().InsertBool("bool.attr", true)

	clientSpan := pdata.NewSpan()
	clientSpan.SetSpanID(pdata.NewSpanID(clientSpanID))
	clientSpan.SetParentSpanID(pdata.NewSpanID(rootTransactionID))
	clientSpan.SetKind(pdata.SpanKindClient)
	clientSpan.SetName("client_span")
	clientSpan.Status().SetCode(pdata.StatusCodeError)
	clientSpan.Attributes().InsertString("string.attr", "string_value")
	clientSpan.Attributes().InsertInt("int.attr", 123)
	clientSpan.Attributes().InsertDouble("double.attr", 123.456)
	clientSpan.Attributes().InsertBool("bool.attr", true)

	serverSpan := pdata.NewSpan()
	serverSpan.SetSpanID(pdata.NewSpanID(serverTransactionID))
	serverSpan.SetParentSpanID(pdata.NewSpanID(clientSpanID))
	serverSpan.SetKind(pdata.SpanKindServer)
	serverSpan.SetName("server_span")
	serverSpan.Status().SetCode(pdata.StatusCodeOk)

	for _, span := range []pdata.Span{rootSpan, clientSpan, serverSpan} {
		span.SetTraceID(pdata.NewTraceID(traceID))
		span.SetStartTimestamp(pdata.NewTimestampFromTime(startTime))
		span.SetEndTimestamp(pdata.NewTimestampFromTime(endTime))
	}

	for _, span := range []pdata.Span{rootSpan, clientSpan, serverSpan} {
		err := elastic.EncodeSpan(span, pdata.NewInstrumentationScope(), pdata.NewResource(), &w)
		require.NoError(t, err)
	}
	sendStream(t, &w, &recorder)

	payloads := recorder.Payloads()
	assert.Equal(t, []model.Transaction{{
		TraceID:   traceID,
		ID:        rootTransactionID,
		Timestamp: model.Time(startTime),
		Duration:  5.0,
		Name:      "root_span",
		Type:      "unknown",
		Context: &model.Context{
			Tags: model.IfaceMap{{
				Key:   "bool_attr",
				Value: true,
			}, {
				Key:   "double_attr",
				Value: 123.456,
			}, {
				Key:   "int_attr",
				Value: float64(123),
			}, {
				Key:   "string_attr",
				Value: "string_value",
			}},
		},
	}, {
		TraceID:   traceID,
		ID:        serverTransactionID,
		ParentID:  clientSpanID,
		Timestamp: model.Time(startTime),
		Duration:  5.0,
		Name:      "server_span",
		Type:      "unknown",
		Result:    "OK",
		Outcome:   "success",
	}}, payloads.Transactions)

	assert.Equal(t, []model.Span{{
		TraceID:   traceID,
		ID:        clientSpanID,
		ParentID:  rootTransactionID,
		Timestamp: model.Time(startTime),
		Duration:  5.0,
		Name:      "client_span",
		Type:      "app",
		Context: &model.SpanContext{
			Tags: model.IfaceMap{{
				Key:   "bool_attr",
				Value: true,
			}, {
				Key:   "double_attr",
				Value: 123.456,
			}, {
				Key:   "int_attr",
				Value: float64(123),
			}, {
				Key:   "string_attr",
				Value: "string_value",
			}},
		},
		Outcome: "failure",
	}}, payloads.Spans)

	assert.Empty(t, payloads.Errors)
}

func TestEncodeSpanStatus(t *testing.T) {
	testStatusCode := func(t *testing.T, statusCode pdata.StatusCode, expectedResult, expectedOutcome string) {
		t.Helper()

		var w fastjson.Writer
		var recorder transporttest.RecorderTransport
		elastic.EncodeResourceMetadata(pdata.NewResource(), &w)

		span := pdata.NewSpan()
		span.SetTraceID(pdata.NewTraceID([16]byte{1}))
		span.SetSpanID(pdata.NewSpanID([8]byte{1}))
		span.SetName("span")

		if statusCode >= 0 {
			span.Status().SetCode(statusCode)
		}

		err := elastic.EncodeSpan(span, pdata.NewInstrumentationScope(), pdata.NewResource(), &w)
		require.NoError(t, err)
		sendStream(t, &w, &recorder)
		payloads := recorder.Payloads()
		require.Len(t, payloads.Transactions, 1)
		assert.Equal(t, expectedResult, payloads.Transactions[0].Result)
		assert.Equal(t, expectedOutcome, payloads.Transactions[0].Outcome)
	}

	testStatusCode(t, -1, "", "")
	testStatusCode(t, pdata.StatusCodeUnset, "", "")
	testStatusCode(t, pdata.StatusCodeOk, "OK", "success")
	testStatusCode(t, pdata.StatusCodeError, "Error", "failure")
}

func TestEncodeSpanTruncation(t *testing.T) {
	span := pdata.NewSpan()
	span.SetName(strings.Repeat("x", 1300))

	var w fastjson.Writer
	var recorder transporttest.RecorderTransport
	elastic.EncodeResourceMetadata(pdata.NewResource(), &w)
	err := elastic.EncodeSpan(span, pdata.NewInstrumentationScope(), pdata.NewResource(), &w)
	require.NoError(t, err)
	sendStream(t, &w, &recorder)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Equal(t, strings.Repeat("x", 1024), payloads.Transactions[0].Name)
}

func TestTransactionHTTPRequestURL(t *testing.T) {
	test := func(t *testing.T, expectedFull string, attrs map[string]interface{}) {
		transaction := transactionWithAttributes(t, attrs)
		assert.Equal(t, expectedFull, transaction.Context.Request.URL.Full)
	}
	t.Run("scheme_host_target", func(t *testing.T) {
		test(t, "https://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.scheme": "https",
			"http.host":   "testing.invalid:80",
			"http.target": "/foo?bar",
		})
	})
	t.Run("scheme_servername_nethostport_target", func(t *testing.T) {
		test(t, "https://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.scheme":      "https",
			"http.server_name": "testing.invalid",
			"net.host.port":    80,
			"http.target":      "/foo?bar",
		})
	})
	t.Run("scheme_nethostname_nethostport_target", func(t *testing.T) {
		test(t, "https://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.scheme":   "https",
			"net.host.name": "testing.invalid",
			"net.host.port": 80,
			"http.target":   "/foo?bar",
		})
	})
	t.Run("http.url", func(t *testing.T) {
		const httpURL = "https://testing.invalid:80/foo?bar"
		test(t, httpURL, map[string]interface{}{
			"http.url": httpURL,
		})
	})
	t.Run("host_no_port", func(t *testing.T) {
		test(t, "https://testing.invalid/foo?bar", map[string]interface{}{
			"http.scheme": "https",
			"http.host":   "testing.invalid",
			"http.target": "/foo?bar",
		})
	})
	t.Run("ipv6_host_no_port", func(t *testing.T) {
		test(t, "https://[::1]/foo?bar", map[string]interface{}{
			"http.scheme": "https",
			"http.host":   "[::1]",
			"http.target": "/foo?bar",
		})
	})

	// Scheme is set to "http" if it can't be deduced from attributes.
	t.Run("default_scheme", func(t *testing.T) {
		test(t, "http://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.host":   "testing.invalid:80",
			"http.target": "/foo?bar",
		})
	})
}

func TestTransactionHTTPRequestURLInvalid(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.url": "0.0.0.0:8081",
	})
	require.NotNil(t, transaction.Context)
	assert.Nil(t, transaction.Context.Request)
	assert.Equal(t, model.IfaceMap{
		{Key: "http_url", Value: "0.0.0.0:8081"},
	}, transaction.Context.Tags)
}

func TestTransactionHTTPRequestSocketRemoteAddr(t *testing.T) {
	test := func(t *testing.T, expected string, attrs map[string]interface{}) {
		transaction := transactionWithAttributes(t, attrs)
		assert.Equal(t, expected, transaction.Context.Request.Socket.RemoteAddress)
	}
	t.Run("net.peer.ip_port", func(t *testing.T) {
		test(t, "192.168.0.1:1234", map[string]interface{}{
			"http.url":      "http://testing.invalid",
			"net.peer.ip":   "192.168.0.1",
			"net.peer.port": 1234,
		})
	})
	t.Run("net.peer.ip", func(t *testing.T) {
		test(t, "192.168.0.1", map[string]interface{}{
			"http.url":    "http://testing.invalid",
			"net.peer.ip": "192.168.0.1",
		})
	})
	t.Run("http.remote_addr", func(t *testing.T) {
		test(t, "192.168.0.1:1234", map[string]interface{}{
			"http.url":         "http://testing.invalid",
			"http.remote_addr": "192.168.0.1:1234",
		})
	})
	t.Run("http.remote_addr_no_port", func(t *testing.T) {
		test(t, "192.168.0.1", map[string]interface{}{
			"http.url":         "http://testing.invalid",
			"http.remote_addr": "192.168.0.1",
		})
	})
}

func TestTransactionHTTPRequestHTTPVersion(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.flavor": "1.1",
	})
	assert.Equal(t, "1.1", transaction.Context.Request.HTTPVersion)
}

func TestTransactionHTTPRequestHTTPMethod(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.method": "PATCH",
	})
	assert.Equal(t, "PATCH", transaction.Context.Request.Method)
}

func TestTransactionHTTPRequestUserAgent(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.user_agent": "Foo/bar (baz)",
	})
	assert.Equal(t, model.Headers{{
		Key:    "User-Agent",
		Values: []string{"Foo/bar (baz)"},
	}}, transaction.Context.Request.Headers)
}

func TestTransactionHTTPRequestClientIP(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.client_ip": "256.257.258.259",
	})
	assert.Equal(t, model.Headers{{
		Key:    "X-Forwarded-For",
		Values: []string{"256.257.258.259"},
	}}, transaction.Context.Request.Headers)
}

func TestTransactionHTTPResponseStatusCode(t *testing.T) {
	transaction := transactionWithAttributes(t, map[string]interface{}{
		"http.status_code": 200,
	})
	assert.Equal(t, 200, transaction.Context.Response.StatusCode)
}

func TestSpanHTTPURL(t *testing.T) {
	test := func(t *testing.T, expectedURL string, attrs map[string]interface{}) {
		span := spanWithAttributes(t, attrs)
		assert.Equal(t, expectedURL, span.Context.HTTP.URL.String())
	}
	t.Run("http.url", func(t *testing.T) {
		const httpURL = "https://testing.invalid:80/foo?bar"
		test(t, httpURL, map[string]interface{}{
			"http.url": httpURL,
		})
	})
	t.Run("scheme_host_target", func(t *testing.T) {
		test(t, "https://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.scheme": "https",
			"http.host":   "testing.invalid:80",
			"http.target": "/foo?bar",
		})
	})
	t.Run("scheme_netpeername_netpeerport_target", func(t *testing.T) {
		test(t, "https://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.scheme":   "https",
			"net.peer.name": "testing.invalid",
			"net.peer.ip":   "::1", // net.peer.name preferred
			"net.peer.port": 80,
			"http.target":   "/foo?bar",
		})
	})
	t.Run("scheme_netpeerip_netpeerport_target", func(t *testing.T) {
		test(t, "https://[::1]:80/foo?bar", map[string]interface{}{
			"http.scheme":   "https",
			"net.peer.ip":   "::1",
			"net.peer.port": 80,
			"http.target":   "/foo?bar",
		})
	})

	// Scheme is set to "http" if it can't be deduced from attributes.
	t.Run("default_scheme", func(t *testing.T) {
		test(t, "http://testing.invalid:80/foo?bar", map[string]interface{}{
			"http.host":   "testing.invalid:80",
			"http.target": "/foo?bar",
		})
	})
}

func TestSpanHTTPDestination(t *testing.T) {
	test := func(t *testing.T, expectedAddr string, expectedPort int, expectedName string, expectedResource string,
		attrs map[string]interface{}) {
		span := spanWithAttributes(t, attrs)
		assert.Equal(t, &model.DestinationSpanContext{
			Address: expectedAddr,
			Port:    expectedPort,
			Service: &model.DestinationServiceSpanContext{
				Type:     "external",
				Name:     expectedName,
				Resource: expectedResource,
			},
		}, span.Context.Destination)
	}
	t.Run("url_default_port_specified", func(t *testing.T) {
		test(t, "testing.invalid", 443, "https://testing.invalid", "testing.invalid:443", map[string]interface{}{
			"http.url": "https://testing.invalid:443/foo?bar",
		})
	})
	t.Run("url_port_scheme", func(t *testing.T) {
		test(t, "testing.invalid", 443, "https://testing.invalid", "testing.invalid:443", map[string]interface{}{
			"http.url": "https://testing.invalid/foo?bar",
		})
	})
	t.Run("url_non_default_port", func(t *testing.T) {
		test(t, "testing.invalid", 444, "https://testing.invalid:444", "testing.invalid:444", map[string]interface{}{
			"http.url": "https://testing.invalid:444/foo?bar",
		})
	})
	t.Run("scheme_host_target", func(t *testing.T) {
		test(t, "testing.invalid", 444, "https://testing.invalid:444", "testing.invalid:444", map[string]interface{}{
			"http.scheme": "https",
			"http.host":   "testing.invalid:444",
			"http.target": "/foo?bar",
		})
	})
	t.Run("scheme_netpeername_nethostport_target", func(t *testing.T) {
		test(t, "::1", 444, "https://[::1]:444", "[::1]:444", map[string]interface{}{
			"http.scheme":   "https",
			"net.peer.ip":   "::1",
			"net.peer.port": 444,
			"http.target":   "/foo?bar",
		})
	})
}

func TestSpanHTTPURLInvalid(t *testing.T) {
	span := spanWithAttributes(t, map[string]interface{}{
		"http.url": "0.0.0.0:8081",
	})
	require.NotNil(t, span.Context)
	assert.Nil(t, span.Context.HTTP)
	assert.Equal(t, model.IfaceMap{
		{Key: "http_url", Value: "0.0.0.0:8081"},
	}, span.Context.Tags)
}

func TestSpanHTTPStatusCode(t *testing.T) {
	span := spanWithAttributes(t, map[string]interface{}{
		"http.status_code": 200,
	})
	assert.Equal(t, 200, span.Context.HTTP.StatusCode)
}

func TestSpanDatabaseContext(t *testing.T) {
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/database.md#mysql
	connectionString := "Server=shopdb.example.com;Database=ShopDb;Uid=billing_user;TableCache=true;UseCompression=True;MinimumPoolSize=10;MaximumPoolSize=50;"
	span := spanWithAttributes(t, map[string]interface{}{
		"db.system":            "mysql",
		"db.connection_string": connectionString,
		"db.user":              "billing_user",
		"db.name":              "ShopDb",
		"db.statement":         "SELECT * FROM orders WHERE order_id = 'o4711'",
		"net.peer.name":        "shopdb.example.com",
		"net.peer.ip":          "192.0.2.12",
		"net.peer.port":        3306,
		"net.transport":        "IP.TCP",
	})

	assert.Equal(t, "db", span.Type)
	assert.Equal(t, "mysql", span.Subtype)
	assert.Equal(t, "", span.Action)

	assert.Equal(t, &model.DatabaseSpanContext{
		Type:      "mysql",
		Instance:  "ShopDb",
		Statement: "SELECT * FROM orders WHERE order_id = 'o4711'",
		User:      "billing_user",
	}, span.Context.Database)

	assert.Equal(t, model.IfaceMap{
		{Key: "db_connection_string", Value: connectionString},
		{Key: "net_transport", Value: "IP.TCP"},
	}, span.Context.Tags)

	assert.Equal(t, &model.DestinationSpanContext{
		Address: "shopdb.example.com",
		Port:    3306,
		Service: &model.DestinationServiceSpanContext{
			Type:     "db",
			Name:     "mysql",
			Resource: "mysql",
		},
	}, span.Context.Destination)
}

func TestInstrumentationLibrary(t *testing.T) {
	var w fastjson.Writer
	var recorder transporttest.RecorderTransport

	span := pdata.NewSpan()
	span.SetName("root_span")

	library := pdata.NewInstrumentationScope()
	library.SetName("library-name")
	library.SetVersion("1.2.3")

	resource := pdata.NewResource()
	elastic.EncodeResourceMetadata(resource, &w)
	err := elastic.EncodeSpan(span, library, resource, &w)
	assert.NoError(t, err)
	sendStream(t, &w, &recorder)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	assert.Equal(t, &model.Context{
		Service: &model.Service{
			Framework: &model.Framework{
				Name:    "library-name",
				Version: "1.2.3",
			},
		},
	}, payloads.Transactions[0].Context)
}

func transactionWithAttributes(t *testing.T, attrs map[string]interface{}) model.Transaction {
	var w fastjson.Writer
	var recorder transporttest.RecorderTransport

	span := pdata.NewSpan()
	pdata.NewMapFromRaw(attrs).CopyTo(span.Attributes())

	resource := pdata.NewResource()
	elastic.EncodeResourceMetadata(resource, &w)
	err := elastic.EncodeSpan(span, pdata.NewInstrumentationScope(), resource, &w)
	assert.NoError(t, err)
	sendStream(t, &w, &recorder)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Transactions, 1)
	return payloads.Transactions[0]
}

func spanWithAttributes(t *testing.T, attrs map[string]interface{}) model.Span {
	var w fastjson.Writer
	var recorder transporttest.RecorderTransport

	span := pdata.NewSpan()
	span.SetParentSpanID(pdata.NewSpanID([8]byte{1}))
	pdata.NewMapFromRaw(attrs).CopyTo(span.Attributes())

	resource := pdata.NewResource()
	elastic.EncodeResourceMetadata(resource, &w)
	err := elastic.EncodeSpan(span, pdata.NewInstrumentationScope(), resource, &w)
	assert.NoError(t, err)
	sendStream(t, &w, &recorder)

	payloads := recorder.Payloads()
	require.Len(t, payloads.Spans, 1)
	return payloads.Spans[0]
}
