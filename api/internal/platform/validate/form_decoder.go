package validate

import (
	"fmt"
	"net/url"
	"reflect"
	"strings"
)

// formDecoder 把表单值按标签映射到结构体字段，避免为上传场景引入额外依赖。
type formDecoder struct {
	Target any
	Values url.Values
}

// decode 遍历目标结构字段并按标签赋值。
func (d *formDecoder) decode() error {
	target := reflect.ValueOf(d.Target)
	if target.Kind() != reflect.Pointer || target.IsNil() {
		return fmt.Errorf("表单解码目标必须是非空指针")
	}
	element := target.Elem()
	if element.Kind() != reflect.Struct {
		return fmt.Errorf("表单解码目标必须是结构体指针")
	}
	return d.decodeStruct(element)
}

// decodeStruct 逐字段处理结构体内容。
func (d *formDecoder) decodeStruct(element reflect.Value) error {
	structType := element.Type()
	for index := 0; index < structType.NumField(); index++ {
		field := structType.Field(index)
		value := element.Field(index)
		if !value.CanSet() {
			continue
		}
		switch value.Kind() {
		case reflect.Struct:
			if err := d.decodeStruct(value); err != nil {
				return err
			}
		case reflect.String:
			value.SetString(d.firstValue(field))
		case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
			parsed, err := parseInt(d.firstValue(field))
			if err != nil {
				return fmt.Errorf("字段 %s 不是合法整数: %w", field.Name, err)
			}
			value.SetInt(parsed)
		case reflect.Bool:
			value.SetBool(d.firstValue(field) == "true")
		}
	}
	return nil
}

// firstValue 按字段标签读取表单值，兼容 form 与 json 两种命名。
func (d *formDecoder) firstValue(field reflect.StructField) string {
	for _, tagName := range []string{"form", "json"} {
		tag := field.Tag.Get(tagName)
		if tag == "" {
			continue
		}
		name := strings.Split(tag, ",")[0]
		if name == "" {
			continue
		}
		if values, ok := d.Values[name]; ok && len(values) > 0 {
			return values[0]
		}
	}
	return ""
}
