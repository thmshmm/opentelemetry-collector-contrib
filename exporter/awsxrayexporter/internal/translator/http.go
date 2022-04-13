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

package translator // import "github.com/open-telemetry/opentelemetry-collector-contrib/exporter/awsxrayexporter/internal/translator"

import (
	"strconv"

	"github.com/aws/aws-sdk-go/aws"
	"go.opentelemetry.io/collector/model/pdata"
	conventions "go.opentelemetry.io/collector/model/semconv/v1.6.1"

	awsxray "github.com/open-telemetry/opentelemetry-collector-contrib/internal/aws/xray"
)

func makeHTTP(span pdata.Span) (map[string]pdata.Value, *awsxray.HTTPData) {
	var (
		info = awsxray.HTTPData{
			Request:  &awsxray.RequestData{},
			Response: &awsxray.ResponseData{},
		}
		filtered = make(map[string]pdata.Value)
		urlParts = make(map[string]string)
	)

	if span.Attributes().Len() == 0 {
		return filtered, nil
	}

	hasHTTP := false
	hasHTTPRequestURLAttributes := false

	span.Attributes().Range(func(key string, value pdata.Value) bool {
		switch key {
		case conventions.AttributeHTTPMethod:
			info.Request.Method = awsxray.String(value.StringVal())
			hasHTTP = true
		case conventions.AttributeHTTPClientIP:
			info.Request.ClientIP = awsxray.String(value.StringVal())
			info.Request.XForwardedFor = aws.Bool(true)
			hasHTTP = true
		case conventions.AttributeHTTPUserAgent:
			info.Request.UserAgent = awsxray.String(value.StringVal())
			hasHTTP = true
		case conventions.AttributeHTTPStatusCode:
			info.Response.Status = aws.Int64(value.IntVal())
			hasHTTP = true
		case conventions.AttributeHTTPURL:
			urlParts[key] = value.StringVal()
			hasHTTP = true
			hasHTTPRequestURLAttributes = true
		case conventions.AttributeHTTPScheme:
			urlParts[key] = value.StringVal()
			hasHTTP = true
		case conventions.AttributeHTTPHost:
			urlParts[key] = value.StringVal()
			hasHTTP = true
			hasHTTPRequestURLAttributes = true
		case conventions.AttributeHTTPTarget:
			urlParts[key] = value.StringVal()
			hasHTTP = true
		case conventions.AttributeHTTPServerName:
			urlParts[key] = value.StringVal()
			hasHTTP = true
			hasHTTPRequestURLAttributes = true
		case conventions.AttributeNetHostPort:
			urlParts[key] = value.StringVal()
			hasHTTP = true
			if len(urlParts[key]) == 0 {
				urlParts[key] = strconv.FormatInt(value.IntVal(), 10)
			}
		case conventions.AttributeHostName:
			urlParts[key] = value.StringVal()
			hasHTTPRequestURLAttributes = true
		case conventions.AttributeNetHostName:
			urlParts[key] = value.StringVal()
			hasHTTPRequestURLAttributes = true
		case conventions.AttributeNetPeerName:
			urlParts[key] = value.StringVal()
		case conventions.AttributeNetPeerPort:
			urlParts[key] = value.StringVal()
			if len(urlParts[key]) == 0 {
				urlParts[key] = strconv.FormatInt(value.IntVal(), 10)
			}
		case conventions.AttributeNetPeerIP:
			// Prefer HTTP forwarded information (AttributeHTTPClientIP) when present.
			if info.Request.ClientIP == nil {
				info.Request.ClientIP = awsxray.String(value.StringVal())
			}
			urlParts[key] = value.StringVal()
			hasHTTPRequestURLAttributes = true
		default:
			filtered[key] = value
		}
		return true
	})

	if !hasHTTP {
		// Didn't have any HTTP-specific information so don't need to fill it in segment
		return filtered, nil
	}

	if hasHTTPRequestURLAttributes {
		if span.Kind() == pdata.SpanKindServer {
			info.Request.URL = awsxray.String(constructServerURL(urlParts))
		} else {
			info.Request.URL = awsxray.String(constructClientURL(urlParts))
		}
	}

	info.Response.ContentLength = aws.Int64(extractResponseSizeFromEvents(span))

	return filtered, &info
}

func extractResponseSizeFromEvents(span pdata.Span) int64 {
	// Support insrumentation that sets response size in span or as an event.
	size := extractResponseSizeFromAttributes(span.Attributes())
	if size != 0 {
		return size
	}
	for i := 0; i < span.Events().Len(); i++ {
		event := span.Events().At(i)
		size = extractResponseSizeFromAttributes(event.Attributes())
		if size != 0 {
			return size
		}
	}
	return size
}

func extractResponseSizeFromAttributes(attributes pdata.Map) int64 {
	typeVal, ok := attributes.Get("message.type")
	if ok && typeVal.StringVal() == "RECEIVED" {
		if sizeVal, ok := attributes.Get(conventions.AttributeMessagingMessagePayloadSizeBytes); ok {
			return sizeVal.IntVal()
		}
	}
	return 0
}

func constructClientURL(urlParts map[string]string) string {
	// follows OpenTelemetry specification-defined combinations for client spans described in
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/http.md#http-client

	url, ok := urlParts[conventions.AttributeHTTPURL]
	if ok {
		// full URL available so no need to assemble
		return url
	}

	scheme, ok := urlParts[conventions.AttributeHTTPScheme]
	if !ok {
		scheme = "http"
	}
	port := ""
	host, ok := urlParts[conventions.AttributeHTTPHost]
	if !ok {
		host, ok = urlParts[conventions.AttributeNetPeerName]
		if !ok {
			host = urlParts[conventions.AttributeNetPeerIP]
		}
		port, ok = urlParts[conventions.AttributeNetPeerPort]
		if !ok {
			port = ""
		}
	}
	url = scheme + "://" + host
	if len(port) > 0 && !(scheme == "http" && port == "80") && !(scheme == "https" && port == "443") {
		url += ":" + port
	}
	target, ok := urlParts[conventions.AttributeHTTPTarget]
	if ok {
		url += target
	} else {
		url += "/"
	}
	return url
}

func constructServerURL(urlParts map[string]string) string {
	// follows OpenTelemetry specification-defined combinations for server spans described in
	// https://github.com/open-telemetry/opentelemetry-specification/blob/main/specification/trace/semantic_conventions/http.md#http-server-semantic-conventions

	url, ok := urlParts[conventions.AttributeHTTPURL]
	if ok {
		// full URL available so no need to assemble
		return url
	}

	scheme, ok := urlParts[conventions.AttributeHTTPScheme]
	if !ok {
		scheme = "http"
	}
	port := ""
	host, ok := urlParts[conventions.AttributeHTTPHost]
	if !ok {
		host, ok = urlParts[conventions.AttributeHTTPServerName]
		if !ok {
			host, ok = urlParts[conventions.AttributeNetHostName]
			if !ok {
				host = urlParts[conventions.AttributeHostName]
			}
		}
		port, ok = urlParts[conventions.AttributeNetHostPort]
		if !ok {
			port = ""
		}
	}
	url = scheme + "://" + host
	if len(port) > 0 && !(scheme == "http" && port == "80") && !(scheme == "https" && port == "443") {
		url += ":" + port
	}
	target, ok := urlParts[conventions.AttributeHTTPTarget]
	if ok {
		url += target
	} else {
		url += "/"
	}
	return url
}
