package validate

import "strconv"

// parseInt 解析表单中的整数值，空串返回零值。
func parseInt(raw string) (int64, error) {
	if raw == "" {
		return 0, nil
	}
	return strconv.ParseInt(raw, 10, 64)
}
