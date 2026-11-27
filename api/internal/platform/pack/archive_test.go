// Package pack 的归档测试，验证下载包只包含根目录内的普通文件。
package pack

import (
	"archive/zip"
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestWriteArchiveIncludesRegularFiles(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, PackFileName), []byte("{}"), 0o600); err != nil {
		t.Fatalf("准备 pack.json 失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, FilesDirName), 0o750); err != nil {
		t.Fatalf("创建附件目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, FilesDirName, "a.txt"), []byte("evidence"), 0o600); err != nil {
		t.Fatalf("准备附件失败: %v", err)
	}

	var buffer bytes.Buffer
	if err := WriteArchive(root, &buffer); err != nil {
		t.Fatalf("归档应当成功: %v", err)
	}
	reader, err := zip.NewReader(bytes.NewReader(buffer.Bytes()), int64(buffer.Len()))
	if err != nil {
		t.Fatalf("读取 ZIP 失败: %v", err)
	}
	names := make(map[string]bool, len(reader.File))
	for _, file := range reader.File {
		names[file.Name] = true
	}
	for _, want := range []string{PackFileName, FilesDirName + "/a.txt"} {
		if !names[want] {
			t.Fatalf("归档缺少条目 %s，实际为 %v", want, names)
		}
	}
	if len(reader.File) != 2 {
		t.Fatalf("归档条目数量应为 2，实际为 %d", len(reader.File))
	}
}

func TestWriteArchiveRejectsSymlink(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, PackFileName)
	if err := os.WriteFile(target, []byte("{}"), 0o600); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}
	link := filepath.Join(root, "escape.txt")
	if err := os.Symlink(target, link); err != nil {
		t.Skipf("当前环境无法创建符号链接，跳过: %v", err)
	}

	var buffer bytes.Buffer
	if err := WriteArchive(root, &buffer); err == nil {
		t.Fatalf("含符号链接的 Pack 应当被拒绝")
	}
}

func TestWriteArchiveRejectsNonDirectory(t *testing.T) {
	root := filepath.Join(t.TempDir(), "pack.json")
	if err := os.WriteFile(root, []byte("{}"), 0o600); err != nil {
		t.Fatalf("准备文件失败: %v", err)
	}

	var buffer bytes.Buffer
	if err := WriteArchive(root, &buffer); err == nil {
		t.Fatalf("非目录路径应当被拒绝")
	}
}
