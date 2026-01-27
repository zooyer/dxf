package entities

import (
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/zooyer/dxf/core"
)

type Dimension struct {
	BaseEntity
	DimType           int        // 组码 70 (关键：区分标注类型)
	StyleName         string     // 组码 3 (标注样式名称，用于关联 TABLES)
	ActualMeasurement float64    // 组码 42
	Text              string     // 组码 1
	Angle             float64    // 组码 50
	TextMidPoint      core.Point // 组码 11 (中间的点)
	DefPoint          core.Point // 组码 10 (标注线起点)
	MeasureStart      core.Point // 组码 13 (被测量的起点)
	MeasureEnd        core.Point // 组码 14 (被测量的终点)
}

func init() {
	Register("DIMENSION", func() Entity {
		return &Dimension{BaseEntity: BaseEntity{TypeName: "DIMENSION"}}
	})
}

func (d *Dimension) Parse(scanner *core.Scanner) error {
	for {
		tag := scanner.LastTag
		switch tag.Code {
		case 8:
			d.LayerName = tag.AsString()
		case 3:
			// 核心：读取标注样式名称
			d.StyleName = strings.ToUpper(tag.AsString())
		case 1:
			d.Text = tag.AsString()
		case 42:
			d.ActualMeasurement = tag.AsFloat()
		case 50:
			d.Angle = tag.AsFloat()
		// 解析 5 个核心点坐标
		case 10:
			d.DefPoint.X = tag.AsFloat()
		case 20:
			d.DefPoint.Y = tag.AsFloat()
		case 11:
			d.TextMidPoint.X = tag.AsFloat()
		case 21:
			d.TextMidPoint.Y = tag.AsFloat()
		case 13:
			d.MeasureStart.X = tag.AsFloat()
		case 23:
			d.MeasureStart.Y = tag.AsFloat()
		case 14:
			d.MeasureEnd.X = tag.AsFloat()
		case 24:
			d.MeasureEnd.Y = tag.AsFloat()
		case 70:
			// 组码 70 包含了很多信息，我们只需要低 3 位来判定类型
			d.DimType = tag.AsInt() & 0x07
		}
		if !scanner.Next() || scanner.LastTag.Code == 0 {
			break
		}
	}
	return nil
}

// BBox 覆盖：为了通用库的严谨性，标注的 BBox 应该包含所有定义点
func (d *Dimension) BBox() core.BBox {
	return d.BBox2(0)
}

// GetExtensionPoints 计算标注线上的两个转角点
// 返回：对应 P13 的转角点, 对应 P14 的转角点
func (d *Dimension) GetExtensionPoints() (p13Corner, p14Corner core.Point) {
	// 将角度从角度制转为弧度制
	rad := d.Angle * math.Pi / 180.0
	cos := math.Cos(rad)
	sin := math.Sin(rad)

	// 标注线的单位方向向量
	v := core.Point{X: cos, Y: sin}

	// 计算 P13 在标注线上的投影
	// 向量 (P13 - P10) 在方向向量 v 上的投影
	dx13 := d.MeasureStart.X - d.DefPoint.X
	dy13 := d.MeasureStart.Y - d.DefPoint.Y
	dot13 := dx13*v.X + dy13*v.Y

	p13Corner = core.Point{
		X: d.DefPoint.X + v.X*dot13,
		Y: d.DefPoint.Y + v.Y*dot13,
	}

	// 计算 P14 在标注线上的投影
	dx14 := d.MeasureEnd.X - d.DefPoint.X
	dy14 := d.MeasureEnd.Y - d.DefPoint.Y
	dot14 := dx14*v.X + dy14*v.Y

	p14Corner = core.Point{
		X: d.DefPoint.X + v.X*dot14,
		Y: d.DefPoint.Y + v.Y*dot14,
	}

	return
}

// BBox2 实现“完美矩形”包围盒
// exe 代表标注线超出延伸线的长度 (DIMEXE)
func (d *Dimension) BBox2(exe float64) core.BBox {
	// 1. 获取基础的转角投影点 (标注线上的两个端点)
	c13, c14 := d.GetExtensionPoints()

	// 2. 计算延伸线的方向向量 (垂直于标注线的方向)
	// 标注线角度是 d.Angle，延伸线角度是 d.Angle + 90°
	upRad := (d.Angle + 90.0) * math.Pi / 180.0
	u := core.Point{X: math.Cos(upRad), Y: math.Sin(upRad)}

	// 3. 计算“冒尖”后的顶点
	// 逻辑：从转角点 (c13, c14) 沿着 u 方向再往外推 exe 距离
	// 注意：这里需要判定 u 的方向是远离测量点还是靠近测量点
	// 我们通过向量 (c13 - MeasureStart) 与 u 的点积来判定方向
	vecToLine := core.Point{X: c13.X - d.MeasureStart.X, Y: c13.Y - d.MeasureStart.Y}
	dot := vecToLine.X*u.X + vecToLine.Y*u.Y

	direction := 1.0
	if dot < 0 {
		direction = -1.0
	}

	p13Top := core.Point{
		X: c13.X + u.X*exe*direction,
		Y: c13.Y + u.Y*exe*direction,
	}
	p14Top := core.Point{
		X: c14.X + u.X*exe*direction,
		Y: c14.Y + u.Y*exe*direction,
	}

	// 4. 收集 4 个关键顶点：2个测量原点 + 2个冒尖的顶点
	points := []core.Point{
		d.MeasureStart,
		d.MeasureEnd,
		p13Top,
		p14Top,
		d.TextMidPoint, // 文字位置
	}

	// 5. 计算包围盒
	minX, minY := math.MaxFloat64, math.MaxFloat64
	maxX, maxY := -math.MaxFloat64, -math.MaxFloat64
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	return core.BBox{
		Min: core.Point{X: minX, Y: minY},
		Max: core.Point{X: maxX, Y: maxY},
	}
}

// GetCleanVal 正则提取数值
func (d *Dimension) GetCleanVal() float64 {
	val := d.ActualMeasurement
	if val <= 0 && d.Text != "" {
		reFormat := regexp.MustCompile(`\\[A-Z].*?;`)
		cleanText := reFormat.ReplaceAllString(d.Text, "")
		reNum := regexp.MustCompile(`[0-9.]+`)
		if match := reNum.FindString(cleanText); match != "" {
			parsed, _ := strconv.ParseFloat(match, 64)
			val = parsed
		}
	}
	return val
}
