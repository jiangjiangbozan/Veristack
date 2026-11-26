package pack

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveInsideRoot(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.txt")
	if err := os.WriteFile(outside, []byte("secret"), 0o600); err != nil {
		t.Fatalf("准备外部文件失败: %v", err)
	}
	relativeOutside, err := filepath.Rel(root, outside)
	if err != nil {
		t.Fatalf("计算相对路径失败: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(root, FilesDirName), 0o750); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, FilesDirName, "a.png"), []byte("data"), 0o600); err != nil {
		t.Fatalf("准备附件失败: %v", err)
	}
	cases := []struct {
		name    string
		path    string
		wantErr bool
	}{
		{name: "正常值：目录内的相对路径被接受", path: "files/a.png", wantErr: false},
		{name: "边界值：单层子目录被接受", path: "files/nested/a.png", wantErr: false},
		{name: "非法输入：空路径被拒绝", path: "", wantErr: true},
		{name: "非法输入：上级目录被拒绝", path: "../secret.txt", wantErr: true},
		{name: "非法输入：嵌套上级目录被拒绝", path: "files/../../secret.txt", wantErr: true},
		{name: "非法输入：绝对路径被拒绝", path: outside, wantErr: true},
		{name: "非法输入：以斜杠开头的路径被拒绝", path: "/etc/passwd", wantErr: true},
		{name: "异常场景：指向目录外的相对路径被拒绝", path: relativeOutside, wantErr: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			resolved, err := resolveInsideRoot(root, item.path)
			if item.wantErr {
				if err == nil {
					t.Fatalf("路径 %q 应当被拒绝，实际解析为 %s", item.path, resolved)
				}
				return
			}
			if err != nil {
				t.Fatalf("路径 %q 应当被接受: %v", item.path, err)
			}
			if !strings.HasPrefix(resolved, root) {
				t.Fatalf("解析结果 %s 超出根目录 %s", resolved, root)
			}
		})
	}
}

func TestResolveInsideRootRejectsSymlinkEscape(t *testing.T) {
	root := t.TempDir()
	outsideDir := t.TempDir()
	if err := os.WriteFile(filepath.Join(outsideDir, "secret.txt"), []byte("secret"), 0o600); err != nil {
		t.Fatalf("准备外部文件失败: %v", err)
	}
	linkPath := filepath.Join(root, "escape")
	if err := os.Symlink(outsideDir, linkPath); err != nil {
		t.Skipf("当前环境不支持创建符号链接: %v", err)
	}
	if _, err := resolveInsideRoot(root, "escape/secret.txt"); err == nil {
		t.Fatalf("指向根目录外的符号链接必须被拒绝")
	}
}

func TestIsWithin(t *testing.T) {
	root := filepath.Join(string(filepath.Separator), "pack")
	cases := []struct {
		name      string
		candidate string
		want      bool
	}{
		{name: "正常值：子目录属于根目录", candidate: filepath.Join(root, "files", "a.png"), want: true},
		{name: "边界值：根本身属于根目录", candidate: root, want: true},
		{name: "非法输入：上级目录不属于根目录", candidate: filepath.Dir(root), want: false},
		{name: "非法输入：同名兄弟目录不属于根目录", candidate: filepath.Join(filepath.Dir(root), "pack-other"), want: false},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			if got := isWithin(root, item.candidate); got != item.want {
				t.Fatalf("isWithin(%q) = %v，期望 %v", item.candidate, got, item.want)
			}
		})
	}
}
