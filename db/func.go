package db

import (
	"github.com/lib/pq"
	"reflect"
	"strconv"
	"strings"
)

// CastInterfaceString
//Функция возвращает исходя из типа, строку для секции VALUES()
func CastInterfaceString(i interface{}) (s string) {
	// Исходя и типа возвращаем строку
	if reflect.ValueOf(i).Kind() == reflect.Ptr {
		t := reflect.ValueOf(i)
		if t.IsNil() {
			return "NULL"
		} else {
			switch reflect.TypeOf(i).Elem().Kind() {
			case reflect.Uint64:
				return strconv.FormatUint(reflect.ValueOf(i).Elem().Uint(), 10)

			case reflect.Int:
				return strconv.Itoa(int(reflect.ValueOf(i).Elem().Int()))

			case reflect.Float64:
				var floatType = reflect.TypeOf(float64(0))
				v := reflect.ValueOf(i)
				v = reflect.Indirect(v)
				if !v.Type().ConvertibleTo(floatType) {
					return "NULL"
				}
				fv := v.Convert(floatType)
				return strconv.FormatFloat(fv.Float(), 'f', 2, 64)

			case reflect.String:
				return pq.QuoteLiteral(reflect.ValueOf(i).Elem().String())

			case reflect.Bool:
				if reflect.ValueOf(i).Elem().Bool() {
					return "'t'"
				} else {
					return "'f'"
				}

			}
		}
	}

	switch i.(type) {
	case nil:
		return "NULL"
	case string:
		if strings.Compare(i.(string), "") == 0 {
			return "NULL"
		}
		return pq.QuoteLiteral(i.(string))

	case int:
		return strconv.Itoa(i.(int))

	case uint64:
		return strconv.FormatUint(i.(uint64), 10)

	case float64:
		return strconv.FormatFloat(i.(float64), 'f', 2, 64)

	case bool:
		if i.(bool) {
			return "'t'"
		} else {
			return "'f'"
		}

	default:
		return ""
	}
}
