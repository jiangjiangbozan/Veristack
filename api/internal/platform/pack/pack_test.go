package pack

import (
	"crypto/ed25519"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

type testPack struct {
	meta       Pack
	privateKey ed25519.PrivateKey
}

// newTestPack 构造一份带签名的测试 Pack。
func newTestPack(t *testing.T) testPack {
	t.Helper()
	_, privateKey, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatalf("生成测试密钥失败: %v", err)
	}
	meta := Pack{
		SchemaVersion: SchemaVersion,
		GeneratedAt:   "2026-11-25T10:00:00+08:00",
		Delivery: DeliveryInfo{
			BaselineVersion: 2,
			ID:              "delivery-1",
			ProjectName:     "官网改版",
			Version:         1,
		},
		Manifest: ManifestRef{Path: ManifestFileName, SHA256: "abc123"},
		Files: []FileEntry{
			{EvidenceID: "e1", Path: "files/aaa.png", SHA256: "aaa", Size: 10, Title: "截图一"},
			{EvidenceID: "e2", Path: "files/bbb.png", SHA256: "bbb", Size: 20, Title: "截图二"},
		},
		Issuer: Issuer{KeyID: "abcdef0123456789", OrganizationID: "org-1"},
	}
	signature, err := Sign(meta, privateKey)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	meta.Signature = signature
	return testPack{meta: meta, privateKey: privateKey}
}

// clone 复制一份 Pack，避免用例之间互相污染。
func (p testPack) clone() Pack {
	copied := p.meta
	copied.Files = append([]FileEntry{}, p.meta.Files...)
	return copied
}

func TestSignatureCoversWholePack(t *testing.T) {
	source := newTestPack(t)
	cases := []struct {
		name    string
		mutate  func(pack *Pack)
		wantErr bool
	}{
		{
			name:    "正常值：未篡改的 Pack 验签通过",
			mutate:  func(pack *Pack) {},
			wantErr: false,
		},
		{
			name:    "非法输入：删除附件条目后验签失败",
			mutate:  func(pack *Pack) { pack.Files = pack.Files[:1] },
			wantErr: true,
		},
		{
			name:    "非法输入：替换交付信息后验签失败",
			mutate:  func(pack *Pack) { pack.Delivery.ProjectName = "另一个项目" },
			wantErr: true,
		},
		{
			name:    "非法输入：替换清单摘要后验签失败",
			mutate:  func(pack *Pack) { pack.Manifest.SHA256 = "tampered" },
			wantErr: true,
		},
		{
			name:    "异常场景：篡改签发主体后验签失败",
			mutate:  func(pack *Pack) { pack.Issuer.OrganizationID = "other-org" },
			wantErr: true,
		},
		{
			name:    "异常场景：篡改格式版本后验签失败",
			mutate:  func(pack *Pack) { pack.SchemaVersion = "3" },
			wantErr: true,
		},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			mutated := source.clone()
			item.mutate(&mutated)
			result := verifySignature(mutated, source.meta.Manifest.SHA256)
			if item.wantErr && result.Passed {
				t.Fatalf("篡改后签名校验应当失败")
			}
			if !item.wantErr && !result.Passed {
				t.Fatalf("未篡改时签名校验应当通过: %s", result.Detail)
			}
		})
	}
}

func TestVerifyRejectsLegacySchemaVersion(t *testing.T) {
	source := newTestPack(t)
	mutated := source.clone()
	mutated.SchemaVersion = LegacySchemaVersion
	result := verifySignature(mutated, mutated.Manifest.SHA256)
	if result.Passed {
		t.Fatalf("v1 格式的签名覆盖不完整，必须判定为不通过")
	}
}

func TestCanonicalPayloadIsStable(t *testing.T) {
	source := newTestPack(t)
	first, err := CanonicalPayload(source.meta)
	if err != nil {
		t.Fatalf("构造载荷失败: %v", err)
	}
	second, err := CanonicalPayload(source.meta)
	if err != nil {
		t.Fatalf("构造载荷失败: %v", err)
	}
	if string(first) != string(second) {
		t.Fatalf("同一份 Pack 必须产生相同的规范载荷")
	}
	var decoded map[string]any
	if err := json.Unmarshal(first, &decoded); err != nil {
		t.Fatalf("载荷不是合法 JSON: %v", err)
	}
	for _, key := range []string{"delivery", "files", "generatedAt", "issuer", "manifest", "schemaVersion"} {
		if _, ok := decoded[key]; !ok {
			t.Fatalf("签名载荷缺少字段 %q", key)
		}
	}
	if _, ok := decoded["signature"]; ok {
		t.Fatalf("签名载荷不得包含签名本身")
	}
}

func TestVerifyDetectsRemovedAttachmentEntry(t *testing.T) {
	source := newTestPack(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, FilesDirName), 0o750); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	firstContent := []byte("first")
	secondContent := []byte("second")
	if err := os.WriteFile(filepath.Join(root, FilesDirName, "aaa.png"), firstContent, 0o600); err != nil {
		t.Fatalf("写入附件失败: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, FilesDirName, "bbb.png"), secondContent, 0o600); err != nil {
		t.Fatalf("写入附件失败: %v", err)
	}
	meta := source.clone()
	manifestContent := []byte("manifest")
	if err := os.WriteFile(filepath.Join(root, ManifestFileName), manifestContent, 0o600); err != nil {
		t.Fatalf("写入清单失败: %v", err)
	}
	meta.Manifest.SHA256 = sha256Hex(manifestContent)
	meta.Files[0].SHA256 = sha256Hex(firstContent)
	meta.Files[0].Size = int64(len(firstContent))
	meta.Files[1].SHA256 = sha256Hex(secondContent)
	meta.Files[1].Size = int64(len(secondContent))
	signature, err := Sign(meta, source.privateKey)
	if err != nil {
		t.Fatalf("签名失败: %v", err)
	}
	meta.Signature = signature
	if err := writePackMeta(t, root, meta); err != nil {
		t.Fatalf("写入 Pack 元数据失败: %v", err)
	}
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("核验失败: %v", err)
	}
	if report.Summary != "全部核验通过，交付内容与签名均有效" {
		t.Fatalf("完整 Pack 应当核验通过，实际结论：%s", report.Summary)
	}
	tampered := meta
	tampered.Files = tampered.Files[:1]
	if err := writePackMeta(t, root, tampered); err != nil {
		t.Fatalf("写入篡改元数据失败: %v", err)
	}
	report, err = Verify(root)
	if err != nil {
		t.Fatalf("核验失败: %v", err)
	}
	if report.Summary == "全部核验通过，交付内容与签名均有效" {
		t.Fatalf("删除附件条目后核验必须失败")
	}
}

func TestVerifyRejectsUnsafeAttachmentPath(t *testing.T) {
	source := newTestPack(t)
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, FilesDirName), 0o750); err != nil {
		t.Fatalf("创建目录失败: %v", err)
	}
	meta := source.clone()
	meta.Files = []FileEntry{{EvidenceID: "e1", Path: "../secret.txt", SHA256: "x", Size: 1, Title: "越界附件"}}
	if err := writePackMeta(t, root, meta); err != nil {
		t.Fatalf("写入 Pack 元数据失败: %v", err)
	}
	report, err := Verify(root)
	if err != nil {
		t.Fatalf("核验失败: %v", err)
	}
	if report.Summary == "全部核验通过，交付内容与签名均有效" {
		t.Fatalf("越界附件路径必须导致核验失败")
	}
}

// writePackMeta 把 Pack 元数据写入目录。
func writePackMeta(t *testing.T, root string, meta Pack) error {
	t.Helper()
	content, err := json.MarshalIndent(meta, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, PackFileName), content, 0o600)
}
