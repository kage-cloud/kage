package annos

import (
	"errors"
	"fmt"
	"reflect"
	"strings"
)

func ToMap(domain string, anno interface{}) map[string]string {
	m := map[string]string{}

	if anno == nil {
		return m
	}

	v := reflect.ValueOf(anno)
	fields := getFieldsByJsonName(v)

	toMapRecurse(domain, fields, m)

	return m
}

func FromMap(domain string, m map[string]string, a interface{}) error {
	if a == nil {
		return errors.New("the annotation struct must not be nil")
	}
	v := reflect.ValueOf(a)
	if !isStructPtr(v.Type()) {
		return errors.New("the annotation must be a ptr to a struct")
	}

	for k, ele := range m {
		if !strings.HasPrefix(k, domain) {
			continue
		}

		spl := strings.Split(k, "/")
		if len(spl) <= 1 {
			continue
		}

		if err := setFieldFromAnnoStr(spl[1:], ele, v); err != nil {
			return fmt.Errorf("failed to read %s: %s", k, err.Error())
		}
	}

	return nil
}
