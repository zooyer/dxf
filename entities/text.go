package entities

import (
	"github.com/zooyer/dxf/core"
)

type Text struct {
	BaseEntity // 假设你有一个基础实体包含图层等信息
	Value      string
	Point      core.Point // 插入点
	Height     float64
	Layer      string
}

func init() {
	Register("TEXT", func() Entity {
		return &Text{BaseEntity: BaseEntity{TypeName: "TEXT"}}
	})
}

func (t *Text) Parse(scanner *core.Scanner) {
	for scanner.Next() {
		tag := scanner.LastTag
		if tag.Code == 0 {
			break // 遇到下一个实体，退出
		}
		switch tag.Code {
		case 1:
			t.Value = tag.Value
		case 8:
			t.Layer = tag.Value
		case 10:
			t.Point.X = tag.AsFloat()
		case 20:
			t.Point.Y = tag.AsFloat()
		case 30:
			t.Point.Z = tag.AsFloat()
		case 40:
			t.Height = tag.AsFloat()
		}
	}
}
