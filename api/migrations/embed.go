// Package migrations 将迁移脚本嵌入二进制，使服务启动时无需依赖外部文件。
package migrations

import "embed"

// Files 保存全部 Goose 迁移脚本。
//
//go:embed *.sql
var Files embed.FS
