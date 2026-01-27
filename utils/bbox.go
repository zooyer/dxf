package utils

import (
	"math"
	"strings"

	"github.com/zooyer/dxf"
	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
)

// TransformBBox 执行矩阵变换：将局部坐标变换到插入点所在的世界坐标
func TransformBBox(local core.BBox, ins *entities.Insert) core.BBox {
	rad := ins.Rotation * math.Pi / 180.0
	cos, sin := math.Cos(rad), math.Sin(rad)

	corners := []core.Point{
		{X: local.Min.X, Y: local.Min.Y, Z: local.Min.Z},
		{X: local.Max.X, Y: local.Min.Y, Z: local.Min.Z},
		{X: local.Max.X, Y: local.Max.Y, Z: local.Min.Z},
		{X: local.Min.X, Y: local.Max.Y, Z: local.Min.Z},
		{X: local.Min.X, Y: local.Min.Y, Z: local.Max.Z},
		{X: local.Max.X, Y: local.Min.Y, Z: local.Max.Z},
		{X: local.Max.X, Y: local.Max.Y, Z: local.Max.Z},
		{X: local.Min.X, Y: local.Max.Y, Z: local.Max.Z},
	}

	wMinX, wMinY, wMinZ := math.MaxFloat64, math.MaxFloat64, math.MaxFloat64
	wMaxX, wMaxY, wMaxZ := -math.MaxFloat64, -math.MaxFloat64, -math.MaxFloat64

	for _, p := range corners {
		// 缩放
		tx, ty, tz := p.X*ins.Scale.X, p.Y*ins.Scale.Y, p.Z*ins.Scale.Z

		// XY 旋转（绕 Z 轴）
		rx := tx*cos - ty*sin
		ry := tx*sin + ty*cos

		// 平移
		wx, wy, wz := rx+ins.InsertionPoint.X, ry+ins.InsertionPoint.Y, tz+ins.InsertionPoint.Z

		wMinX = math.Min(wMinX, wx)
		wMinY = math.Min(wMinY, wy)
		wMinZ = math.Min(wMinZ, wz)
		wMaxX = math.Max(wMaxX, wx)
		wMaxY = math.Max(wMaxY, wy)
		wMaxZ = math.Max(wMaxZ, wz)
	}

	return core.BBox{
		Min: core.Point{X: wMinX, Y: wMinY, Z: wMinZ},
		Max: core.Point{X: wMaxX, Y: wMaxY, Z: wMaxZ},
	}
}

// MergeBoxes 合并重叠的矩形
func MergeBoxes(boxes []core.BBox, gap float64) []core.BBox {
	if len(boxes) < 2 {
		return boxes
	}

	for {
		changed := false
		var merged []core.BBox
		visited := make([]bool, len(boxes))
		for i := 0; i < len(boxes); i++ {
			if visited[i] {
				continue
			}
			curr := boxes[i]
			visited[i] = true
			for j := i + 1; j < len(boxes); j++ {
				if !visited[j] && !IsSeparate(curr, boxes[j], gap) {
					curr.Min.X = math.Min(curr.Min.X, boxes[j].Min.X)
					curr.Min.Y = math.Min(curr.Min.Y, boxes[j].Min.Y)
					curr.Max.X = math.Max(curr.Max.X, boxes[j].Max.X)
					curr.Max.Y = math.Max(curr.Max.Y, boxes[j].Max.Y)
					visited[j], changed = true, true
				}
			}
			merged = append(merged, curr)
		}
		boxes = merged
		if !changed {
			break
		}
	}

	return boxes
}

// IsSeparate 判断两个 BBox 是否完全分离
func IsSeparate(a, b core.BBox, gap float64) bool {
	return a.Max.X+gap < b.Min.X || a.Min.X-gap > b.Max.X ||
		a.Max.Y+gap < b.Min.Y || a.Min.Y-gap > b.Max.Y
}

func InBox(box core.BBox, point core.Point) bool {
	if point.X >= box.Min.X && point.X <= box.Max.X && point.Y >= box.Min.Y && point.Y <= box.Max.Y {
		return true
	}

	return false
}

func GetEntityBBoxWCS(d *dxf.Document, entity entities.Entity) core.BBox {
	switch e := entity.(type) {
	case *entities.Insert:
		block, ok := d.Blocks[strings.ToUpper(e.BlockName)]
		if !ok || len(block.Entities) == 0 {
			return core.BBox{Min: e.InsertionPoint, Max: e.InsertionPoint}
		}

		miX, miY := math.MaxFloat64, math.MaxFloat64
		maX, maY := -math.MaxFloat64, -math.MaxFloat64

		for _, sub := range block.Entities {
			sb := sub.BBox()
			miX = math.Min(miX, sb.Min.X)
			miY = math.Min(miY, sb.Min.Y)
			maX = math.Max(maX, sb.Max.X)
			maY = math.Max(maY, sb.Max.Y)
		}

		localBox := core.BBox{
			Min: core.Point{X: miX, Y: miY, Z: 0},
			Max: core.Point{X: maX, Y: maY, Z: 0},
		}
		return TransformBBox(localBox, e)
	default:
		return e.BBox()
	}
}
