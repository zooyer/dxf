package core

import (
	"strconv"
	"strings"
)

// Tag 代表 DXF 中的一组标签对
type Tag struct {
	Code  int
	Value string
}

// AsFloat 将值转换为 float64
func (t Tag) AsFloat() float64 {
	f, _ := strconv.ParseFloat(strings.TrimSpace(t.Value), 64)
	return f
}

// AsInt 将值转换为 int
func (t Tag) AsInt() int {
	i, _ := strconv.Atoi(strings.TrimSpace(t.Value))
	return i
}

// AsString 清洗字符串（去除多余空格）
func (t Tag) AsString() string {
	return strings.TrimSpace(t.Value)
}

// Point 代表三维空间中的一个点
type Point struct {
	X, Y, Z float64
}

// BBox 代表包围盒
type BBox struct {
	Min, Max Point
}
