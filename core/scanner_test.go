package core

import (
	"strings"
	"testing"
)

func TestScanner_Basic(t *testing.T) {
	// 模拟一个简单的 DXF 片段
	dxfData := "0\nSECTION\n2\nHEADER\n0\nENDSEC\n"
	r := strings.NewReader(dxfData)
	scanner := NewScanner(r)

	expected := []Tag{
		{0, "SECTION"},
		{2, "HEADER"},
		{0, "ENDSEC"},
	}

	for i, exp := range expected {
		if !scanner.Next() {
			t.Fatalf("第 %d 步读取失败: %v", i, scanner.Err())
		}
		if scanner.LastTag.Code != exp.Code || scanner.LastTag.Value != exp.Value {
			t.Errorf("第 %d 步数据不符: 期望 %+v, 得到 %+v", i, exp, scanner.LastTag)
		}
	}
}
