package annos

import (
	"encoding/json"
	"fmt"
	"github.com/fatih/structtag"
	"path"
	"reflect"
	"strconv"
	"strings"
	"unicode"
)

type fields struct {
	// list of field names to ignore based on tags
	Struct       reflect.Value
	Fields       map[string]reflect.Value
	NestedFields map[string]fields
}

func toMapRecurse(domain, keyPath string, fields fields, m map[string]string) {
	for k, v := range fields.Fields {
		m[path.Join(domain, keyPath, k)] = valAsString(v)
	}

	for k, v := range fields.NestedFields {
		toMapRecurse(domain, strings.Join([]string{keyPath, k}, "/"), v, m)
	}
}

func setFieldFromAnnoStr(splitKeyPath []string, annoVal string, field reflect.Value) error {
	var ok bool
	for _, v := range splitKeyPath {
		if isStructOrStructPtr(field.Type()) {
			field, ok = findField(field, v)
			if !ok {
				return nil
			}
		}
	}

	return setValFromString(annoVal, field)
}

func findField(v reflect.Value, fieldName string) (reflect.Value, bool) {
	v = stripAndInstantiatePtrs(v)
	for i := 0; i < v.NumField(); i++ {
		field := v.Type().Field(i)

		var name string
		if field.Anonymous {
			val, ok := findField(v.Field(i), fieldName)
			if ok {
				return val, ok
			}
		} else {
			name, _ = getFieldNameAndJsonTag(field)
		}

		if fieldName == name {
			return v.Field(i), true
		}
	}
	return reflect.Value{}, false
}

func stripAndInstantiatePtrs(v reflect.Value) reflect.Value {
	for v.Kind() == reflect.Ptr {
		if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}
		v = v.Elem()
	}
	return v
}

func setValFromString(s string, val reflect.Value) error {
	if !val.CanSet() || !val.IsValid() {
		return nil
	}

	switch val.Kind() {
	case reflect.Ptr:
		if val.IsNil() {
			val.Set(reflect.New(val.Type().Elem()))
		}
		return setValFromString(s, val.Elem())
	case reflect.Slice, reflect.Array:
		spli := strings.Split(s, ",")
		valSlice := reflect.MakeSlice(val.Type(), len(spli), len(spli))
		for i, ele := range spli {
			eleVal := reflect.New(val.Type().Elem()).Elem()
			if err := setValFromString(strings.TrimSpace(ele), eleVal); err != nil {
				return err
			}
			valSlice.Index(i).Set(eleVal)
		}
		val.Set(valSlice)
	case reflect.Map:
		spli := strings.Split(s, ",")
		refMap := reflect.MakeMap(val.Type())
		for _, ele := range spli {
			splitEle := strings.Split(strings.TrimSpace(ele), "=")
			if len(splitEle) < 2 {
				continue
			}
			keyVal := reflect.New(val.Type().Key()).Elem()
			eleVal := reflect.New(val.Type().Elem()).Elem()
			if err := setValFromString(splitEle[0], keyVal); err != nil {
				return err
			}
			if err := setValFromString(strings.Join(splitEle[1:], "="), eleVal); err != nil {
				return err
			}
			refMap.SetMapIndex(keyVal, eleVal)
		}
		val.Set(refMap)
	case reflect.Struct:
		n := reflect.New(val.Type())
		if err := json.Unmarshal([]byte(s), n.Interface()); err != nil {
			return err
		}
		val.Set(n.Elem())

	case reflect.Bool:
		trimmed := strings.TrimSpace(s)
		isTrue := trimmed == "true"
		if isTrue || trimmed == "false" {
			val.SetBool(isTrue)
		} else {
			return fmt.Errorf(`expected %s but got "%s"`, val.Type().String(), s)
		}
	case reflect.Float32, reflect.Float64:
		v, err := strconv.ParseFloat(s, 64)
		if err != nil {
			return fmt.Errorf(`expected %s but got "%s"`, val.Type().String(), s)
		}
		val.SetFloat(v)
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		v, err := strconv.ParseInt(s, 10, 64)
		if err != nil {
			return fmt.Errorf(`expected %s but got "%s"`, val.Type().String(), s)
		}
		val.SetInt(v)
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		v, err := strconv.ParseUint(s, 10, 64)
		if err != nil {
			return fmt.Errorf(`expected %s but got "%s"`, val.Type().String(), s)
		}
		val.SetUint(v)
	case reflect.String:
		val.SetString(s)
	default:
		return fmt.Errorf(`expected %s but got "%s"`, val.Type().String(), s)

	}

	return nil
}

func getFieldsByJsonName(v reflect.Value) fields {
	m := fields{
		Fields:       map[string]reflect.Value{},
		NestedFields: map[string]fields{},
	}

	if !v.IsValid() {
		return m
	}

	typ := v.Type()
	if typ.Kind() == reflect.Ptr {
		return getFieldsByJsonName(v.Elem())
	}

	if typ.Kind() != reflect.Struct {
		return m
	}

	m.Struct = v

	for i := 0; i < typ.NumField(); i++ {
		fieldType := typ.Field(i)
		fieldVal := v.Field(i)
		if !fieldVal.IsValid() {
			continue
		}

		name, jsonTag := getFieldNameAndJsonTag(fieldType)
		if name == "" {
			continue
		}

		if jsonTag != nil && jsonTag.HasOption("omitempty") && fieldVal.IsZero() {
			continue
		}

		if fieldType.Anonymous {
			if isStructOrStructPtr(fieldType.Type) {
				subFieldVal := getFieldsByJsonName(fieldVal)
				for k, v := range subFieldVal.Fields {
					m.Fields[k] = v
				}
			}
		} else if isStructOrStructPtr(fieldType.Type) {
			subFieldVal := getFieldsByJsonName(fieldVal)
			m.NestedFields[name] = subFieldVal
		} else {
			m.Fields[name] = fieldVal
		}
	}
	return m
}

func getFieldNameAndJsonTag(field reflect.StructField) (string, *structtag.Tag) {
	name := field.Name

	if unicode.IsLower(rune(name[0])) {
		return "", nil
	}

	tags, err := structtag.Parse(string(field.Tag))
	if err != nil {
		return name, nil
	}

	tag, err := tags.Get("json")
	if err != nil {
		return name, nil
	}

	if !tag.HasOption("-") {
		name = tag.Name
	}
	return name, tag
}

func isStructOrStructPtr(v reflect.Type) bool {
	return v.Kind() == reflect.Struct || isStructPtr(v)
}

func isStructPtr(v reflect.Type) bool {
	return v.Kind() == reflect.Ptr && v.Elem().Kind() == reflect.Struct
}

func valAsString(val reflect.Value) string {
	if !val.IsValid() || !val.CanInterface() {
		return ""
	}

	switch val.Kind() {
	case reflect.Ptr:
		return valAsString(val.Elem())
	case reflect.Slice, reflect.Array:
		s := make([]string, 0, val.Len())
		for i := 0; i < val.Len(); i++ {
			s = append(s, valAsString(val.Index(i)))
		}
		return strings.Join(s, ",")
	case reflect.Map:
		iter := val.MapRange()
		s := make([]string, 0, val.Len())
		for iter.Next() {
			k := iter.Key()
			v := iter.Value()
			s = append(s, fmt.Sprintf("%s=%s", valAsString(k), valAsString(v)))
			return strings.Join(s, ",")
		}
	case reflect.Struct:
		s, _ := json.Marshal(val.Interface())
		return string(s)
	}
	return fmt.Sprintf("%v", val.Interface())
}
