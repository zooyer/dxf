package entities

import (
	"github.com/zooyer/dxf/core"
)

// Entity 是一切几何实体的接口
type Entity interface {
	Parse(scanner *core.Scanner) error
	Type() string
	Layer() string
	BBox() core.BBox
}

// BaseEntity 存放所有实体通用的属性（如 Layer, Color, Handle）
type BaseEntity struct {
	TypeName  string
	LayerName string
	Handle    string
}

func (b *BaseEntity) Type() string { return b.TypeName }

func (b *BaseEntity) Layer() string { return b.LayerName }

// EntityFactory 定义了如何从标签流中创建一个实体
type EntityFactory func() Entity

var registry = map[string]EntityFactory{}

// Register 允许以后动态扩展新的实体类型
func Register(typeName string, factory EntityFactory) {
	registry[typeName] = factory
}

// CreateEntity 根据实体名称生产对应的结构体
func CreateEntity(typeName string) Entity {
	if factory, ok := registry[typeName]; ok {
		return factory()
	}
	return nil
}
