package main

import (
	"bytes"
	"fmt"
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

func parseNumbers(line string, index int) (csv CSVLine) {
	var (
		unit     string
		building string
		resident string
	)

	var numbers = strings.Split(line, "-")
	if len(numbers) != 3 {
		log.Println("not is three numbers")
		return
	}

	var fields = strings.Split(line, ",")
	if index >= len(fields) {
		log.Println("is too few fields")
		return
	}

	if fields = strings.Split(fields[index], "-"); len(fields) != 3 {
		log.Println("not is three fields")
		return
	}

	return CSVLine{}
}

func SortCSV(filename string, index int, header bool) (err error) {
	data, err := os.ReadFile(filename)
	if err != nil {
		return
	}

	var head string
	var lines []CSVLine
	for i, line := range strings.Split(string(data), "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		if header && head == "" {
			head = line
			continue
		}

		lines = append(lines, parseNumbers(line))

		lines = append(lines, CSVLine{
			Line:     line,
			Building: fields[0],
			Unit:     fields[1],
			Resident: fields[2],
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
	if head != "" {
		buf.WriteString(head)
		buf.WriteString("\n")
	}

	for _, line := range lines {
		buf.WriteString(line.Line)
		buf.WriteString("\n")
	}

	if err = os.WriteFile(strings.ReplaceAll(filename, ".csv", "_排序.csv"), buf.Bytes(), 0666); err != nil {
		return
	}

	return
}

func main01() {
	// const filename = "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/5月1日 预约测量/5 月 1 日预约测量.csv"
	const filename = "/Users/zzy/Downloads/下单pdf导出楼号.csv"

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
		if len(fields) < 1 {
			log.Println(i, "is too few fields")
			continue
		}

		if fields[0] == "序号" || fields[0] == "楼号" {
			header = line
			continue
		}

		var numbers = strings.Split(fields[0], "-")
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

func main02() {
	const nums = `
17-1-101
12-6-301
16-1-301
12-3-602
12-3-602
1-2-402
15-4-302
15-6-201
15-4-302
16-1-502
1-10-602
1-10-101
4-1-604
17-2-603
1-2-402
1-2-402
1-13-402
17-8-502
3-1-602
17-2-301
16-2-502
16-2-502
16-13-402
11-4-301
4-4-101
13-8-501
17-2-403
16-1-502
15-6-201
14-1-301
16-1-301
17-2-403
15-3-502
4-1-604
15-5-102
16-1-301
1-10-201
1-2-402
17-3-101
1-3-602
17-2-303
11-6-602
4-1-604
13-3-101
4-1-601
1-13-402
5-3-602
16-2-502
1-10-101
3-1-602
11-6-602
3-1-602
13-3-101
1-13-402
17-2-303
6-6-602
17-3-101
16-8-101
1-7-402
15-2-101
16-3-602
17-1-101
17-1-302
1-7-402
13-3-101
11-6-602
16-2-502
16-2-502
3-2-201
17-1-101
17-2-603
17-2-403
13-3-101
1-1-201
16-1-301
4-4-101
16-1-502
16-8-402
1-12-601
15-6-201
15-5-102
6-6-602
15-4-302
16-1-301
1-13-402
3-2-201
17-1-302
16-1-502
17-2-403
1-12-601
14-1-301
15-4-302
1-12-601
4-4-101
11-4-301
15-3-502
11-4-301
15-6-201
16-2-501
17-1-302
11-4-301
17-2-403
1-2-402
17-1-302
5-2-201
17-3-101
5-3-602
17-2-301
4-1-604
5-2-201
5-3-602
15-5-102
6-1-602
15-5-102
16-2-501
14-1-301
3-2-201
16-1-301
1-2-402
15-6-201
15-5-102
15-5-102
12-3-602
1-2-402
17-2-303
13-8-501
12-6-301
17-1-101
13-3-602
1-10-201
16-1-301
15-4-302
16-1-301
12-6-301
15-3-502
13-8-501
6-1-602
16-1-502
1-7-402
6-6-602
16-5-602
16-1-601
16-5-601
16-5-602
16-3-602
6-1-602
17-1-101
17-2-603
15-4-302
13-8-501
1-10-201
17-2-303
1-2-402
11-4-301
16-1-502
17-1-101
13-8-501
6-1-602
17-2-303
14-1-301
12-6-301
16-8-402
1-12-601
15-3-502
15-3-502
`

	var hu = make(map[string]int)

	for _, line := range strings.Split(nums, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		hu[line] += 1
	}

	fmt.Println("hu, len:", len(hu))

	for hu, count := range hu {
		fmt.Println(fmt.Sprintf("%s,%d", hu, count))
	}
}

func main() {
	const filename = "/Users/zzy/Downloads/已发货清单.csv"

	if err := SortCSV(filename, 6, true); err != nil {
		panic(err)
	}
}
