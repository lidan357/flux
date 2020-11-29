package support

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/bytepowered/flux"
	"github.com/bytepowered/flux/ext"
	"github.com/spf13/cast"
	"io"
	"io/ioutil"
	"net/url"
	"strings"
)

var (
	errCastToByteTypeNotSupported = errors.New("cannot convert value to []byte")
)

var (
	stringResolver = flux.TypedValueResolver(func(_ string, genericTypes []string, value flux.MIMEValue) (interface{}, error) {
		return CastDecodeToString(value)
	})
	integerResolver = flux.TypedValueResolveWrapper(func(value interface{}) (interface{}, error) {
		return cast.ToInt(value), nil
	}).ResolveFunc
	longResolver = flux.TypedValueResolveWrapper(func(value interface{}) (interface{}, error) {
		return cast.ToInt64(value), nil
	}).ResolveFunc
	float32Resolver = flux.TypedValueResolveWrapper(func(value interface{}) (interface{}, error) {
		return cast.ToFloat32(value), nil
	}).ResolveFunc
	float64Resolver = flux.TypedValueResolveWrapper(func(value interface{}) (interface{}, error) {
		return cast.ToFloat64(value), nil
	}).ResolveFunc
	booleanResolver = flux.TypedValueResolveWrapper(func(value interface{}) (interface{}, error) {
		return cast.ToBool(value), nil
	}).ResolveFunc
	mapResolver = flux.TypedValueResolver(func(_ string, genericTypes []string, value flux.MIMEValue) (interface{}, error) {
		return CastDecodeToStringMap(value)
	})
	listResolver = flux.TypedValueResolver(func(_ string, genericTypes []string, value flux.MIMEValue) (interface{}, error) {
		return CastToArrayList(genericTypes, value)
	})
	defaultResolver = flux.TypedValueResolver(func(typeClass string, typeGeneric []string, value flux.MIMEValue) (interface{}, error) {
		return map[string]interface{}{
			"class":   typeClass,
			"generic": typeGeneric,
			"value":   value,
		}, nil
	})
)

func init() {
	ext.StoreTypedValueResolver("string", stringResolver)
	ext.StoreTypedValueResolver("String", stringResolver)
	ext.StoreTypedValueResolver(flux.JavaLangStringClassName, stringResolver)

	ext.StoreTypedValueResolver("int", integerResolver)
	ext.StoreTypedValueResolver("Integer", integerResolver)
	ext.StoreTypedValueResolver(flux.JavaLangIntegerClassName, integerResolver)

	ext.StoreTypedValueResolver("int64", longResolver)
	ext.StoreTypedValueResolver("long", longResolver)
	ext.StoreTypedValueResolver("Long", longResolver)
	ext.StoreTypedValueResolver(flux.JavaLangLongClassName, longResolver)

	ext.StoreTypedValueResolver("float", float32Resolver)
	ext.StoreTypedValueResolver("Float", float32Resolver)
	ext.StoreTypedValueResolver(flux.JavaLangFloatClassName, float32Resolver)

	ext.StoreTypedValueResolver("double", float64Resolver)
	ext.StoreTypedValueResolver("Double", float64Resolver)
	ext.StoreTypedValueResolver(flux.JavaLangDoubleClassName, float64Resolver)

	ext.StoreTypedValueResolver("bool", booleanResolver)
	ext.StoreTypedValueResolver("Boolean", booleanResolver)
	ext.StoreTypedValueResolver(flux.JavaLangBooleanClassName, booleanResolver)

	ext.StoreTypedValueResolver("map", mapResolver)
	ext.StoreTypedValueResolver("Map", mapResolver)
	ext.StoreTypedValueResolver(flux.JavaUtilMapClassName, mapResolver)

	ext.StoreTypedValueResolver("slice", listResolver)
	ext.StoreTypedValueResolver("List", listResolver)
	ext.StoreTypedValueResolver(flux.JavaUtilListClassName, listResolver)

	ext.StoreTypedValueResolver(ext.DefaultTypedValueResolverName, defaultResolver)
}

// CastDecodeToString 最大努力地将值转换成String类型。
// 如果类型无法安全地转换成String或者解析异常，返回错误。
func CastDecodeToString(mimeV flux.MIMEValue) (string, error) {
	switch mimeV.MIMEType {
	case flux.ValueMIMETypeGoText:
		return mimeV.Value.(string), nil
	case flux.ValueMIMETypeGoStringMap:
		decoder := ext.LoadSerializer(ext.TypeNameSerializerJson)
		if data, err := decoder.Marshal(mimeV.Value); nil != err {
			return "", err
		} else {
			return string(data), nil
		}
	default:
		if data, err := _toBytes0(mimeV.Value); nil != err {
			if errCastToByteTypeNotSupported == err {
				return cast.ToStringE(mimeV.Value)
			} else {
				return "", err
			}
		} else {
			return string(data), nil
		}
	}
}

// CastDecodeToStringMap 最大努力地将值转换成map[string]any类型。
// 如果类型无法安全地转换成map[string]any或者解析异常，返回错误。
func CastDecodeToStringMap(mimeV flux.MIMEValue) (map[string]interface{}, error) {
	switch mimeV.MIMEType {
	case flux.ValueMIMETypeGoStringMap:
		return cast.ToStringMap(mimeV.Value), nil
	case flux.ValueMIMETypeGoText:
		decoder := ext.LoadSerializer(ext.TypeNameSerializerJson)
		var hashmap = map[string]interface{}{}
		if err := decoder.Unmarshal([]byte(mimeV.Value.(string)), &hashmap); nil != err {
			return nil, fmt.Errorf("cannot decode text to hashmap, text: %s, error:%w", mimeV.Value, err)
		} else {
			return hashmap, nil
		}
	case flux.ValueMIMETypeGoObject:
		if sm, err := cast.ToStringMapE(mimeV.Value); nil != err {
			return nil, fmt.Errorf("cannot cast object to hashmap, object: %+v, object.type:%T", mimeV.Value, mimeV.Value)
		} else {
			return sm, nil
		}
	default:
		var data []byte
		if strings.Contains(mimeV.MIMEType, "application/json") {
			if bs, err := _toBytes(mimeV.Value); nil != err {
				return nil, err
			} else {
				data = bs
			}
		} else if strings.Contains(mimeV.MIMEType, "application/x-www-form-urlencoded") {
			if bs, err := _toBytes(mimeV.Value); nil != err {
				return nil, err
			} else if jbs, err := JSONBytesFromQueryString(bs); nil != err {
				return nil, err
			} else {
				data = jbs
			}
		} else {
			if sm, err := cast.ToStringMapE(mimeV.Value); nil == err {
				return sm, nil
			} else {
				return nil, fmt.Errorf("unsupported mime-type to hashmap, value: %+v, value.type:%T, mime-type: %s",
					mimeV.Value, mimeV.Value, mimeV.MIMEType)
			}
		}
		decoder := ext.LoadSerializer(ext.TypeNameSerializerJson)
		var hashmap = map[string]interface{}{}
		err := decoder.Unmarshal(data, &hashmap)
		return hashmap, err
	}
}

// CastToArrayList 最大努力地将值转换成[]any类型。
// 如果类型无法安全地转换成[]any或者解析异常，返回错误。
func CastToArrayList(genericTypes []string, mimeV flux.MIMEValue) ([]interface{}, error) {
	// SingleValue to arraylist
	if len(genericTypes) > 0 {
		typeClass := genericTypes[0]
		resolver := ext.LoadTypedValueResolver(typeClass)
		if v, err := resolver(typeClass, []string{}, mimeV); nil != err {
			return nil, err
		} else {
			return []interface{}{v}, nil
		}
	} else {
		return []interface{}{mimeV.Value}, nil
	}
}

func _toBytes(v interface{}) ([]byte, error) {
	if bs, err := _toBytes0(v); nil != err {
		return nil, fmt.Errorf("value: %+v, value.type:%T, error: %w", v, v, err)
	} else {
		return bs, nil
	}
}

func _toBytes0(v interface{}) ([]byte, error) {
	switch v.(type) {
	case []byte:
		return v.([]byte), nil
	case string:
		return []byte(v.(string)), nil
	case io.Reader:
		data, err := ioutil.ReadAll(v.(io.Reader))
		if closer, ok := v.(io.Closer); ok {
			_ = closer.Close()
		}
		if nil != err {
			return nil, err
		} else {
			return data, nil
		}
	default:
		return nil, errCastToByteTypeNotSupported
	}
}

// Tested
func JSONBytesFromQueryString(queryStr []byte) ([]byte, error) {
	queryValues, err := url.ParseQuery(string(queryStr))
	if nil != err {
		return nil, err
	}
	fields := make([]string, 0, len(queryValues))
	for key, values := range queryValues {
		if len(values) > 1 {
			// quote with ""
			copied := make([]string, len(values))
			for i, val := range values {
				copied[i] = "\"" + string(JSONStringValueEncode(&val)) + "\""
			}
			fields = append(fields, "\""+key+"\":["+strings.Join(copied, ",")+"]")
		} else {
			fields = append(fields, "\""+key+"\":\""+string(JSONStringValueEncode(&values[0]))+"\"")
		}
	}
	bf := new(bytes.Buffer)
	bf.WriteByte('{')
	bf.WriteString(strings.Join(fields, ","))
	bf.WriteByte('}')
	return bf.Bytes(), nil
}

func JSONStringValueEncode(str *string) []byte {
	return []byte(strings.Replace(*str, `"`, `\"`, -1))
}
