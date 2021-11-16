package gopcua

import "github.com/gopcua/opcua/ua"

// nolint:gocyclo // long switch
func VariantToFloat64(value *ua.Variant) float64 {
	switch value.Type() {
	case ua.TypeIDBoolean:
		r, _ := value.Value().(bool)
		if r {
			return 1
		}
		return 0
	case ua.TypeIDSByte:
		return float64(value.Value().(int8))
	case ua.TypeIDByte:
		return float64(value.Value().(uint8))
	case ua.TypeIDInt16:
		return float64(value.Value().(int16))
	case ua.TypeIDUint16:
		return float64(value.Value().(uint16))
	case ua.TypeIDInt32:
		return float64(value.Value().(int32))
	case ua.TypeIDUint32:
		return float64(value.Value().(uint32))
	case ua.TypeIDInt64:
		return float64(value.Value().(int64))
	case ua.TypeIDUint64:
		return float64(value.Value().(uint64))
	case ua.TypeIDFloat:
		return float64(value.Value().(float32))
	case ua.TypeIDDouble:
		return value.Value().(float64)
	default:
		return 0
	}
}
