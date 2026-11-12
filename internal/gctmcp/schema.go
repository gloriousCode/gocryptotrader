package gctmcp

import "google.golang.org/protobuf/reflect/protoreflect"

const (
	schemaType    = "type"
	schemaString  = "string"
	schemaInteger = "integer"
	schemaObject  = "object"
)

// requestSchema derives command inputs from GCT's compiled protobuf contract.
func requestSchema(method protoreflect.MethodDescriptor) map[string]any {
	definitions := make(map[string]map[string]any)
	request := messageSchema(method.Input(), definitions)
	if fields := requiredMessageFields[string(method.Name())]; len(fields) > 0 {
		required := make([]string, 0, len(fields))
		for _, name := range fields {
			required = append(required, method.Input().Fields().ByName(name).JSONName())
		}
		definitions[string(method.Input().FullName())]["required"] = required
	}
	properties := map[string]any{"request": request}
	if method.IsStreamingServer() {
		properties["maxMessages"] = map[string]any{schemaType: schemaInteger, "minimum": 1, "maximum": 1000, "description": "Maximum sample count, further bounded by the operator's startup limit."}
	}
	return map[string]any{schemaType: schemaObject, "properties": properties, "required": []string{"request"}, "additionalProperties": false, "$defs": definitions}
}

// messageSchema uses references so recursive protobuf messages remain finite.
func messageSchema(message protoreflect.MessageDescriptor, definitions map[string]map[string]any) map[string]any {
	name := string(message.FullName())
	if name == "google.protobuf.Timestamp" {
		return map[string]any{schemaType: schemaString, "format": "date-time"}
	}
	ref := map[string]any{"$ref": "#/$defs/" + name}
	if _, exists := definitions[name]; exists {
		return ref
	}
	properties := make(map[string]any)
	definitions[name] = map[string]any{schemaType: schemaObject, "properties": properties, "additionalProperties": false}
	for i := range message.Fields().Len() {
		field := message.Fields().Get(i)
		valueField := field
		if field.IsMap() {
			valueField = field.MapValue()
		}
		var schema map[string]any
		switch valueField.Kind() {
		case protoreflect.BoolKind:
			schema = map[string]any{schemaType: "boolean"}
			if message.FullName() == "gctrpc.GetHistoricCandlesRequest" {
				switch field.Name() {
				case "use_db", "sync", "fill_missing_with_trades", "force":
					schema["const"] = false
				}
			}
		case protoreflect.StringKind:
			schema = map[string]any{schemaType: schemaString}
		case protoreflect.BytesKind:
			schema = map[string]any{schemaType: schemaString, "contentEncoding": "base64"}
		case protoreflect.MessageKind, protoreflect.GroupKind:
			schema = messageSchema(valueField.Message(), definitions)
		case protoreflect.EnumKind:
			values := make([]string, 0, valueField.Enum().Values().Len())
			for j := range valueField.Enum().Values().Len() {
				values = append(values, string(valueField.Enum().Values().Get(j).Name()))
			}
			schema = map[string]any{"anyOf": []any{map[string]any{schemaType: schemaString, "enum": values}, map[string]any{schemaType: schemaInteger}}}
		case protoreflect.Int64Kind, protoreflect.Sint64Kind, protoreflect.Sfixed64Kind:
			schema = map[string]any{"anyOf": []any{map[string]any{schemaType: schemaString, "pattern": "^-?[0-9]+$"}, map[string]any{schemaType: schemaInteger}}}
		case protoreflect.Uint64Kind, protoreflect.Fixed64Kind:
			schema = map[string]any{"anyOf": []any{map[string]any{schemaType: schemaString, "pattern": "^[0-9]+$"}, map[string]any{schemaType: schemaInteger, "minimum": 0}}}
		case protoreflect.FloatKind, protoreflect.DoubleKind:
			schema = map[string]any{schemaType: "number"}
		default:
			schema = map[string]any{schemaType: schemaInteger}
		}
		if field.IsMap() {
			schema = map[string]any{schemaType: schemaObject, "additionalProperties": schema}
		} else if field.IsList() {
			schema = map[string]any{schemaType: "array", "items": schema}
		}
		properties[field.JSONName()] = schema
	}
	return ref
}
