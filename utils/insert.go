package utils

import (
	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
)

// CombineInserts 合并嵌套块的变换矩阵逻辑
func CombineInserts(parent, child *entities.Insert) *entities.Insert {
	// 1. 旋转叠加
	combinedRotation := parent.Rotation + child.Rotation

	// 2. 缩放叠加
	combinedScale := core.Point{
		X: parent.Scale.X * child.Scale.X,
		Y: parent.Scale.Y * child.Scale.Y,
		Z: parent.Scale.Z * child.Scale.Z,
	}

	// 3. 插入点叠加：子块的插入点需要经过父块的 缩放 -> 旋转 -> 平移 变换
	combinedInsertionPoint := TransformPoint(child.InsertionPoint, parent)

	return &entities.Insert{
		BlockName:      child.BlockName,
		Rotation:       combinedRotation,
		Scale:          combinedScale,
		InsertionPoint: combinedInsertionPoint,
	}
}
