package entities

import "github.com/zooyer/dxf/core"

type Insert struct {
	BaseEntity
	BlockName      string
	InsertionPoint core.Point
	Scale          core.Point
	Rotation       float64
	Attributes     []*Attrib
}

func init() {
	Register("INSERT", func() Entity {
		return &Insert{
			BaseEntity: BaseEntity{TypeName: "INSERT"},
			Scale:      core.Point{X: 1, Y: 1, Z: 1}, // 默认缩放为 1
			Attributes: []*Attrib{},
		}
	})
}

func (i *Insert) Parse(scanner *core.Scanner) error {
	hasAttributes := false

	for {
		tag := scanner.LastTag
		switch tag.Code {
		case 2:
			i.BlockName = tag.AsString()
		case 8:
			i.LayerName = tag.AsString()
		case 10:
			i.InsertionPoint.X = tag.AsFloat()
		case 20:
			i.InsertionPoint.Y = tag.AsFloat()
		case 30:
			i.InsertionPoint.Z = tag.AsFloat()
		case 41:
			i.Scale.X = tag.AsFloat()
		case 42:
			i.Scale.Y = tag.AsFloat()
		case 43:
			i.Scale.Z = tag.AsFloat()
		case 50:
			i.Rotation = tag.AsFloat()
		case 66:
			if tag.AsInt() == 1 {
				hasAttributes = true
			}
		}

		if !scanner.Next() || scanner.LastTag.Code == 0 {
			break
		}
	}

	// 核心逻辑：如果标记了有属性，则继续在当前流中抓取 ATTRIB 直到 SEQEND
	if hasAttributes {
		for {
			tag := scanner.LastTag
			if tag.Code == 0 {
				if tag.Value == "SEQEND" {
					scanner.Next() // 消耗掉 SEQEND
					break
				}
				// 创建子实体
				subEntity := CreateEntity(tag.Value)
				if attr, ok := subEntity.(*Attrib); ok {
					attr.Parse(scanner)
					i.Attributes = append(i.Attributes, attr)
					continue // Parse 内部已经 Next 了，直接进入下一次判断
				}
			}
			if !scanner.Next() {
				break
			}
		}
	}
	return nil
}

func (i *Insert) BBox() core.BBox {
	// Insert 的包围盒比较特殊，通常需要结合 Block 定义计算
	// 这里先返回插入点
	return core.BBox{Min: i.InsertionPoint, Max: i.InsertionPoint}
}
