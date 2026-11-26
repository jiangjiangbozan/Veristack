package config

import "testing"

func TestValidateSecretStrength(t *testing.T) {
	cases := []struct {
		name    string
		secret  string
		wantErr bool
	}{
		{name: "正常值：足够长度且随机的密钥通过", secret: "b3f1c9d2e8a7b6c5d4e3f2a1b0c9d8e7f6a5b4c3d2e1f0a9b8c7d6e5f4a3b2c1", wantErr: false},
		{name: "边界值：恰好 32 位通过", secret: "0123456789abcdef0123456789abcdef", wantErr: false},
		{name: "边界值：31 位被拒绝", secret: "0123456789abcdef0123456789abcde", wantErr: true},
		{name: "非法输入：示例默认值被拒绝", secret: "change-me-in-production", wantErr: true},
		{name: "非法输入：默认���大小写不敏感", secret: "Change-Me-In-Production", wantErr: true},
		{name: "非法输入：changeme 被拒绝", secret: "changeme", wantErr: true},
		{name: "异常场景：空字符串被拒绝", secret: "", wantErr: true},
		{name: "异常场景：前后空白视为同一密钥", secret: "  change-me-in-production  ", wantErr: true},
	}
	for _, item := range cases {
		t.Run(item.name, func(t *testing.T) {
			err := ValidateSecretStrength(item.secret)
			if item.wantErr && err == nil {
				t.Fatalf("期望校验失败，实际通过")
			}
			if !item.wantErr && err != nil {
				t.Fatalf("期望校验通过，实际失败: %v", err)
			}
		})
	}
}

func TestValidateRejectsUnsupportedStorageDriver(t *testing.T) {
	cfg := &Config{
		Auth:     AuthConfig{Secret: "0123456789abcdef0123456789abcdef"},
		Database: DatabaseConfig{DSN: "postgres://localhost/veristack"},
		Storage:  StorageConfig{Driver: storageDriverS3, LocalRoot: "./data"},
	}
	err := cfg.Validate()
	if err == nil {
		t.Fatalf("未实现的 s3 驱动必须被拒绝启动")
	}
}

func TestValidateAcceptsLocalStorage(t *testing.T) {
	cfg := &Config{
		Auth:     AuthConfig{Secret: "0123456789abcdef0123456789abcdef"},
		Database: DatabaseConfig{DSN: "postgres://localhost/veristack"},
		Storage:  StorageConfig{Driver: storageDriverLocal, LocalRoot: "./data"},
	}
	if err := cfg.Validate(); err != nil {
		t.Fatalf("本地存储配置应当通过校验: %v", err)
	}
}

func TestValidateRejectsWeakSecret(t *testing.T) {
	cfg := &Config{
		Auth:     AuthConfig{Secret: "change-me-in-production"},
		Database: DatabaseConfig{DSN: "postgres://localhost/veristack"},
		Storage:  StorageConfig{Driver: storageDriverLocal, LocalRoot: "./data"},
	}
	if err := cfg.Validate(); err == nil {
		t.Fatalf("弱密钥必须在启动阶段被拒绝")
	}
}
