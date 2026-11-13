package main

import (
	"bytes"
	"log"
	"os"
	"slices"
	"strings"
)

type CSVLine struct {
	Line     string
	Building string
	Unit     string
	Resident string
}

func CompareStringNumber(s1, s2 string) int {
	if len(s1) != len(s2) {
		if len(s1) < len(s2) {
			return -1
		}
		return 1
	}

	return strings.Compare(s1, s2)
}

func main() {
	const filename = "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/5月1日 预约测量/5 月 1 日预约测量.csv"

	data, err := os.ReadFile(filename)
	if err != nil {
		panic(err)
	}

	var (
		lines  []CSVLine
		header string
	)

	for i, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			log.Println(i, "is empty line")
			continue
		}

		var fields = strings.Split(line, ",")
		if len(fields) < 2 {
			log.Println(i, "is too few fields")
			continue
		}

		if fields[0] == "序号" {
			header = line
			continue
		}

		var numbers = strings.Split(fields[1], "-")
		if len(numbers) != 3 {
			log.Println(i, "not is three numbers")
			continue
		}

		lines = append(lines, CSVLine{
			Line:     line,
			Building: numbers[0],
			Unit:     numbers[1],
			Resident: numbers[2],
		})
	}

	slices.SortFunc(lines, func(a, b CSVLine) (cmp int) {
		if cmp = CompareStringNumber(a.Building, b.Building); cmp != 0 {
			return
		}

		if cmp = CompareStringNumber(a.Unit, b.Unit); cmp != 0 {
			return
		}

		if cmp = CompareStringNumber(a.Resident, b.Resident); cmp != 0 {
			return
		}

		return
	})

	var buf bytes.Buffer

	buf.WriteString(header)
	buf.WriteString("\n")
	for _, line := range lines {
		buf.WriteString(line.Line)
		buf.WriteString("\n")
	}

	if err = os.WriteFile(strings.ReplaceAll(filename, ".csv", ".sorted.csv"), buf.Bytes(), 0666); err != nil {
		panic(err)
	}
}
