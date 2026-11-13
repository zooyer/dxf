package entities

import (
	"strings"

	"github.com/zooyer/dxf/core"
)

type MText struct {
	BaseEntity
	Value string
	Point core.Point
	Layer string
}

func (m *MText) Parse(scanner *core.Scanner) {
	var sb strings.Builder
	for scanner.Next() {
		tag := scanner.LastTag
		if tag.Code == 0 {
			break
		}
		switch tag.Code {
		case 1, 3: // MTEXT 的内容可能分段存在组码 1 和 3 里
			sb.WriteString(tag.Value)
		case 8:
			m.Layer = tag.Value
		case 10:
			m.Point.X = tag.AsFloat()
		case 20:
			m.Point.Y = tag.AsFloat()
		}
	}
	m.Value = sb.String()
}
