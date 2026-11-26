package pack

import (
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// CheckResult 是单项核验的结果。
type CheckResult struct {
	Detail string `json:"detail"` // 核验详情
	Name   string `json:"name"`   // 核验项名称
	Passed bool   `json:"passed"` // 是否通过
}

// Report 是整份 Pack 的核验报告。
type Report struct {
	Checks  []CheckResult `json:"checks"`  // 逐项核验结果
	Pack    Pack          `json:"pack"`    // Pack 元数据
	Summary string        `json:"summary"` // 汇总结论
}

// Verify 独立核验 Pack 目录：校验格式版本、清单摘要、附件摘要与签名。
//
// 该函数按处理不可信输入的标准设计：路径受根目录约束，附件体积受限，
// 任何一项异常都会在报告中显式列出，并以非零退出码反馈给调用方。
func Verify(root string) (Report, error) {
	report := Report{Checks: make([]CheckResult, 0)}
	meta, err := readPackMeta(root)
	if err != nil {
		return report, err
	}
	report.Pack = meta
	report.Checks = append(report.Checks, verifySchemaVersion(meta))
	manifestCheck, actualSHA := verifyManifest(root, meta)
	report.Checks = append(report.Checks, manifestCheck)
	report.Checks = append(report.Checks, verifyFiles(root, meta)...)
	report.Checks = append(report.Checks, verifySignature(meta, actualSHA))
	report.Summary = summarize(report.Checks)
	return report, nil
}

// verifySchemaVersion 校验格式版本，未知版本一律拒绝，避免误判新格式。
func verifySchemaVersion(meta Pack) CheckResult {
	result := CheckResult{Name: "Pack 格式版本", Passed: false}
	switch meta.SchemaVersion {
	case SchemaVersion:
		result.Detail = "格式版本 " + meta.SchemaVersion
		result.Passed = true
	case LegacySchemaVersion:
		result.Detail = "该 Pack 使用 v" + LegacySchemaVersion + " 格式，签名未覆盖附件清单，请重新导出后再核验"
	default:
		result.Detail = "不支持的格式版本：" + meta.SchemaVersion
	}
	return result
}

// readPackMeta 读取 Pack 元数据文件。
func readPackMeta(root string) (Pack, error) {
	content, err := os.ReadFile(filepath.Join(root, PackFileName))
	if err != nil {
		return Pack{}, fmt.Errorf("读取 Pack 元数据失败: %w", err)
	}
	var meta Pack
	if err := json.Unmarshal(content, &meta); err != nil {
		return Pack{}, fmt.Errorf("解析 Pack 元数据失败: %w", err)
	}
	return meta, nil
}

// verifyManifest 校验交付清单快照的摘要是否与元数据一致，并返回实际摘要。
func verifyManifest(root string, meta Pack) (CheckResult, string) {
	result := CheckResult{Name: "交付清单摘要", Passed: false}
	content, err := os.ReadFile(filepath.Join(root, ManifestFileName))
	if err != nil {
		result.Detail = "读取交付清单失败：" + err.Error()
		return result, ""
	}
	actual := sha256Hex(content)
	if actual != meta.Manifest.SHA256 {
		result.Detail = fmt.Sprintf("摘要不一致，期望 %s，实际 %s", meta.Manifest.SHA256, actual)
		return result, actual
	}
	result.Detail = "清单内容与元数据一致"
	result.Passed = true
	return result, actual
}

// verifyFiles 逐个校验证据附件的路径安全性、摘要与大小。
func verifyFiles(root string, meta Pack) []CheckResult {
	results := make([]CheckResult, 0, len(meta.Files))
	seen := make(map[string]bool, len(meta.Files))
	var totalSize int64
	for _, file := range meta.Files {
		result := CheckResult{Name: "证据附件 " + file.Path, Passed: false}
		if seen[file.Path] {
			result.Detail = "附件路径重复，拒绝核验"
			results = append(results, result)
			continue
		}
		seen[file.Path] = true
		resolved, err := resolveInsideRoot(root, file.Path)
		if err != nil {
			result.Detail = "附件路径不安全：" + err.Error()
			results = append(results, result)
			continue
		}
		if file.Size > maxFileSize {
			result.Detail = fmt.Sprintf("附件超过大小上限 %d 字节", maxFileSize)
			results = append(results, result)
			continue
		}
		totalSize += file.Size
		if totalSize > maxPackSize {
			result.Detail = fmt.Sprintf("Pack 总体积超过上限 %d 字节", maxPackSize)
			results = append(results, result)
			break
		}
		actual, size, err := hashFile(resolved)
		if err != nil {
			result.Detail = "读取附件失败：" + err.Error()
			results = append(results, result)
			continue
		}
		if actual != file.SHA256 {
			result.Detail = fmt.Sprintf("摘要不一致，期望 %s，实际 %s", file.SHA256, actual)
			results = append(results, result)
			continue
		}
		if size != file.Size {
			result.Detail = fmt.Sprintf("大小不一致，期望 %d，实际 %d", file.Size, size)
			results = append(results, result)
			continue
		}
		result.Detail = "附件内容与元数据一致"
		result.Passed = true
		results = append(results, result)
	}
	return results
}

// verifySignature 校验规范载荷签名。
//
// 签名覆盖除签名本身外的全部元数据，因此删除附件条目、替换交付信息都会导致验签失败。
func verifySignature(meta Pack, actualSHA string) CheckResult {
	result := CheckResult{Name: "交付清单签名", Passed: false}
	if meta.Signature == nil {
		result.Detail = "该 Pack 未附带签名，无法验证来源"
		return result
	}
	if meta.Signature.Algorithm != SignatureAlgorithm {
		result.Detail = "签名算法不受支持：" + meta.Signature.Algorithm
		return result
	}
	if meta.SchemaVersion != SchemaVersion {
		result.Detail = "该 Pack 使用旧格式，签名未覆盖附件清单，无法证明内容完整"
		return result
	}
	publicKey, err := base64.StdEncoding.DecodeString(meta.Signature.PublicKey)
	if err != nil {
		result.Detail = "解析公钥失败：" + err.Error()
		return result
	}
	signature, err := base64.StdEncoding.DecodeString(meta.Signature.Value)
	if err != nil {
		result.Detail = "解析签名失败：" + err.Error()
		return result
	}
	if len(publicKey) != ed25519.PublicKeySize {
		result.Detail = "公钥长度不合法"
		return result
	}
	payload, err := CanonicalPayload(meta)
	if err != nil {
		result.Detail = "构造签名载荷失败：" + err.Error()
		return result
	}
	if !ed25519.Verify(ed25519.PublicKey(publicKey), payload, signature) {
		result.Detail = "签名与 Pack 内容不匹配，内容可能已被篡改"
		return result
	}
	if actualSHA != meta.Manifest.SHA256 {
		result.Detail = "签名有效，但当前文件内容与被签名内容不一致"
		return result
	}
	result.Detail = "签名有效，已覆盖交付信息、清单与全部附件"
	result.Passed = true
	return result
}

// summarize 依据逐项结果生成汇总结论。
func summarize(checks []CheckResult) string {
	failed := make([]string, 0)
	unsigned := false
	for _, check := range checks {
		if check.Passed {
			continue
		}
		if strings.Contains(check.Detail, "未附带签名") {
			unsigned = true
			continue
		}
		failed = append(failed, check.Name)
	}
	if len(failed) > 0 {
		return fmt.Sprintf("核验未通过，%d 项异常：%s", len(failed), strings.Join(failed, "、"))
	}
	if unsigned {
		return "内容核验通过，但该 Pack 未附带签名，无法验证来源"
	}
	return "全部核验通过，交付内容与签名均有效"
}
