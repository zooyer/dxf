package main

import (
	"fmt"
	"sort"

	"github.com/zooyer/dxf"
	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
	"github.com/zooyer/dxf/utils"
)

type Form struct {
	doc  *dxf.Document         // 文档
	tka4 *entities.Insert      // A4纸，名称TKA4
	scs  []*entities.Insert    // 楼号信息，名称SC
	pjs  []core.BBox           // 楼号窗户，图层PJ
	bzs  []*entities.Dimension // 窗户标注，图层BZ
}

func (f Form) getAttr(key string) string {
	for _, sc := range f.scs {
		// 直接返回第一个
		//return utils.GetAttr(sc, key)

		if attr := utils.GetAttr(sc, key); attr != "" {
			return attr
		}
	}

	return ""
}

func (f Form) BBox() core.BBox {
	return utils.GetEntityBBoxWCS(f.doc, f.tka4)
}

func (f Form) Area() string {
	return f.getAttr("面积")
}

func (f Form) Amount() string {
	return f.getAttr("金额")
}

func (f Form) Serial() string {
	return f.getAttr("序号")
}

func (f Form) Building() string {
	return f.getAttr("楼号")
}

func (f Form) Phone() string {
	return f.getAttr("手机号")
}

//func (f Form) Windows() (windows []Window) {
//	// 合并散线为矩形
//	var boxes = utils.MergeBoxes(f.pjs, winGap)
//
//	// 排序窗户 (从上到下)
//	sort.Slice(boxes, func(i, j int) bool {
//		if math.Abs(boxes[i].Max.Y-boxes[j].Max.Y) > 500 {
//			return boxes[i].Max.Y > boxes[j].Max.Y
//		}
//		return boxes[i].Min.X < boxes[j].Min.X
//	})
//
//	for i, box := range boxes {
//		var (
//			area  = box                 // 扩展范围
//			alls  = f.bzs               // 所有标注
//			curr  []*entities.Dimension // 当前标注
//			nears []*entities.Dimension // 附近标注
//		)
//
//		fmt.Println(fmt.Sprintf("窗户%d:", i+1))
//
//		for {
//			if alls, curr, area = getBZ3(f.doc, alls, area, widthFactor, heightFactor); len(curr) == 0 {
//				break
//			}
//
//			nears = append(nears, curr...)
//		}
//
//		var widths, heights []float64
//
//		for _, near := range nears {
//			// 标准化角度到 0-360
//			//var angle = math.Mod(near.Angle, 360)
//			//if angle < 0 {
//			//	angle += 360
//			//}
//			//
//			//isW := angle < 45 || angle > 315 || (angle > 135 && angle < 225)
//
//			var value = utils.GetDimValue(f.doc, near)
//
//			switch int(near.Angle) {
//			case 0, 180:
//				widths = append(widths, value)
//			case 90, 270:
//				heights = append(heights, value)
//			}
//		}
//
//		windows = append(windows, Window{
//			Box:     box,
//			Area:    area,
//			Label:   nears,
//			Widths:  widths,
//			Heights: heights,
//		})
//	}
//
//	return
//}

// getBox 查找当前及子结构中所有在图层中的实体组件(递归)
func getBox(doc *dxf.Document, layer string, entity entities.Entity, parent *entities.Insert) (boxes []core.BBox) {
	if entity == nil {
		return
	}

	// 收集 PJ 层线条
	if entity.Layer() == layer {
		if parent == nil {
			boxes = append(boxes, entity.BBox())
		} else {
			boxes = append(boxes, utils.TransformBBox(entity.BBox(), parent))
		}
	}

	insert, ok := entity.(*entities.Insert)
	if !ok || doc == nil {
		return
	}

	block, exists := doc.Blocks[insert.BlockName]
	if !exists || block == nil {
		return
	}

	if parent != nil {
		insert = utils.CombineInserts(parent, insert)
	}

	for _, sub := range block.Entities {
		for _, box := range getBox(doc, layer, sub, insert) {
			boxes = append(boxes, box)
		}
	}

	return
}

// 提取门窗确认单
func getForms(doc *dxf.Document) []Form {
	if doc == nil {
		return nil
	}

	var (
		pjs []core.BBox
		scs []*entities.Insert
		a4s []*entities.Insert
		bzs []*entities.Dimension
	)

	// 1. 提取所有组件、信息
	// 确认单A4(名称TKA4)、楼号信息(名称SC)、楼号门窗(图层PJ)、门窗标注(图层BZ)
	fmt.Printf("[开始处理]: %d 个实体组件...\n", len(doc.Entities))
	for _, entity := range doc.Entities {
		switch e := entity.(type) {
		case *entities.Insert:
			switch e.BlockName {
			case "SC":
				scs = append(scs, e)
			case "TKA4":
				a4s = append(a4s, e)
			}
		case *entities.Dimension:
			bzs = append(bzs, e)
		}

		pjs = append(pjs, getBox(doc, "PJ", entity, nil)...)
	}

	// 2. 排序确认单A4 TKA4 (按 X 坐标，从左到右，符合人类阅读)
	sort.Slice(a4s, func(i, j int) bool {
		return a4s[i].InsertionPoint.X < a4s[j].InsertionPoint.X
	})

	// 3. 计算包含、相邻关系，划分组件、信息归属
	var forms = make([]Form, 0, len(a4s))
	fmt.Printf("[开始处理]: %d 个门窗数据...\n", len(a4s))
	for _, a4 := range a4s {
		var (
			box   = utils.GetEntityBBoxWCS(doc, a4)
			attrs []*entities.Insert
		)

		// 提取 SC 属性
		for _, sc := range scs {
			if utils.InBox(box, sc.InsertionPoint) {
				attrs = append(attrs, sc)
			}
		}

		// 提取图框内的 PJ 窗户散线
		var innerPJ []core.BBox
		for _, pb := range pjs {
			midX := (pb.Min.X + pb.Max.X) / 2
			midY := (pb.Min.Y + pb.Max.Y) / 2
			if utils.InBox(box, core.Point{X: midX, Y: midY, Z: 0}) {
				innerPJ = append(innerPJ, pb)
			}
		}

		forms = append(forms, Form{
			doc:  doc,
			tka4: a4,
			scs:  attrs,
			pjs:  innerPJ,
			bzs:  bzs,
		})
	}

	return forms
}

func main() {
	const filename = "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/4月28日 纱窗核对/铝合金.dxf"

	doc, err := dxf.Open(filename)
	if err != nil {
		panic(err)
	}

	var forms = getForms(doc)

	for _, form := range forms {
		fmt.Println(form.Building(), form.Area(), form.Amount(), form.Serial())
	}
}
