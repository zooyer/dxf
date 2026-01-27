package utils

import (
	"math"
	"strings"

	"github.com/zooyer/dxf"
	"github.com/zooyer/dxf/entities"
)

func GetDimValue(doc *dxf.Document, dim *entities.Dimension) float64 {
	// 1. 如果有手动文字覆盖，直接按文字提取数字
	if dim.Text != "" && !strings.Contains(dim.Text, "<>") {
		return dim.GetCleanVal()
	}

	// 2. 查找标注样式定义的精度
	// 注意：Dimension 实体需要解析组码 3 (StyleName)
	precision := 0 // 默认取整
	if style, ok := doc.DimStyles[strings.ToUpper(dim.StyleName)]; ok {
		precision = style.Precision
	}

	// 3. 根据精度进行四舍五入
	p := math.Pow(10, float64(precision))

	return math.Round(dim.ActualMeasurement*p) / p
}
