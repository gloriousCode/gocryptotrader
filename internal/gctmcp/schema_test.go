package gctmcp

import (
	"maps"
	"testing"

	"github.com/google/jsonschema-go/jsonschema"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/thrasher-corp/gocryptotrader/encoding/json"
	"github.com/thrasher-corp/gocryptotrader/gctrpc"
	"google.golang.org/protobuf/reflect/protodesc"
	"google.golang.org/protobuf/reflect/protoregistry"
	"google.golang.org/protobuf/types/descriptorpb"
	"google.golang.org/protobuf/types/known/timestamppb"
)

func TestRequestSchema(t *testing.T) {
	t.Parallel()
	methods := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods()
	for i := range methods.Len() {
		method := methods.Get(i)
		t.Run(string(method.Name()), func(t *testing.T) {
			t.Parallel()
			encoded, err := json.Marshal(requestSchema(method))
			require.NoError(t, err, "request schema must serialise")
			var schema jsonschema.Schema
			require.NoError(t, json.Unmarshal(encoded, &schema), "request schema must decode as JSON Schema")
			resolved, err := schema.Resolve(nil)
			require.NoError(t, err, "every RPC request schema must resolve all references")
			input := make(map[string]any)
			for _, name := range requiredMessageFields[string(method.Name())] {
				input[method.Input().Fields().ByName(name).JSONName()] = map[string]any{}
			}
			assert.NoError(t, resolved.Validate(map[string]any{"request": input}), "request with required nested messages should validate")
			assert.Error(t, resolved.Validate(map[string]any{}), "request object should be required")
			unknown := maps.Clone(input)
			unknown["unknownField"] = true
			assert.Error(t, resolved.Validate(map[string]any{"request": unknown}), "unknown request fields should be rejected")
			for _, name := range requiredMessageFields[string(method.Name())] {
				jsonName := method.Input().Fields().ByName(name).JSONName()
				missing := maps.Clone(input)
				delete(missing, jsonName)
				assert.Error(t, resolved.Validate(map[string]any{"request": missing}), "missing required nested message should be rejected")
				null := maps.Clone(input)
				null[jsonName] = nil
				assert.Error(t, resolved.Validate(map[string]any{"request": null}), "null required nested message should be rejected")
			}
			for _, limit := range []any{1, 1000, 0, 1001, 1.5} {
				err = resolved.Validate(map[string]any{"request": input, "maxMessages": limit})
				if method.IsStreamingServer() && (limit == 1 || limit == 1000) {
					assert.NoError(t, err, "stream limit within the advertised range should validate")
				} else {
					assert.Error(t, err, "invalid limits and unary stream options should be rejected")
				}
			}
		})
	}
	t.Run("nested candle command", func(t *testing.T) {
		t.Parallel()
		encoded, err := json.Marshal(requestSchema(methods.ByName("GetHistoricCandles")))
		require.NoError(t, err, "candle schema must serialise")
		var schema jsonschema.Schema
		require.NoError(t, json.Unmarshal(encoded, &schema), "candle schema must decode")
		resolved, err := schema.Resolve(nil)
		require.NoError(t, err, "candle schema must resolve")
		assert.NoError(t, resolved.Validate(map[string]any{"request": map[string]any{
			"exchange": "okx", "assetType": "spot", "pair": map[string]any{"base": "BTC", "quote": "USDT", "delimiter": "-"},
			"start": "2026-11-12 00:00:00 UTC", "end": "2026-11-12 01:00:00 UTC", "timeInterval": "3600000000000", "sync": false,
		}}), "representative nested command should validate")
		for _, name := range []string{"useDb", "sync", "fillMissingWithTrades", "force"} {
			assert.Error(t, resolved.Validate(map[string]any{"request": map[string]any{name: true}}), "database history options should be excluded by the candle schema")
			assert.NoError(t, resolved.Validate(map[string]any{"request": map[string]any{name: false}}), "disabled database history options should preserve exchange candle access")
		}
	})
}

func TestMessageSchema(t *testing.T) {
	t.Parallel()
	t.Run("candle database options are false", func(t *testing.T) {
		t.Parallel()
		definitions := make(map[string]map[string]any)
		message := gctrpc.File_rpc_proto.Services().ByName("GoCryptoTraderService").Methods().ByName("GetHistoricCandles").Input()
		messageSchema(message, definitions)
		properties, ok := definitions[string(message.FullName())]["properties"].(map[string]any)
		require.True(t, ok, "candle schema must expose properties")
		for _, name := range []string{"useDb", "sync", "fillMissingWithTrades", "force"} {
			assert.Equal(t, map[string]any{schemaType: "boolean", "const": false}, properties[name], "candle history database options should only accept false")
		}
		assert.Equal(t, map[string]any{schemaType: "boolean"}, properties["exRequest"], "exchange-backed extended requests should remain available")
	})
	t.Run("recursive messages and protobuf field kinds", func(t *testing.T) {
		t.Parallel()
		field := func(name string, number int32, kind descriptorpb.FieldDescriptorProto_Type, typeName string, repeated bool) *descriptorpb.FieldDescriptorProto {
			result := &descriptorpb.FieldDescriptorProto{Name: new(name), Number: new(number), Type: kind.Enum(), Label: descriptorpb.FieldDescriptorProto_LABEL_OPTIONAL.Enum()}
			if typeName != "" {
				result.TypeName = new(typeName)
			}
			if repeated {
				result.Label = descriptorpb.FieldDescriptorProto_LABEL_REPEATED.Enum()
			}
			return result
		}
		file, err := protodesc.NewFile(&descriptorpb.FileDescriptorProto{
			Name: new("mcp_schema_test.proto"), Package: new("mcp_schema_test"), Syntax: new("proto3"),
			Dependency: []string{"google/protobuf/timestamp.proto"},
			EnumType: []*descriptorpb.EnumDescriptorProto{{Name: new("Mode"), Value: []*descriptorpb.EnumValueDescriptorProto{
				{Name: new("UNKNOWN"), Number: new(int32(0))}, {Name: new("ACTIVE"), Number: new(int32(1))},
			}}},
			MessageType: []*descriptorpb.DescriptorProto{{
				Name: new("Node"),
				Field: []*descriptorpb.FieldDescriptorProto{
					field("enabled", 1, descriptorpb.FieldDescriptorProto_TYPE_BOOL, "", false),
					field("title", 2, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
					field("payload", 3, descriptorpb.FieldDescriptorProto_TYPE_BYTES, "", false),
					field("child", 4, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".mcp_schema_test.Node", false),
					field("children", 5, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".mcp_schema_test.Node", true),
					field("state", 6, descriptorpb.FieldDescriptorProto_TYPE_ENUM, ".mcp_schema_test.Mode", false),
					field("signed", 7, descriptorpb.FieldDescriptorProto_TYPE_INT64, "", false),
					field("unsigned", 8, descriptorpb.FieldDescriptorProto_TYPE_UINT64, "", false),
					field("ratio", 9, descriptorpb.FieldDescriptorProto_TYPE_DOUBLE, "", false),
					field("count", 10, descriptorpb.FieldDescriptorProto_TYPE_INT32, "", false),
					field("labels", 11, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".mcp_schema_test.Node.LabelsEntry", true),
					field("observed_at", 12, descriptorpb.FieldDescriptorProto_TYPE_MESSAGE, ".google.protobuf.Timestamp", false),
					field("signed_zigzag", 13, descriptorpb.FieldDescriptorProto_TYPE_SINT64, "", false),
					field("signed_fixed", 14, descriptorpb.FieldDescriptorProto_TYPE_SFIXED64, "", false),
					field("unsigned_fixed", 15, descriptorpb.FieldDescriptorProto_TYPE_FIXED64, "", false),
					field("single_ratio", 16, descriptorpb.FieldDescriptorProto_TYPE_FLOAT, "", false),
					field("unsigned_count", 17, descriptorpb.FieldDescriptorProto_TYPE_UINT32, "", false),
					field("zigzag_count", 18, descriptorpb.FieldDescriptorProto_TYPE_SINT32, "", false),
					field("fixed_count", 19, descriptorpb.FieldDescriptorProto_TYPE_SFIXED32, "", false),
					field("fixed_unsigned_count", 20, descriptorpb.FieldDescriptorProto_TYPE_FIXED32, "", false),
				},
				NestedType: []*descriptorpb.DescriptorProto{{
					Name: new("LabelsEntry"), Options: &descriptorpb.MessageOptions{MapEntry: new(true)},
					Field: []*descriptorpb.FieldDescriptorProto{
						field("key", 1, descriptorpb.FieldDescriptorProto_TYPE_STRING, "", false),
						field("value", 2, descriptorpb.FieldDescriptorProto_TYPE_BOOL, "", false),
					},
				}},
			}},
		}, protoregistry.GlobalFiles)
		require.NoError(t, err, "synthetic protobuf contract must be valid")
		definitions := make(map[string]map[string]any)
		root := messageSchema(file.Messages().ByName("Node"), definitions)
		root["$defs"] = definitions
		encoded, err := json.Marshal(root)
		require.NoError(t, err, "recursive schema must remain finite and serialisable")
		var schema jsonschema.Schema
		require.NoError(t, json.Unmarshal(encoded, &schema), "message schema must decode")
		resolved, err := schema.Resolve(nil)
		require.NoError(t, err, "recursive references must resolve")
		for _, tc := range []struct {
			name  string
			input string
			valid bool
		}{
			{name: "all scalar forms", input: `{"enabled":true,"title":"sample","payload":"aGk=","state":"ACTIVE","signed":"-9223372036854775808","unsigned":"18446744073709551615","ratio":1.5,"count":2,"observedAt":"2026-11-12T00:00:00Z","signedZigzag":-1,"signedFixed":"-2","unsignedFixed":"3","singleRatio":0.25,"unsignedCount":1,"zigzagCount":-1,"fixedCount":-1,"fixedUnsignedCount":1}`, valid: true},
			{name: "integer enum and numeric long values", input: `{"state":7,"signed":-1,"unsigned":2}`, valid: true},
			{name: "recursive children", input: `{"child":{"child":{"title":"tail"}},"children":[{}, {"enabled":false}]}`, valid: true},
			{name: "map values", input: `{"labels":{"accepted":true,"rejected":false}}`, valid: true},
			{name: "unknown enum", input: `{"state":"MISSING"}`},
			{name: "negative unsigned string", input: `{"unsigned":"-1"}`},
			{name: "negative unsigned number", input: `{"unsigned":-1}`},
			{name: "invalid long string", input: `{"signed":"one"}`},
			{name: "fractional count", input: `{"count":1.5}`},
			{name: "nonboolean map value", input: `{"labels":{"accepted":"yes"}}`},
			{name: "list must be array", input: `{"children":{}}`},
			{name: "nested unknown field", input: `{"child":{"secret":"omitted"}}`},
			{name: "byte payload must be text", input: `{"payload":42}`},
			{name: "timestamp must use protobuf JSON text", input: `{"observedAt":{"seconds":0}}`},
		} {
			t.Run(tc.name, func(t *testing.T) {
				t.Parallel()
				var input any
				require.NoError(t, json.Unmarshal([]byte(tc.input), &input), "sample input must decode")
				if tc.valid {
					assert.NoError(t, resolved.Validate(input), "valid protobuf JSON shape should pass the schema")
				} else {
					assert.Error(t, resolved.Validate(input), "invalid protobuf JSON shape should fail the schema")
				}
			})
		}
		assert.Len(t, definitions, 1, "recursive message should create one reusable definition")
		assert.Equal(t, map[string]any{"$ref": "#/$defs/mcp_schema_test.Node"}, messageSchema(file.Messages().ByName("Node"), definitions), "subsequent references should reuse the same definition")
	})
	t.Run("timestamp", func(t *testing.T) {
		t.Parallel()
		definitions := make(map[string]map[string]any)
		assert.Equal(t, map[string]any{schemaType: schemaString, "format": "date-time"}, messageSchema((&timestamppb.Timestamp{}).ProtoReflect().Descriptor(), definitions), "timestamp should use its protobuf JSON representation")
		assert.Empty(t, definitions, "well-known timestamp should not create an object definition")
	})
}
