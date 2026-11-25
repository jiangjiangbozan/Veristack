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

// Verify 独立核验 Pack 目录：校验清单摘要、附件摘要与签名。
func Verify(root string) (Report, error) {
	report := Report{Checks: make([]CheckResult, 0)}
	meta, err := readPackMeta(root)
	if err != nil {
		return report, err
	}
	report.Pack = meta
	manifestCheck, actualSHA := verifyManifest(root, meta)
	report.Checks = append(report.Checks, manifestCheck)
	report.Checks = append(report.Checks, verifyFiles(root, meta)...)
	report.Checks = append(report.Checks, verifySignature(meta, actualSHA))
	report.Summary = summarize(report.Checks)
	return report, nil
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
	sum := sha256.Sum256(content)
	actual := hex.EncodeToString(sum[:])
	if actual != meta.ManifestSHA {
		result.Detail = fmt.Sprintf("摘要不一致，期望 %s，实际 %s", meta.ManifestSHA, actual)
		return result, actual
	}
	result.Detail = "清单内容与元数据一致"
	result.Passed = true
	return result, actual
}

// verifyFiles 逐个校验证据附件的摘要。
func verifyFiles(root string, meta Pack) []CheckResult {
	results := make([]CheckResult, 0, len(meta.Files))
	for _, file := range meta.Files {
		result := CheckResult{Name: "证据附件 " + file.Path, Passed: false}
		content, err := os.ReadFile(filepath.Join(root, filepath.FromSlash(file.Path)))
		if err != nil {
			result.Detail = "读取附件失败：" + err.Error()
			results = append(results, result)
			continue
		}
		sum := sha256.Sum256(content)
		actual := hex.EncodeToString(sum[:])
		if actual != file.SHA256 {
			result.Detail = fmt.Sprintf("摘要不一致，期望 %s，实际 %s", file.SHA256, actual)
			results = append(results, result)
			continue
		}
		if int64(len(content)) != file.Size {
			result.Detail = fmt.Sprintf("大小不一致，期望 %d，实际 %d", file.Size, len(content))
			results = append(results, result)
			continue
		}
		result.Detail = "附件内容与元数据一致"
		result.Passed = true
		results = append(results, result)
	}
	return results
}

// verifySignature 校验清单签名，缺少签名时提示未签名而非判定失败。
// 签名必须同时覆盖元数据声明的摘要与实际文件摘要，才能证明签的就是看到的内容。
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
	if !ed25519.Verify(ed25519.PublicKey(publicKey), []byte(meta.ManifestSHA), signature) {
		result.Detail = "签名与清单摘要不匹配，内容可能已被篡改"
		return result
	}
	if actualSHA != meta.ManifestSHA {
		result.Detail = "签名有效，但当前文件内容与被签名内容不一致"
		return result
	}
	result.Detail = "签名有效"
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
