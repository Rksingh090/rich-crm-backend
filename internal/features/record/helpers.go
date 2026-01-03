package record

import (
	"encoding/json"
	"strconv"
)

// ParseInt64 parses a string or number into an int64
func ParseInt64(val interface{}, defaultVal int64) int64 {
	if val == nil {
		return defaultVal
	}
	switch v := val.(type) {
	case int64:
		return v
	case int:
		return int64(v)
	case float64:
		return int64(v)
	case string:
		if i, err := strconv.ParseInt(v, 10, 64); err == nil {
			return i
		}
	case json.Number:
		if i, err := v.Int64(); err == nil {
			return i
		}
	}
	return defaultVal
}
