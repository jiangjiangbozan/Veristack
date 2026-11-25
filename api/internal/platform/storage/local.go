// Package storage 的本地文件实现，适用于私有化部署的默认存储方式。
package storage

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

const (
	defaultFileMode = 0o600
	defaultDirMode  = 0o700
)

// LocalStore 把对象保存在本地目录，键即为相对根目录的文件路径。
type LocalStore struct {
	Root string // 存储根目录
}

// Delete 按对象键删除本地文件。
func (s *LocalStore) Delete(_ context.Context, key string) error {
	path, err := s.resolve(key)
	if err != nil {
		return err
	}
	if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("删除对象失败: %w", err)
	}
	return nil
}

// Open 打开本地文件供读取。
func (s *LocalStore) Open(_ context.Context, key string) (io.ReadCloser, error) {
	path, err := s.resolve(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("打开对象失败: %w", err)
	}
	return file, nil
}

// Put 把内容写入本地文件，自动创建父目录。
func (s *LocalStore) Put(_ context.Context, key string, content io.Reader) (string, error) {
	path, err := s.resolve(key)
	if err != nil {
		return "", err
	}
	if err := os.MkdirAll(filepath.Dir(path), defaultDirMode); err != nil {
		return "", fmt.Errorf("创建存储目录失败: %w", err)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, defaultFileMode)
	if err != nil {
		return "", fmt.Errorf("创建对象失败: %w", err)
	}
	defer file.Close()
	if _, err := io.Copy(file, content); err != nil {
		return "", fmt.Errorf("写入对象失败: %w", err)
	}
	return key, nil
}

// resolve 把对象键解析为绝对路径，并阻止跳出存储根目录的路径。
func (s *LocalStore) resolve(key string) (string, error) {
	path := filepath.Join(s.Root, key)
	absoluteRoot, err := filepath.Abs(s.Root)
	if err != nil {
		return "", fmt.Errorf("解析存储根目录失败: %w", err)
	}
	absolutePath, err := filepath.Abs(path)
	if err != nil {
		return "", fmt.Errorf("解析对象路径失败: %w", err)
	}
	relative, err := filepath.Rel(absoluteRoot, absolutePath)
	if err != nil || relative == ".." || strings.HasPrefix(relative, ".."+string(filepath.Separator)) {
		return "", fmt.Errorf("对象键非法: %s", key)
	}
	return absolutePath, nil
}
