package main

import (
	"cmp"
	"fmt"
	"log"
	"slices"
	"strconv"
	"strings"

	"github.com/pdfcpu/pdfcpu/pkg/api"
	"github.com/pdfcpu/pdfcpu/pkg/pdfcpu/model"
)

// RemovePdfPages 接收输入路径、输出路径以及需要删除的页码切片
// 页码从 1 开始计数
func RemovePdfPages(inFile, outFile string, pagesToRemove []string) error {
	// 配置 pdfcpu 的处理参数，使用默认配置即可
	conf := model.NewDefaultConfiguration()

	// 调用 api.RemovePages
	// 参数说明：输入文件，输出文件，要删除的页码（字符串切片），配置项
	// 注意：如果 outFile 为空字符串，pdfcpu 可能会覆盖原文件
	err := api.RemovePagesFile(inFile, outFile, pagesToRemove, conf)
	if err != nil {
		return fmt.Errorf("删除页面失败: %v", err)
	}

	return nil
}

// DedupUsingOfficialLib 使用官方 slices 库进行去重
// 注意：这会改变原切片的顺序，并且由于 Compact 的特性，建议在调用前排序
func DedupUsingOfficialLib[T cmp.Ordered](input []T) []T {
	if len(input) < 2 {
		return input
	}

	// 1. 必须先排序，因为 Compact 只处理连续重复的元素
	slices.Sort(input)

	// 2. Compact 会原地处理并返回去重后的切片（长度可能变小）
	return slices.Compact(input)
}

func main下单() {
	inputPath := "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/测量下单/复尺.pdf"     // 源文件
	outputPath := "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/测量下单/复尺-去重.pdf" // 处理后的文件

	// 定义要删除的页码
	const deleted = "10,23,31,33,36,48,49,54,22,24,16,15,13,12,22"
	// 格式支持：单一数字 "2"、范围 "3-5"、或者从某页到最后 "n-"
	var pages []string
	var set = make(map[string]bool)
	for _, page := range strings.Split(deleted, ",") {
		pages = append(pages, page)
		set[page] = true
		fmt.Println("page:", page)
	}

	fmt.Printf("正在处理文件: %s，准备删除第 %v 页...\n", inputPath, pages)

	pages = make([]string, 0, len(pages))
	for page := range set {
		pages = append(pages, page)
	}

	err := RemovePdfPages(inputPath, outputPath, pages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	fmt.Printf("处理完成！生成文件: %s\n", outputPath)
}

func main() {
	const (
		inputPath  = "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/测量下单/复尺.pdf"         // 源文件
		outputPath = "/Users/zzy/Local/2026-04 北京丰台老旧小区改造/测量下单/复尺-去重-电话核实.pdf" // 处理后的文件
	)

	var aaa = `
74	5-3-602
75	1-6-301
76	1-7-602
77	1-8-501
78	1-10-402
79	1-11-201
80	1-11-602
81	1-4-502
82	2-6-101
83	12-7-502
84	12-3-102
85	12-5-202
86	12-7-202
87	12-8-501
88	12-8-401
89	13-5-302
90	13-2-602
91	13-2-502
92	13-5-501
93	13-6-601
94	13-7-502
95	13-2-102
96	13-5-302
97	13-2-402
98	14-3-201
99	14-4-302
100	14-5-502
101	14-6-502
102	16-1-302
103	16-5-502
104	17-6-502
105	3-2-401
106	10-5-202
107	15-5-201
108	4-2-601
109	3-1-201
110	3-1-101
111	3-1-201
112	3-5-101
113	3-6-601
114	3-1-401
115	5-1-304
116	6-5-202
117	6-6-502
118	6-5-301
119	6-3-301
120	11-3-502
121	11-6-302
122	11-6-302
123	11-5-502
124	1-16-603
125	1-20-502
126	1-14-102
127	1-13-102
128	1-13-302
129	1-12-501
130	1-4-302
131	1-4-301
132	1-6-602
133	1-6-302
`

	const deleted = "10,23,31,33,36,48,49,54,22,24,16,15,13,12,22"
	var baseNum = 73
	var pageMap = make(map[string]int)
	for _, line := range strings.Split(aaa, "\n") {
		if line = strings.TrimSpace(line); line == "" {
			continue
		}

		var fields = strings.Fields(line)
		if len(fields) != 2 {
			fmt.Println("fields length:", len(fields), fields, line)
			continue
		}

		fmt.Println("f1:", fields[0], "f2:", fields[1])
		i, err := strconv.Atoi(fields[0])
		if err != nil {
			panic(err)
		}

		pageMap[fields[1]] = i - baseNum
	}

	// 格式支持：单一数字 "2"、范围 "3-5"、或者从某页到最后 "n-"
	var pages = strings.Split(deleted, ",")

	fmt.Println("重复项删除数:", len(pages))

	const deletedNum = "1-7-602,1-8-501,1-11-201,1-11-602,1-4-502,2-6-101,12-3-102,14-4-302,14-5-502,16-1-302,16-5-502,15-5-201,4-2-601,3-1-101,3-6-601,3-1-401,6-5-301,1-16-603,1-13-302,1-12-501,1-4-301,1-6-602,1-6-302"

	// TODO 这户待定，暂时删除
	pages2 := strings.Split("13-5-501,"+deletedNum, ",")

	fmt.Println("电话核实删核数:", len(pages2))

	for _, num := range pages2 {
		page, exists := pageMap[num]
		if !exists {
			panic("不存在")
		}

		pages = append(pages, strconv.Itoa(page))
	}

	pages = DedupUsingOfficialLib(pages)

	fmt.Println("删除数量:", len(pages))
	fmt.Println("删除页面:", pages)

	err := RemovePdfPages(inputPath, outputPath, pages)
	if err != nil {
		log.Fatalf("错误: %v", err)
	}

	fmt.Printf("处理完成！生成文件: %s\n", outputPath)
}
