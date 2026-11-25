// Package validate 在请求边界统一解码并校验外部输入，让错误以 400 返回。
package validate

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"github.com/go-playground/validator/v10"
)

const maxBodyBytes = 1 << 20

var instance = validator.New(validator.WithRequiredStructEnabled())

// Decode 从请求体解码到目标结构并执行结构标签校验。
func Decode(r *http.Request, target any) error {
	decoder := json.NewDecoder(io.LimitReader(r.Body, maxBodyBytes))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return fmt.Errorf("请求体格式无效: %w", err)
	}
	if err := instance.Struct(target); err != nil {
		return fmt.Errorf("字段校验失败: %s", describeFields(err))
	}
	return nil
}

// DecodeForm 从表单字段解码到目标结构并执行结构标签校验，用于文件上传等 multipart 请求。
func DecodeForm(r *http.Request, target any) error {
	if err := r.ParseForm(); err != nil {
		return fmt.Errorf("表单格式无效: %w", err)
	}
	if err := decodeStruct(r.Form, target); err != nil {
		return err
	}
	if err := instance.Struct(target); err != nil {
		return fmt.Errorf("字段校验失败: %s", describeFields(err))
	}
	return nil
}

// decodeStruct 把表单键值映射到结构体字段，支持 form 标签与 json 标签两种命名。
func decodeStruct(values url.Values, target any) error {
	decoder := formDecoder{Target: target, Values: values}
	return decoder.decode()
}

// describeFields 把校验错误压缩为可读的字段名列表。
func describeFields(err error) string {
	validationErrors, ok := err.(validator.ValidationErrors)
	if !ok {
		return err.Error()
	}
	fields := make([]string, 0, len(validationErrors))
	for _, item := range validationErrors {
		fields = append(fields, fmt.Sprintf("%s(%s)", item.Field(), item.Tag()))
	}
	return strings.Join(fields, ", ")
}
