// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package avrologencodingextension

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.opentelemetry.io/collector/component/componenttest"
	"go.opentelemetry.io/collector/pdata/plog"
)

const testJsonBody = "{\"count\":5,\"hostname\":\"host1\",\"level\":\"warn\",\"levelEnum\":\"INFO\",\"mapField\":{},\"message\":\"log message\",\"nestedRecord\":{\"field1\":12,\"field2\":\"val2\"},\"properties\":[\"prop1\",\"prop2\"],\"severity\":1,\"timestamp\":1697187201488000000}"

func TestExtension_Start_Shutdown(t *testing.T) {
	avroExtention := &avroLogExtension{}

	err := avroExtention.Start(context.Background(), componenttest.NewNopHost())
	require.NoError(t, err)

	err = avroExtention.Shutdown(context.Background())
	require.NoError(t, err)
}

func TestMarshal(t *testing.T) {
	t.Parallel()

	schema, _ := createAVROTestData(t)

	e, err := newExtension(&Config{Schema: schema})
	assert.NoError(t, err)

	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr(testJsonBody)

	_, err = e.MarshalLogs(ld)

	assert.NoError(t, err)
}

func TestInvalidMarshal(t *testing.T) {
	t.Parallel()

	schema, err := loadAVROSchemaFromFile("testdata/schema1.avro")
	if err != nil {
		t.Fatalf("Failed to read avro schema file: %q", err.Error())
	}

	e, err := newExtension(&Config{Schema: string(schema)})
	assert.NoError(t, err)

	ld := plog.NewLogs()
	ld.ResourceLogs().AppendEmpty().ScopeLogs().AppendEmpty().LogRecords().AppendEmpty().Body().SetStr("INVALID")

	_, err = e.MarshalLogs(ld)
	assert.Error(t, err)
}

func TestUnmarshal(t *testing.T) {
	t.Parallel()

	schema, data := createAVROTestData(t)

	e, err := newExtension(&Config{Schema: schema})
	assert.NoError(t, err)

	logs, err := e.UnmarshalLogs(data)
	logRecord := logs.ResourceLogs().At(0).ScopeLogs().At(0).LogRecords().At(0)

	assert.NoError(t, err)
	assert.Equal(t, testJsonBody, logRecord.Body().AsString())
}

func TestInvalidUnmarshal(t *testing.T) {
	t.Parallel()

	schema, err := loadAVROSchemaFromFile("testdata/schema1.avro")
	if err != nil {
		t.Fatalf("Failed to read avro schema file: %q", err.Error())
	}

	e, err := newExtension(&Config{Schema: string(schema)})
	assert.NoError(t, err)

	_, err = e.UnmarshalLogs([]byte("NOT A AVRO"))
	assert.Error(t, err)
}
