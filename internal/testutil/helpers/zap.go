package helpers

import (
	"go.uber.org/zap/zapcore"
)

// GetZapFieldsAsMap converts zapcore.Field array to map for easy testing
func GetZapFieldsAsMap(fields []zapcore.Field) map[string]interface{} {
	result := make(map[string]interface{})
	for _, field := range fields {
		switch field.Type {
		case zapcore.StringType:
			result[field.Key] = field.String
		case zapcore.Int64Type:
			result[field.Key] = field.Integer
		case zapcore.DurationType:
			result[field.Key] = field.Integer
		case zapcore.ArrayMarshalerType:
			// stringArrayの場合
			if arr, ok := field.Interface.(zapcore.ArrayMarshaler); ok {
				result[field.Key] = arr
			}
		case zapcore.ObjectMarshalerType:
			if obj, ok := field.Interface.(zapcore.ObjectMarshaler); ok {
				result[field.Key] = obj
			}
		case zapcore.BoolType:
			result[field.Key] = field.Integer == 1
		case zapcore.ErrorType:
			if field.Interface != nil {
				result[field.Key] = field.Interface.(error)
			}
		default:
			// その他の型はInterface経由で取得
			if field.Interface != nil {
				result[field.Key] = field.Interface
			}
		}
	}
	return result
}
