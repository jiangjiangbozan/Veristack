// Package pack 定义 Evidence Pack 的文件格式与校验逻辑，使交付证据可被独立核验。
package pack

import (
	"crypto/ed25519"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const (
	// SchemaVersion 是当前 Evidence Pack 格式版本。
	SchemaVersion = "1"
	// ManifestFileName 是交付清单快照在 Pack 中的文件名。
	ManifestFileName = "manifest.json"
	// PackFileName 是 Pack 元数据在 Pack 中的文件名。
	PackFileName = "pack.json"
	// FilesDirName 是证据附件在 Pack 中的目录名。
	FilesDirName = "files"
	// SignatureAlgorithm 是清单签名使用的算法名称。
	SignatureAlgorithm = "ed25519"
)

// FileEntry 描述 Pack 内的一个证据附件及其校验信息。
type FileEntry struct {
	EvidenceID string `json:"evidenceId"` // 证据 ID
	Path       string `json:"path"`       // Pack 内相对路径
	SHA256     string `json:"sha256"`     // 文件内容摘要
	Size       int64  `json:"size"`       // 文件字节数
	Title      string `json:"title"`      // 证据标题
}

// Signature 描述交付清单的签名信息。
type Signature struct {
	Algorithm string `json:"algorithm"` // 签名算法
	PublicKey string `json:"publicKey"` // Base64 公钥
	Value     string `json:"value"`     // Base64 签名值
}

// DeliveryInfo 描述交付版本的基本信息。
type DeliveryInfo struct {
	BaselineVersion int32  `json:"baselineVersion"` // 来源基线版本
	BuildNo         string `json:"buildNo"`         // 构建号
	GitRef          string `json:"gitRef"`          // Git 提交或标签
	ID              string `json:"id"`              // 交付 ID
	ProjectName     string `json:"projectName"`     // 项目名称
	Version         int32  `json:"version"`         // 交付版本号
}

// Pack 是 Evidence Pack 的元数据，也是独立核验的入口文件。
type Pack struct {
	SchemaVersion string       `json:"schemaVersion"`  // 格式版本
	GeneratedAt   string       `json:"generatedAt"`    // 生成时间
	Delivery      DeliveryInfo `json:"delivery"`       // 交付信息
	ManifestSHA   string       `json:"manifestSha256"` // 交付清单摘要
	Files         []FileEntry  `json:"files"`          // 证据附件清单
	Signature     *Signature   `json:"signature"`      // 清单签名，未签名时为空
}

// Builder 负责组装 Evidence Pack 的文件内容。
type Builder struct {
	files []FileEntry
	root  string
}

// NewBuilder 创建 Pack 构建器，root 为 Pack 目录。
func NewBuilder(root string) *Builder {
	return &Builder{files: make([]FileEntry, 0), root: root}
}

// AddFile 把证据附件写入 Pack，并记录其摘要。
func (b *Builder) AddFile(evidenceID, title, sourcePath string) (FileEntry, error) {
	content, err := os.ReadFile(sourcePath)
	if err != nil {
		return FileEntry{}, fmt.Errorf("读取证据附件失败: %w", err)
	}
	sum := sha256.Sum256(content)
	name := hex.EncodeToString(sum[:]) + filepath.Ext(sourcePath)
	relative := filepath.ToSlash(filepath.Join(FilesDirName, name))
	target := filepath.Join(b.root, FilesDirName, name)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return FileEntry{}, fmt.Errorf("创建 Pack 文件目录失败: %w", err)
	}
	if err := os.WriteFile(target, content, 0o640); err != nil {
		return FileEntry{}, fmt.Errorf("写入 Pack 文件失败: %w", err)
	}
	entry := FileEntry{
		EvidenceID: evidenceID,
		Path:       relative,
		SHA256:     hex.EncodeToString(sum[:]),
		Size:       int64(len(content)),
		Title:      title,
	}
	b.files = append(b.files, entry)
	return entry, nil
}

// Files 返回已加入 Pack 的证据附件清单。
func (b *Builder) Files() []FileEntry {
	return b.files
}

// WriteManifest 写入交付清单快照并返回其摘要。
func (b *Builder) WriteManifest(manifest []byte) (string, error) {
	if err := os.MkdirAll(b.root, 0o750); err != nil {
		return "", fmt.Errorf("创建 Pack 目录失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(b.root, ManifestFileName), manifest, 0o640); err != nil {
		return "", fmt.Errorf("写入交付清单失败: %w", err)
	}
	sum := sha256.Sum256(manifest)
	return hex.EncodeToString(sum[:]), nil
}

// Write 把 Pack 元数据写入磁盘，可选地对清单摘要签名。
func (b *Builder) Write(pack Pack) error {
	if err := os.MkdirAll(b.root, 0o750); err != nil {
		return fmt.Errorf("创建 Pack 目录失败: %w", err)
	}
	content, err := json.MarshalIndent(pack, "", "  ")
	if err != nil {
		return fmt.Errorf("序列化 Pack 元数据失败: %w", err)
	}
	if err := os.WriteFile(filepath.Join(b.root, PackFileName), content, 0o640); err != nil {
		return fmt.Errorf("写入 Pack 元数据失败: %w", err)
	}
	return nil
}

// Sign 对清单摘要签名，使外部可在不信任服务端的情况下验证交付内容。
func Sign(manifestSHA string, privateKey ed25519.PrivateKey) (*Signature, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("签名私钥长度不合法")
	}
	signature := ed25519.Sign(privateKey, []byte(manifestSHA))
	return &Signature{
		Algorithm: SignatureAlgorithm,
		PublicKey: base64.StdEncoding.EncodeToString(privateKey.Public().(ed25519.PublicKey)),
		Value:     base64.StdEncoding.EncodeToString(signature),
	}, nil
}
