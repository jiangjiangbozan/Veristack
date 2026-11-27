// Package pack 的归档模块，把已发布的 Evidence Pack 目录流式打包为 ZIP 供下载。
package pack

import (
	"archive/zip"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// WriteArchive 把 Pack 根目录下的全部文件写入 ZIP 流。
//
// 下载的是服务端已校验并发布的导出目录，但仍按不可信输入处理：
//   - 根路径必须是普通目录，符号链接一律拒绝
//   - 只允许普通文件，目录与符号链接不能混入归档
//   - 单个文件与整包体积沿用核验时的限制，避免压缩炸弹式占用
//   - 归档内路径统一使用斜杠，且不得逃出根目录
func WriteArchive(root string, target io.Writer) error {
	info, err := os.Lstat(root)
	if err != nil {
		return fmt.Errorf("读取 Pack 目录失败: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return fmt.Errorf("Pack 路径必须是普通目录: %s", root)
	}
	writer := zip.NewWriter(target)
	totalSize := int64(0)
	walkErr := filepath.Walk(root, func(path string, entry os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root || entry.IsDir() {
			return nil
		}
		if err := addArchiveEntry(writer, root, path, entry); err != nil {
			return err
		}
		totalSize += entry.Size()
		if totalSize > maxPackSize {
			return fmt.Errorf("Pack 总体积超过下载限制")
		}
		return nil
	})
	if walkErr != nil {
		_ = writer.Close()
		return walkErr
	}
	if err := writer.Close(); err != nil {
		return fmt.Errorf("生成下载包失败: %w", err)
	}
	return nil
}

// addArchiveEntry 校验单个文件并写入 ZIP 条目，防止符号链接或越界路径进入归档。
func addArchiveEntry(writer *zip.Writer, root, path string, entry os.FileInfo) error {
	if entry.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("Pack 中不允许符号链接: %s", path)
	}
	if !entry.Mode().IsRegular() {
		return fmt.Errorf("Pack 中只允许普通文件: %s", path)
	}
	if entry.Size() > maxFileSize {
		return fmt.Errorf("附件超过大小限制: %s", path)
	}
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return fmt.Errorf("解析 Pack 相对路径失败: %w", err)
	}
	name := filepath.ToSlash(relative)
	if name == "" || strings.HasPrefix(name, "..") {
		return fmt.Errorf("非法 Pack 相对路径: %s", name)
	}
	header, err := zip.FileInfoHeader(entry)
	if err != nil {
		return fmt.Errorf("构造 ZIP 头失败: %w", err)
	}
	header.Method = zip.Deflate
	header.Name = name
	part, err := writer.CreateHeader(header)
	if err != nil {
		return fmt.Errorf("写入 ZIP 条目失败: %w", err)
	}
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("读取附件失败: %w", err)
	}
	_, copyErr := io.Copy(part, file)
	closeErr := file.Close()
	if copyErr != nil {
		return fmt.Errorf("写入附件内容失败: %w", copyErr)
	}
	if closeErr != nil {
		return fmt.Errorf("关闭附件失败: %w", closeErr)
	}
	return nil
}
