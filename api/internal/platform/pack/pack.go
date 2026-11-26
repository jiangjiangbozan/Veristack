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
	//
	// v2 起签名覆盖除签名本身外的全部元数据，v1 仅覆盖交付清单摘要。
	SchemaVersion = "2"
	// ManifestFileName 是交付清单快照在 Pack 中的文件名。
	ManifestFileName = "manifest.json"
	// PackFileName 是 Pack 元数据在 Pack 中的文件名。
	PackFileName = "pack.json"
	// FilesDirName 是证据附件在 Pack 中的目录名。
	FilesDirName = "files"
	// SignatureAlgorithm 是清单签名使用的算法名称。
	SignatureAlgorithm = "ed25519"
	// LegacySchemaVersion 是仅覆盖交付清单摘要的旧格式版本。
	LegacySchemaVersion = "1"
)

const (
	// maxFileCount 限制单份 Pack 的附件数量，避免元数据膨胀。
	maxFileCount = 5000
	// maxFileSize 限制单个附件大小，防止核验时占用过多资源。
	maxFileSize = 512 << 20
	// maxPackSize 限制整份 Pack 的总体积。
	maxPackSize = 2 << 30
	// hashBufferSize 是流式计算摘要时的读取缓冲大小。
	hashBufferSize = 64 << 10
)

// FileEntry 描述 Pack 内的一个证据附件及其校验信息。
type FileEntry struct {
	EvidenceID string `json:"evidenceId"` // 证据 ID
	Path       string `json:"path"`       // Pack 内相对路径
	SHA256     string `json:"sha256"`     // 文件内容摘要
	Size       int64  `json:"size"`       // 文件字节数
	Title      string `json:"title"`      // 证据标题
}

// Issuer 描述签发 Pack 的主体，使签名来源可被独立识别。
type Issuer struct {
	KeyID          string `json:"keyId"`          // 签名公钥指纹
	OrganizationID string `json:"organizationId"` // 签发组织 ID
}

// ManifestRef 描述交付清单快照在 Pack 中的位置与摘要。
type ManifestRef struct {
	Path   string `json:"path"`   // 清单相对路径
	SHA256 string `json:"sha256"` // 清单内容摘要
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
	SchemaVersion string       `json:"schemaVersion"` // 格式版本
	GeneratedAt   string       `json:"generatedAt"`   // 生成时间
	Delivery      DeliveryInfo `json:"delivery"`      // 交付信息
	Manifest      ManifestRef  `json:"manifest"`      // 交付清单引用
	Files         []FileEntry  `json:"files"`         // 证据附件清单
	Issuer        Issuer       `json:"issuer"`        // 签发主体
	Signature     *Signature   `json:"signature"`     // 签名信息，未签名时为空
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
	if len(b.files) >= maxFileCount {
		return FileEntry{}, fmt.Errorf("证据附件数量超过上限 %d", maxFileCount)
	}
	info, err := os.Stat(sourcePath)
	if err != nil {
		return FileEntry{}, fmt.Errorf("读取证据附件失败: %w", err)
	}
	if info.Size() > maxFileSize {
		return FileEntry{}, fmt.Errorf("证据附件超过大小上限 %d 字节", maxFileSize)
	}
	sum, size, err := hashFile(sourcePath)
	if err != nil {
		return FileEntry{}, err
	}
	name := sum + filepath.Ext(sourcePath)
	relative := filepath.ToSlash(filepath.Join(FilesDirName, name))
	target := filepath.Join(b.root, FilesDirName, name)
	if err := os.MkdirAll(filepath.Dir(target), 0o750); err != nil {
		return FileEntry{}, fmt.Errorf("创建 Pack 文件目录失败: %w", err)
	}
	if err := copyFile(sourcePath, target); err != nil {
		return FileEntry{}, err
	}
	entry := FileEntry{
		EvidenceID: evidenceID,
		Path:       relative,
		SHA256:     sum,
		Size:       size,
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

// Write 把 Pack 元数据写入磁盘，可选地对规范载荷签名。
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

// Sign 对规范签名载荷签名，使外部可在不信任服务端的情况下验证交付内容。
//
// 载荷覆盖除签名本身外的全部元数据，因此删除附件条目、替换交付信息都会导致验签失败。
func Sign(pack Pack, privateKey ed25519.PrivateKey) (*Signature, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("签名私钥长度不合法")
	}
	payload, err := CanonicalPayload(pack)
	if err != nil {
		return nil, err
	}
	signature := ed25519.Sign(privateKey, payload)
	publicKey := privateKey.Public().(ed25519.PublicKey)
	return &Signature{
		Algorithm: SignatureAlgorithm,
		PublicKey: base64.StdEncoding.EncodeToString(publicKey),
		Value:     base64.StdEncoding.EncodeToString(signature),
	}, nil
}

// CanonicalPayload 构造被签名的规范化字节串。
//
// 采用键名字典序的 JSON 序列化，保证同一份 Pack 在任何实现下都得到相同载荷。
func CanonicalPayload(pack Pack) ([]byte, error) {
	canonical := map[string]any{
		"delivery":      pack.Delivery,
		"files":         pack.Files,
		"generatedAt":   pack.GeneratedAt,
		"issuer":        pack.Issuer,
		"manifest":      pack.Manifest,
		"schemaVersion": pack.SchemaVersion,
	}
	content, err := json.Marshal(canonical)
	if err != nil {
		return nil, fmt.Errorf("构造签名载荷失败: %w", err)
	}
	return content, nil
}

// KeyID 计算公钥指纹，作为签名密钥的可读标识。
func KeyID(publicKey ed25519.PublicKey) string {
	sum := sha256.Sum256(publicKey)
	return hex.EncodeToString(sum[:8])
}

// hashFile 流式计算文件摘要与大小，避免把大文件整体读入内存。
func hashFile(path string) (string, int64, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", 0, fmt.Errorf("打开文件失败: %w", err)
	}
	defer file.Close()
	hasher := sha256.New()
	buffer := make([]byte, hashBufferSize)
	var total int64
	for {
		read, readErr := file.Read(buffer)
		if read > 0 {
			total += int64(read)
			if _, err := hasher.Write(buffer[:read]); err != nil {
				return "", 0, fmt.Errorf("计算摘要失败: %w", err)
			}
		}
		if readErr != nil {
			if readErr == os.ErrClosed || readErr.Error() == "EOF" {
				break
			}
			return "", 0, fmt.Errorf("读取文件失败: %w", readErr)
		}
	}
	return hex.EncodeToString(hasher.Sum(nil)), total, nil
}

// copyFile 以流式方式复制文件，避免大文件占用内存。
func copyFile(source, target string) error {
	input, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("打开源文件失败: %w", err)
	}
	defer input.Close()
	output, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, 0o640)
	if err != nil {
		return fmt.Errorf("创建目标文件失败: %w", err)
	}
	defer output.Close()
	buffer := make([]byte, hashBufferSize)
	for {
		read, readErr := input.Read(buffer)
		if read > 0 {
			if _, err := output.Write(buffer[:read]); err != nil {
				return fmt.Errorf("写入目标文件失败: %w", err)
			}
		}
		if readErr != nil {
			if readErr.Error() == "EOF" {
				break
			}
			return fmt.Errorf("读取源文件失败: %w", readErr)
		}
	}
	return nil
}
