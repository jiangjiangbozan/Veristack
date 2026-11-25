// Package storage 通过 ObjectStore 接口抽象文件存储，使业务层不感知底层实现。
package storage

import (
	"context"
	"io"
)

// ObjectStore 定义证据文件的读写契约，一期使用本地文件系统，二期可接入 S3。
type ObjectStore interface {
	// Delete 按对象键删除文件。
	Delete(ctx context.Context, key string) error
	// Open 返回对象内容的只读流，调用方负责关闭。
	Open(ctx context.Context, key string) (io.ReadCloser, error)
	// Put 写入对象内容并返回对象键。
	Put(ctx context.Context, key string, content io.Reader) (string, error)
}
