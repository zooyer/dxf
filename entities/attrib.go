package entities

import "github.com/zooyer/dxf/core"

type Attrib struct {
	BaseEntity
	Location core.Point
	Tag      string // 属性标签，如 "序号"
	Text     string // 属性值
	Height   float64
}

func init() {
	Register("ATTRIB", func() Entity {
		return &Attrib{BaseEntity: BaseEntity{TypeName: "ATTRIB"}}
	})
}

func (a *Attrib) Parse(scanner *core.Scanner) error {
	for {
		tag := scanner.LastTag
		switch tag.Code {
		case 8:
			a.LayerName = tag.AsString()
		case 10:
			a.Location.X = tag.AsFloat()
		case 20:
			a.Location.Y = tag.AsFloat()
		case 30:
			a.Location.Z = tag.AsFloat()
		case 40:
			a.Height = tag.AsFloat()
		case 1:
			a.Text = tag.AsString()
		case 2:
			a.Tag = tag.AsString()
		}
		if !scanner.Next() || scanner.LastTag.Code == 0 {
			break
		}
	}
	return nil
}

func (a *Attrib) BBox() core.BBox {
	// 简化处理：属性文字暂时以位置点作为包围盒
	return core.BBox{Min: a.Location, Max: a.Location}
}
