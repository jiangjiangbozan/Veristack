// Package project 的导入逻辑单测，覆盖需求编号唯一性与里程碑引用校验。
package project

import (
	"strings"
	"testing"

	"github.com/google/uuid"

	"api/internal/scope"
)

// TestBuildImportRequirementsRejectsDuplicateCode 验证同一导入内需求编号不能重复。
func TestBuildImportRequirementsRejectsDuplicateCode(t *testing.T) {
	inputs := []scope.RequirementInput{
		{Code: "REQ-001", Priority: "high", Title: "登录"},
		{Code: "REQ-001", Priority: "medium", Title: "退出"},
	}

	_, _, err := buildImportRequirements(inputs)

	if err == nil || !strings.Contains(err.Error(), "重复") {
		t.Fatalf("应拒绝重复需求编号，实际错误: %v", err)
	}
}

// TestBuildImportRequirementsKeepsStableIDs 验证快照与映射使用同一个需求标识。
func TestBuildImportRequirementsKeepsStableIDs(t *testing.T) {
	inputs := []scope.RequirementInput{
		{AcceptanceCriteria: []string{"登录成功"}, Code: "REQ-001", Priority: "high", Title: "登录"},
	}

	viewsList, requirementIDs, err := buildImportRequirements(inputs)

	if err != nil {
		t.Fatalf("构建需求快照失败: %v", err)
	}
	if len(viewsList) != 1 || len(requirementIDs) != 1 {
		t.Fatalf("快照与编号映射数量不正确: views=%d ids=%d", len(viewsList), len(requirementIDs))
	}
	if requirementIDs["REQ-001"] != viewsList[0].ID {
		t.Fatalf("编号映射与快照标识不一致")
	}
}

// TestValidateImportMilestonesRejectsUnknownCode 验证里程碑引用未知需求编号时拒绝导入。
func TestValidateImportMilestonesRejectsUnknownCode(t *testing.T) {
	milestones := []ImportMilestone{{Name: "首次交付", RequirementCodes: []string{"REQ-999"}}}

	_, err := validateImportMilestones(milestones, map[string]uuid.UUID{})

	if err == nil || !strings.Contains(err.Error(), "不存在") {
		t.Fatalf("应拒绝未知需求编号，实际错误: %v", err)
	}
}
