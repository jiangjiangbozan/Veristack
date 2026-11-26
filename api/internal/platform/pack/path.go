// Package pack 的路径安全模块，把 Pack 内的相对路径限制在根目录之内。
package pack

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// resolveInsideRoot 把 Pack 内的相对路径解析为根目录内的绝对路径。
//
// 校验分四步，缺一不可：
//  1. 路径必须非空，且不是绝对路径
//  2. 路径中不得包含 `..` 路径段
//  3. 拼接后的绝对路径必须仍在根目录之内
//  4. 符号链接解析后仍必须位于根目录之内
//
// 第 4 步防止攻击者放置指向目录外的符号链接来绕过第 3 步。
func resolveInsideRoot(root, relativePath string) (string, error) {
	if strings.TrimSpace(relativePath) == "" {
		return "", fmt.Errorf("对象路径不能为空")
	}
	normalized := filepath.ToSlash(relativePath)
	if strings.HasPrefix(normalized, "/") || filepath.IsAbs(relativePath) {
		return "", fmt.Errorf("对象路径必须是相对路径: %s", relativePath)
	}
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", fmt.Errorf("对象路径不得包含上级目录: %s", relativePath)
		}
	}
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return "", fmt.Errorf("解析 Pack 根目录失败: %w", err)
	}
	candidate := filepath.Join(absoluteRoot, filepath.FromSlash(normalized))
	if !isWithin(absoluteRoot, candidate) {
		return "", fmt.Errorf("对象路径超出 Pack 根目录: %s", relativePath)
	}
	resolved, err := filepath.EvalSymlinks(candidate)
	if err != nil {
		if os.IsNotExist(err) {
			return candidate, nil
		}
		return "", fmt.Errorf("解析对象路径失败: %w", err)
	}
	resolvedRoot, err := filepath.EvalSymlinks(absoluteRoot)
	if err != nil {
		resolvedRoot = absoluteRoot
	}
	if !isWithin(resolvedRoot, resolved) {
		return "", fmt.Errorf("对象路径经符号链接后超出 Pack 根目录: %s", relativePath)
	}
	return candidate, nil
}

// isWithin 判断候选路径是否位于根目录之内。
func isWithin(root, candidate string) bool {
	relative, err := filepath.Rel(root, candidate)
	if err != nil {
		return false
	}
	if relative == "." {
		return true
	}
	return relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}
