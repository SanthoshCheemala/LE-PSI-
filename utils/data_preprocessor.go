package utils

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
)

// SerializeData converts any data type to a deterministic string representation.
func SerializeData(data interface{}) (string, error) {
	if data == nil {
		return "", fmt.Errorf("cannot serialize nil data")
	}

	val := reflect.ValueOf(data)
	return serializeValue(val)
}

// serializeValue recursively handles different types
func serializeValue(val reflect.Value) (string, error) {
	if val.Kind() == reflect.Ptr {
		if val.IsNil() {
			return "null", nil
		}
		return serializeValue(val.Elem())
	}

	switch val.Kind() {
	case reflect.String:
		return val.String(), nil

	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return strconv.FormatInt(val.Int(), 10), nil

	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return strconv.FormatUint(val.Uint(), 10), nil

	case reflect.Float32, reflect.Float64:
		return strconv.FormatFloat(val.Float(), 'f', -1, 64), nil

	case reflect.Bool:
		return strconv.FormatBool(val.Bool()), nil

	case reflect.Slice, reflect.Array:
		return serializeSlice(val)

	case reflect.Map:
		return serializeMap(val)

	case reflect.Struct:
		return serializeStruct(val)

	default:
		jsonBytes, err := json.Marshal(val.Interface())
		if err != nil {
			return "", fmt.Errorf("unsupported type %v: %w", val.Kind(), err)
		}
		return string(jsonBytes), nil
	}
}

func serializeSlice(val reflect.Value) (string, error) {
	parts := make([]string, val.Len())
	for i := 0; i < val.Len(); i++ {
		serialized, err := serializeValue(val.Index(i))
		if err != nil {
			return "", err
		}
		parts[i] = serialized
	}
	return "[" + join(parts, ",") + "]", nil
}

func serializeMap(val reflect.Value) (string, error) {
	keys := val.MapKeys()
	keyStrings := make([]string, len(keys))
	
	for i, key := range keys {
		serialized, err := serializeValue(key)
		if err != nil {
			return "", err
		}
		keyStrings[i] = serialized
	}
	sort.Strings(keyStrings)

	pairs := make([]string, len(keyStrings))
	for i, keyStr := range keyStrings {
		for _, key := range keys {
			serializedKey, _ := serializeValue(key)
			if serializedKey == keyStr {
				mapVal := val.MapIndex(key)
				serializedVal, err := serializeValue(mapVal)
				if err != nil {
					return "", err
				}
				pairs[i] = keyStr + ":" + serializedVal
				break
			}
		}
	}
	return "{" + join(pairs, ",") + "}", nil
}

func serializeStruct(val reflect.Value) (string, error) {
	typ := val.Type()
	fields := make([]string, 0, val.NumField())
	
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := typ.Field(i)
		
		if !field.CanInterface() {
			continue
		}
		
		serialized, err := serializeValue(field)
		if err != nil {
			return "", err
		}
		fields = append(fields, fieldType.Name+":"+serialized)
	}
	
	sort.Strings(fields)
	return "{" + join(fields, ",") + "}", nil
}

func join(parts []string, sep string) string {
	if len(parts) == 0 {
		return ""
	}
	result := parts[0]
	for i := 1; i < len(parts); i++ {
		result += sep + parts[i]
	}
	return result
}

// PrepareDataForPSI converts any data type array to serialized strings.
// Returns serialized representations preserving original indices.
func PrepareDataForPSI(dataset []interface{}) ([]string, error) {
	if len(dataset) == 0 {
		return nil, fmt.Errorf("dataset is empty")
	}

	hashedData := make([]string, len(dataset))
	
	for i, data := range dataset {
		serialized, err := SerializeData(data)
		if err != nil {
			return nil, fmt.Errorf("error serializing item %d: %w", i, err)
		}
		hashedData[i] = serialized
	}

	return hashedData, nil
}

// HashDataPoints converts serialized strings to uint64 hashes using SHA-256.
func HashDataPoints(serializedData []string) []uint64 {
	hashes := make([]uint64, len(serializedData))
	
	for i, data := range serializedData {
		hash := sha256.Sum256([]byte(data))
		hashes[i] = binary.BigEndian.Uint64(hash[:8])
	}
	
	return hashes
}
