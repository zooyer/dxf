package utils

import (
	"math"

	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
)

// TransformPoint 将局部坐标点经过 Insert 变换转换到父级/世界坐标
func TransformPoint(p core.Point, ins *entities.Insert) core.Point {
	rad := ins.Rotation * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)

	// 1. 缩放
	tx := p.X * ins.Scale.X
	ty := p.Y * ins.Scale.Y
	tz := p.Z * ins.Scale.Z

	// 2. 旋转
	rx := tx*cos - ty*sin
	ry := tx*sin + ty*cos

	// 3. 平移
	return core.Point{
		X: rx + ins.InsertionPoint.X,
		Y: ry + ins.InsertionPoint.Y,
		Z: tz + ins.InsertionPoint.Z,
	}
}
