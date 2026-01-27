package utils

import (
	"github.com/zooyer/dxf/entities"
)

func GetAttrs(ins *entities.Insert) map[string]string {
	var attrs = make(map[string]string)
	for _, a := range ins.Attributes {
		attrs[a.Tag] = a.Text
	}

	return attrs
}

func GetAttr(ins *entities.Insert, key string) string {
	return GetAttrs(ins)[key]
}
