// Copyright The OpenTelemetry Authors
// SPDX-License-Identifier: Apache-2.0

package avrologencodingextension // import "github.com/open-telemetry/opentelemetry-collector-contrib/extension/encoding/avrologencodingextension"

import (
	"fmt"

	"github.com/linkedin/goavro/v2"
)

type avroSerDe interface {
	Serialize([]byte) ([]byte, error)
	Deserialize([]byte) (map[string]any, error)
}

type avroStaticSchemaSerDe struct {
	codec *goavro.Codec
}

func newAVROStaticSchemaSerDe(schema string) (avroSerDe, error) {
	codec, err := goavro.NewCodec(schema)
	if err != nil {
		return nil, fmt.Errorf("failed to create avro codec: %w", err)
	}

	return &avroStaticSchemaSerDe{
		codec: codec,
	}, nil
}

func (d *avroStaticSchemaSerDe) Serialize(data []byte) ([]byte, error) {
	native, _, err := d.codec.NativeFromTextual(data)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize native from textual avro: %w", err)
	}

	avroBinary, err := d.codec.BinaryFromNative(nil, native)
	if err != nil {
		return nil, fmt.Errorf("failed to serialize binary from native avro: %w", err)
	}

	return avroBinary, nil
}

func (d *avroStaticSchemaSerDe) Deserialize(data []byte) (map[string]any, error) {
	native, _, err := d.codec.NativeFromBinary(data)
	if err != nil {
		return nil, fmt.Errorf("failed to deserialize avro record: %w", err)
	}

	return native.(map[string]any), nil
}
