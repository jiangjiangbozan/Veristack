// Package main 是 Veristack 的命令行工具，用于独立核验 Evidence Pack 与生成签名密钥。
package main

import (
	"crypto/ed25519"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"api/internal/platform/pack"
)

const usage = `Veristack 验迹 · 交付证据核验工具

用法:
  veristack verify <pack目录>     核验 Evidence Pack 的清单摘要、附件摘要与签名
  veristack keygen <私钥文件>     生成 Ed25519 签名密钥对

示例:
  veristack verify ./delivery-pack
  veristack keygen ./signing.key
`

func main() {
	if len(os.Args) < 2 {
		fmt.Print(usage)
		os.Exit(1)
	}
	switch os.Args[1] {
	case "verify":
		os.Exit(runVerify(os.Args[2:]))
	case "keygen":
		os.Exit(runKeygen(os.Args[2:]))
	case "help", "-h", "--help":
		fmt.Print(usage)
	default:
		fmt.Printf("未知命令：%s\n\n%s", os.Args[1], usage)
		os.Exit(1)
	}
}

// runVerify 核验指定目录下的 Evidence Pack 并输出核验报告。
func runVerify(args []string) int {
	if len(args) == 0 {
		fmt.Println("请指定 Evidence Pack 目录")
		return 1
	}
	report, err := pack.Verify(args[0])
	if err != nil {
		fmt.Fprintf(os.Stderr, "核验失败：%v\n", err)
		return 1
	}
	printReport(report)
	if report.Summary != "全部核验通过，交付内容与签名均有效" {
		return 1
	}
	return 0
}

// printReport 以可读形式输出核验报告。
func printReport(report pack.Report) {
	fmt.Printf("交付：%s v%d（基线 v%d）\n", report.Pack.Delivery.ProjectName, report.Pack.Delivery.Version, report.Pack.Delivery.BaselineVersion)
	fmt.Printf("清单摘要：%s\n", report.Pack.ManifestSHA)
	if report.Pack.Delivery.GitRef != "" {
		fmt.Printf("Git 提交：%s\n", report.Pack.Delivery.GitRef)
	}
	fmt.Println()
	for _, check := range report.Checks {
		mark := "通过"
		if !check.Passed {
			mark = "异常"
		}
		fmt.Printf("[%s] %s：%s\n", mark, check.Name, check.Detail)
	}
	fmt.Println()
	fmt.Printf("结论：%s\n", report.Summary)
}

// runKeygen 生成 Ed25519 密钥对，公钥写入同名 .pub 文件。
func runKeygen(args []string) int {
	if len(args) == 0 {
		fmt.Println("请指定私钥文件路径")
		return 1
	}
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		fmt.Fprintf(os.Stderr, "生成密钥失败：%v\n", err)
		return 1
	}
	privateBytes, err := os.ReadFile(args[0])
	if err == nil && len(privateBytes) > 0 {
		fmt.Fprintln(os.Stderr, "目标文件已存在，为避免覆盖密钥已退出")
		return 1
	}
	if err := os.MkdirAll(filepath.Dir(args[0]), 0o750); err != nil {
		fmt.Fprintf(os.Stderr, "创建目录失败：%v\n", err)
		return 1
	}
	if err := os.WriteFile(args[0], privateKey, 0o600); err != nil {
		fmt.Fprintf(os.Stderr, "写入私钥失败：%v\n", err)
		return 1
	}
	publicInfo := map[string]string{
		"algorithm": pack.SignatureAlgorithm,
		"publicKey": base64.StdEncoding.EncodeToString(publicKey),
	}
	content, err := json.MarshalIndent(publicInfo, "", "  ")
	if err != nil {
		fmt.Fprintf(os.Stderr, "序列化公钥失败：%v\n", err)
		return 1
	}
	if err := os.WriteFile(args[0]+".pub", content, 0o644); err != nil {
		fmt.Fprintf(os.Stderr, "写入公钥失败：%v\n", err)
		return 1
	}
	fmt.Printf("私钥已写入 %s\n公钥已写入 %s.pub\n", args[0], args[0])
	return 0
}
