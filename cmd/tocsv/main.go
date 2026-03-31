package main

import (
	"os"
	"path/filepath"
	"strings"

	"github.com/xuri/excelize/v2"
)

func init() {
	os.Args = append(os.Args, "/Users/zzy/Downloads/对上1.xlsx")
}

func main() {
	filename, err := filepath.Abs(os.Args[1])
	if err != nil {
		panic(err)
	}

	excel, err := excelize.OpenFile(filename)
	if err != nil {
		panic(err)
	}
	defer excel.Close()

	rows, err := excel.Rows("Sheet1")
	if err != nil {
		panic(err)
	}

	var (
		buf     [][]string
		columns []string
	)

	for rows.Next() {
		if columns, err = rows.Columns(); err != nil {
			panic(err)
		}

		buf = append(buf, columns)
	}

	var sb strings.Builder
	for _, row := range buf {
		sb.WriteString(strings.Join(row, ","))
		sb.WriteString("\n")
	}

	filename = strings.ReplaceAll(filename, ".xlsx", ".csv")

	if err = os.WriteFile(filename, []byte(sb.String()), os.ModePerm); err != nil {
		panic(err)
	}
}
