package main

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	"github.com/zooyer/dxf"
	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
	"github.com/zooyer/dxf/utils"
	"github.com/zooyer/golib/xmath"
	"github.com/zooyer/golib/xos"
)

const (
	bzGap  = 20
	winGap = 10
)

type Window struct {
	Box     core.BBox             // 门窗范围(纯门窗面积)
	Area    core.BBox             // 覆盖范围(含标注面积)
	Label   []*entities.Dimension // 所有标注
	Widths  []float64             // 标注宽度
	Heights []float64             // 标注高度
}

func (w Window) Width() float64 {
	return math.Floor(w.Box.Max.X - w.Box.Min.X)
}

func (w Window) Height() float64 {
	return math.Floor(w.Box.Max.Y - w.Box.Min.Y)
}

func (w Window) MaxWidth() float64 {
	if len(w.Widths) < 1 {
		return 0
	}

	return slices.Max(w.Widths)
}

func (w Window) MaxHeight() float64 {
	if len(w.Heights) < 1 {
		return 0
	}

	return slices.Max(w.Heights)
}

func (w Window) VerifyWidth(epsilon float64) bool {
	if len(w.Widths) > 0 && xmath.Equal(w.Width(), slices.Max(w.Widths), epsilon) {
		return true
	}

	return false
}

func (w Window) VerifyHeight(epsilon float64) bool {
	if len(w.Heights) > 0 && xmath.Equal(w.Height(), slices.Max(w.Heights), epsilon) {
		return true
	}

	return false
}

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

func (f Form) Windows() (windows []Window) {
	// 合并散线为矩形
	var boxes = utils.MergeBoxes(f.pjs, winGap)

	// 排序窗户 (从上到下)
	sort.Slice(boxes, func(i, j int) bool {
		if math.Abs(boxes[i].Max.Y-boxes[j].Max.Y) > 500 {
			return boxes[i].Max.Y > boxes[j].Max.Y
		}
		return boxes[i].Min.X < boxes[j].Min.X
	})

	for _, box := range boxes {
		var (
			area  = box                 // 扩展范围
			alls  = f.bzs               // 所有标注
			curr  []*entities.Dimension // 当前标注
			nears []*entities.Dimension // 附近标注
		)

		for {
			if alls, curr, area = getBZ(f.doc, alls, area, bzGap); len(curr) == 0 {
				break
			}

			// 打印每次扩展范围
			// TODO debug
			//fmt.Printf("RECTANG %f,%f %f,%f\n", wr.Min.X, wr.Min.Y, wr.Max.X, wr.Max.Y)

			nears = append(nears, curr...)
		}

		var widths, heights []float64

		for _, near := range nears {
			// 标准化角度到 0-360
			//var angle = math.Mod(near.Angle, 360)
			//if angle < 0 {
			//	angle += 360
			//}
			//
			//isW := angle < 45 || angle > 315 || (angle > 135 && angle < 225)

			var value = utils.GetDimValue(f.doc, near)

			switch int(near.Angle) {
			case 0, 180:
				widths = append(widths, value)
			case 90, 270:
				heights = append(heights, value)
			}
		}

		windows = append(windows, Window{
			Box:     box,
			Area:    area,
			Label:   nears,
			Widths:  widths,
			Heights: heights,
		})
	}

	return
}

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
	if !exists {
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

// getBZ 寻找与当前 box 邻近的标注
// 返回：未被匹配的标注(rest)、本次匹配到的标注(near)、扩充后的新盒子(newBox)
func getBZ(doc *dxf.Document, bzs []*entities.Dimension, box core.BBox, gap float64) (rest, near []*entities.Dimension, newBox core.BBox) {
	newBox = box // 初始继承旧盒子

	for _, bz := range bzs {
		// 只要转角标注
		if bz.DimType != 0 {
			rest = append(rest, bz)
			continue
		}

		var exe = 0.0

		if style, ok := doc.DimStyles[bz.StyleName]; ok {
			exe = style.ExLimit * style.Scale
		}

		// 1. 精度判定：检查被测量的两个端点 (13 和 14) 是否挨着窗户
		// 使用很小的 gap (比如 10-50) 就能精准匹配
		startIn := !utils.IsSeparate(box, bz.BBox2(exe), gap)
		endIn := !utils.IsSeparate(box, bz.BBox2(exe), gap)

		if startIn || endIn {
			// 匹配成功
			near = append(near, bz)

			b := bz.BBox2(exe)

			// 打印标注范围
			// TODO debug
			//fmt.Printf("BZ [%.0f] RECTANG %f,%f %f,%f\n", GetBZValue(doc, bz), b.Min.X, b.Min.Y, b.Max.X, b.Max.Y)

			// 2. 盒子扩充：按照最远的点（10, 11, 13, 14）补全成最大矩形
			// 这样下一轮迭代就能通过“标注线”抓到更外圈的“总尺寸”标注
			points := []core.Point{b.Min, b.Max}
			//points := []core.Point{bz.TextMidPoint, bz.DefPoint, bz.MeasureStart, bz.MeasureEnd}
			for _, p := range points {
				if p.X < newBox.Min.X {
					newBox.Min.X = p.X
				}
				if p.Y < newBox.Min.Y {
					newBox.Min.Y = p.Y
				}
				if p.X > newBox.Max.X {
					newBox.Max.X = p.X
				}
				if p.Y > newBox.Max.Y {
					newBox.Max.Y = p.Y
				}
			}
		} else {
			// 没匹配上的放回池子
			rest = append(rest, bz)
		}
	}
	return
}

func renderBool(b bool) string {
	if b {
		return "✅"
	}

	return "❌"
}

func init() {
	//os.Args = append(os.Args, "test/testdata/洞口图纸10.dxf")

	if len(os.Args) < 2 {
		fmt.Println("请把PDF文件拖入该程序上执行！")
		xos.PauseExit()
		os.Exit(1)
	}
}

func main() {
	defer xos.PauseExit()

	doc, err := dxf.Open(os.Args[1])
	if err != nil {
		panic(err)
	}

	var (
		pjs []core.BBox
		scs []*entities.Insert
		a4s []*entities.Insert
		bzs []*entities.Dimension
	)

	// 1. 提取所有组件、信息
	// 确认单A4(名称TKA4)、楼号信息(名称SC)、楼号门窗(图层PJ)、门窗标注(图层BZ)
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

	fmt.Printf("开始处理: %d 个门窗数据...\n", len(a4s))

	// 3. 计算包含、相邻关系，划分组件、信息归属
	var forms = make([]Form, 0, len(a4s))
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

	// 4. 写入表头
	var filename = strings.TrimSuffix(os.Args[1], filepath.Ext(os.Args[1])) + ".csv"
	_ = os.WriteFile(filename, []byte("序号,楼号,宽度,高度,校验\n"), 0644)
	fmt.Println("写入文件:", filename)
	fmt.Println()

	// 5. 写入表格，打印输出
	for i, form := range forms {
		var (
			box  = form.BBox()
			wins = form.Windows()
		)

		// 打印信息
		fmt.Printf("[TKA4.%02d] | (%.2f, %.2f) - (%.2f, %.2f)         | %s\n",
			i+1, box.Min.X, box.Min.Y, box.Max.X, box.Max.Y, renderBool(len(form.scs) == 1),
		)
		for j, sc := range form.scs {
			var (
				area     = utils.GetAttr(sc, "面积")
				amount   = utils.GetAttr(sc, "金额")
				serial   = utils.GetAttr(sc, "序号")
				building = utils.GetAttr(sc, "楼号")
			)
			fmt.Printf("    [SC.%02d] | 序号:%s 金额:%s 面积:%s 楼号:%s\n",
				j+1, serial, amount, area, building,
			)
		}

		for j, w := range wins {
			// 打印信息
			fmt.Printf("    [窗户%d] | %.0f x %.0f | RECTANG %.0f,%.0f %.0f,%.0f\n",
				j+1, w.Width(), w.Height(), w.Box.Min.X, w.Box.Min.Y, w.Box.Max.X, w.Box.Max.Y,
			)
			fmt.Println("      W:", w.Widths, renderBool(w.VerifyWidth(1)))
			fmt.Println("      H:", w.Heights, renderBool(w.VerifyHeight(1)))

			var valid = renderBool(w.VerifyWidth(1) && w.VerifyHeight(1))

			var line = fmt.Sprintf(",,%.0f,%.0f,%s\n", w.MaxWidth(), w.MaxHeight(), valid)

			if j == 0 {
				line = fmt.Sprintf("%s,%s,%.0f,%.0f,%s\n", form.Serial(), form.Building(), w.MaxWidth(), w.MaxHeight(), valid)
			}

			// fmt.Print(line)

			if err = xos.AppendFile(filename, []byte(line), 0644); err != nil {
				panic(err)
			}
		}

		// 填充空行，至少7行
		for j := len(wins); j < 7; j++ {
			var empty = ",,,,\n"

			//fmt.Print(empty)

			if err = xos.AppendFile(filename, []byte(empty), 0644); err != nil {
				panic(err)
			}
		}
	}
}
