package entities

import (
	"math"

	"github.com/zooyer/dxf/core"
)

type Line struct {
	BaseEntity
	Start, End core.Point
}

func init() {
	Register("LINE", func() Entity { return &Line{BaseEntity: BaseEntity{TypeName: "LINE"}} })
}

func (l *Line) Parse(s *core.Scanner) error {
	for {
		t := s.LastTag
		switch t.Code {
		case 8:
			l.LayerName = t.AsString()
		case 10:
			l.Start.X = t.AsFloat()
		case 20:
			l.Start.Y = t.AsFloat()
		case 11:
			l.End.X = t.AsFloat()
		case 21:
			l.End.Y = t.AsFloat()
		}
		if !s.Next() || s.LastTag.Code == 0 {
			break
		}
	}
	return nil
}

func (l *Line) BBox() core.BBox {
	return core.BBox{
		Min: core.Point{X: math.Min(l.Start.X, l.End.X), Y: math.Min(l.Start.Y, l.End.Y)},
		Max: core.Point{X: math.Max(l.Start.X, l.End.X), Y: math.Max(l.Start.Y, l.End.Y)},
	}
}
