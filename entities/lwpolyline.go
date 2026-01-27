package entities

import (
	"math"

	"github.com/zooyer/dxf/core"
)

type LWPolyline struct {
	BaseEntity
	Vertices []core.Point
}

func init() {
	Register("LWPOLYLINE", func() Entity { return &LWPolyline{BaseEntity: BaseEntity{TypeName: "LWPOLYLINE"}} })
}

func (l *LWPolyline) Parse(s *core.Scanner) error {
	var x float64
	for {
		t := s.LastTag
		switch t.Code {
		case 8:
			l.LayerName = t.AsString()
		case 10:
			x = t.AsFloat()
		case 20:
			l.Vertices = append(l.Vertices, core.Point{X: x, Y: t.AsFloat()})
		}
		if !s.Next() || s.LastTag.Code == 0 {
			break
		}
	}
	return nil
}

func (l *LWPolyline) BBox() core.BBox {
	if len(l.Vertices) == 0 {
		return core.BBox{}
	}
	miX, miY, maX, maY := l.Vertices[0].X, l.Vertices[0].Y, l.Vertices[0].X, l.Vertices[0].Y
	for _, v := range l.Vertices {
		miX = math.Min(miX, v.X)
		miY = math.Min(miY, v.Y)
		maX = math.Max(maX, v.X)
		maY = math.Max(maY, v.Y)
	}
	return core.BBox{Min: core.Point{X: miX, Y: miY}, Max: core.Point{X: maX, Y: maY}}
}
