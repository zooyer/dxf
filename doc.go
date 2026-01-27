package dxf

import (
	"io"
	"os"
	"strings"

	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
)

type DimStyle struct {
	Name      string
	Precision int     // 对应组码 271 DIMDEC，显示的小数位数
	ExLimit   float64 // 对应组码 44 DIMEXE，标注线超出延伸线的长度
	Scale     float64 // 对应组码 40 DIMSCALE，全局比例，影响所有标注特征)
}

type Block struct {
	Name     string
	Entities []entities.Entity
}

type Document struct {
	Blocks    map[string]*Block
	Entities  []entities.Entity
	DimStyles map[string]*DimStyle
}

func (d *Document) parseBlocks(scanner *core.Scanner) {
	var currentBlock *Block
	for scanner.Next() {
		tag := scanner.LastTag
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "ENDSEC" {
			break
		}
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "BLOCK" {
			currentBlock = &Block{Entities: []entities.Entity{}}
			for scanner.Next() {
				if scanner.LastTag.Code == 2 {
					currentBlock.Name = strings.ToUpper(scanner.LastTag.Value)
					break
				}
				if scanner.LastTag.Code == 0 {
					break
				}
			}
			d.Blocks[currentBlock.Name] = currentBlock
		}
		if currentBlock != nil && tag.Code == 0 &&
			tag.Value != "BLOCK" && tag.Value != "ENDBLK" {
			ent := entities.CreateEntity(tag.Value)
			if ent != nil {
				ent.Parse(scanner)
				currentBlock.Entities = append(currentBlock.Entities, ent)
			}
		}
	}
}

func (d *Document) parseEntities(scanner *core.Scanner) {
	for {
		tag := scanner.LastTag
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "ENDSEC" {
			break
		}
		if tag.Code == 0 {
			ent := entities.CreateEntity(tag.Value)
			if ent != nil {
				ent.Parse(scanner)
				d.Entities = append(d.Entities, ent)
				continue
			}
		}
		if !scanner.Next() {
			break
		}
	}
}
func (d *Document) parseTables(scanner *core.Scanner) {
	for scanner.Next() {
		tag := scanner.LastTag
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "ENDSEC" {
			break
		}
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "TABLE" {
			scanner.Next()
			tableName := strings.ToUpper(scanner.LastTag.Value)
			if tableName == "DIMSTYLE" {
				d.parseDimStyles(scanner)
			}
		}
	}
}

func (d *Document) parseDimStyles(scanner *core.Scanner) {
	var currentStyle *DimStyle
	for {
		tag := scanner.LastTag
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "ENDTAB" {
			break
		}

		if tag.Code == 0 && strings.ToUpper(tag.Value) == "DIMSTYLE" {
			currentStyle = &DimStyle{
				Precision: 0,
				ExLimit:   0.0,
				Scale:     1.0, // 默认为 1.0，防止乘法归零
			}

			for scanner.Next() {
				t := scanner.LastTag
				if t.Code == 0 {
					break
				}
				switch t.Code {
				case 2: // 样式名称
					currentStyle.Name = strings.ToUpper(t.Value)
				case 271: // 精度
					currentStyle.Precision = t.AsInt()
				case 44: // 标注线超出延伸线长度 (DIMEXE)
					currentStyle.ExLimit = t.AsFloat()
				case 40: // 全局标注比例 (DIMSCALE)
					currentStyle.Scale = t.AsFloat()
				}
			}

			if currentStyle.Name != "" {
				d.DimStyles[currentStyle.Name] = currentStyle
			}

			if scanner.LastTag.Code == 0 && strings.ToUpper(scanner.LastTag.Value) == "DIMSTYLE" {
				continue
			}
		}

		if !scanner.Next() {
			break
		}
	}
}

func Open(filename string) (doc *Document, err error) {
	file, err := os.Open(filename)
	if err != nil {
		return
	}

	defer func() {
		if e := file.Close(); e != nil && err == nil {
			err = e
		}
	}()

	return Load(file)
}

func Load(reader io.Reader) (doc *Document, err error) {
	var (
		scanner  = core.NewScanner(reader)
		document = &Document{
			Blocks:    make(map[string]*Block),
			Entities:  make([]entities.Entity, 0, 1024),
			DimStyles: make(map[string]*DimStyle),
		}
	)

	for scanner.Next() {
		tag := scanner.LastTag
		if tag.Code == 0 && strings.ToUpper(tag.Value) == "SECTION" {
			if !scanner.Next() {
				break
			}
			sectionName := strings.ToUpper(scanner.LastTag.Value)
			switch sectionName {
			case "TABLES":
				document.parseTables(scanner)
			case "BLOCKS":
				document.parseBlocks(scanner)
			case "ENTITIES":
				document.parseEntities(scanner)
			}
		}
	}

	return document, scanner.Err()
}
