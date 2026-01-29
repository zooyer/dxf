package main

import (
	"bytes"
	"errors"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"runtime"
	"slices"
	"sort"
	"strings"

	"github.com/zooyer/dxf"
	"github.com/zooyer/dxf/core"
	"github.com/zooyer/dxf/entities"
	"github.com/zooyer/dxf/utils"
	"github.com/zooyer/golib/xmath"
	"github.com/zooyer/golib/xos"

	"github.com/ncruces/zenity"
	//"github.com/sqweek/dialog" // windows系统原生GUI
	//"github.com/progrium/darwinkit" // mac系统原生GUI
	//"github.com/gen2brain/dlgs" // 跨平台原生GUI
)

const winTitle = "CAD数据提取"

const (
	bzGap   = 30 // 标注连接线容错(不超过则认为挨着门窗周围)，验证过: 20
	winGap  = 20 // 窗户连接线容错(不超过则认为是同一个窗户)，验证过: 10
	epsilon = 1  // 浮点数对比精度误差(误差不超过则认为相同)，验证过: 1
)

type Window struct {
	Box     core.BBox             // 门窗范围(纯门窗面积)
	Area    core.BBox             // 覆盖范围(含标注面积)
	Label   []*entities.Dimension // 所有标注
	Widths  []float64             // 标注宽度
	Heights []float64             // 标注高度
}

func (w Window) Width() float64 {
	return w.Box.Max.X - w.Box.Min.X
}

func (w Window) Height() float64 {
	return w.Box.Max.Y - w.Box.Min.Y
}

func (w Window) MaxWidth() float64 {
	if len(w.Widths) < 1 {
		return 0
	}

	return slices.Max(w.Widths)
}

func (w Window) MaxHeight() float64 {
	if len(w.Heights) < 1 {
		return 0
	}

	return slices.Max(w.Heights)
}

func (w Window) VerifyWidth(epsilon float64) bool {
	if len(w.Widths) > 0 && xmath.Equal(w.Width(), slices.Max(w.Widths), epsilon) {
		return true
	}

	return false
}

func (w Window) VerifyHeight(epsilon float64) bool {
	if len(w.Heights) > 0 && xmath.Equal(w.Height(), slices.Max(w.Heights), epsilon) {
		return true
	}

	return false
}

type Form struct {
	doc  *dxf.Document         // 文档
	tka4 *entities.Insert      // A4纸，名称TKA4
	scs  []*entities.Insert    // 楼号信息，名称SC
	pjs  []core.BBox           // 楼号窗户，图层PJ
	bzs  []*entities.Dimension // 窗户标注，图层BZ
}

func (f Form) getAttr(key string) string {
	for _, sc := range f.scs {
		// 直接返回第一个
		//return utils.GetAttr(sc, key)

		if attr := utils.GetAttr(sc, key); attr != "" {
			return attr
		}
	}

	return ""
}

func (f Form) BBox() core.BBox {
	return utils.GetEntityBBoxWCS(f.doc, f.tka4)
}

func (f Form) Area() string {
	return f.getAttr("面积")
}

func (f Form) Amount() string {
	return f.getAttr("金额")
}

func (f Form) Serial() string {
	return f.getAttr("序号")
}

func (f Form) Building() string {
	return f.getAttr("楼号")
}

func (f Form) Windows() (windows []Window) {
	// 合并散线为矩形
	var boxes = utils.MergeBoxes(f.pjs, winGap)

	// 排序窗户 (从上到下)
	sort.Slice(boxes, func(i, j int) bool {
		if math.Abs(boxes[i].Max.Y-boxes[j].Max.Y) > 500 {
			return boxes[i].Max.Y > boxes[j].Max.Y
		}
		return boxes[i].Min.X < boxes[j].Min.X
	})

	for _, box := range boxes {
		var (
			area  = box                 // 扩展范围
			alls  = f.bzs               // 所有标注
			curr  []*entities.Dimension // 当前标注
			nears []*entities.Dimension // 附近标注
		)

		for {
			if alls, curr, area = getBZ(f.doc, alls, area, bzGap); len(curr) == 0 {
				break
			}

			// 打印每次扩展范围
			// TODO debug
			//fmt.Printf("RECTANG %f,%f %f,%f\n", wr.Min.X, wr.Min.Y, wr.Max.X, wr.Max.Y)

			nears = append(nears, curr...)
		}

		var widths, heights []float64

		for _, near := range nears {
			// 标准化角度到 0-360
			//var angle = math.Mod(near.Angle, 360)
			//if angle < 0 {
			//	angle += 360
			//}
			//
			//isW := angle < 45 || angle > 315 || (angle > 135 && angle < 225)

			var value = utils.GetDimValue(f.doc, near)

			switch int(near.Angle) {
			case 0, 180:
				widths = append(widths, value)
			case 90, 270:
				heights = append(heights, value)
			}
		}

		windows = append(windows, Window{
			Box:     box,
			Area:    area,
			Label:   nears,
			Widths:  widths,
			Heights: heights,
		})
	}

	return
}

// getBox 查找当前及子结构中所有在图层中的实体组件(递归)
func getBox(doc *dxf.Document, layer string, entity entities.Entity, parent *entities.Insert) (boxes []core.BBox) {
	if entity == nil {
		return
	}

	// 收集 PJ 层线条
	if entity.Layer() == layer {
		if parent == nil {
			boxes = append(boxes, entity.BBox())
		} else {
			boxes = append(boxes, utils.TransformBBox(entity.BBox(), parent))
		}
	}

	insert, ok := entity.(*entities.Insert)
	if !ok || doc == nil {
		return
	}

	block, exists := doc.Blocks[insert.BlockName]
	if !exists {
		return
	}

	if parent != nil {
		insert = utils.CombineInserts(parent, insert)
	}

	for _, sub := range block.Entities {
		for _, box := range getBox(doc, layer, sub, insert) {
			boxes = append(boxes, box)
		}
	}

	return
}

// getBZ 寻找与当前 box 邻近的标注
// 返回：未被匹配的标注(rest)、本次匹配到的标注(near)、扩充后的新盒子(newBox)
func getBZ(doc *dxf.Document, bzs []*entities.Dimension, box core.BBox, gap float64) (rest, near []*entities.Dimension, newBox core.BBox) {
	newBox = box // 初始继承旧盒子

	for _, bz := range bzs {
		// 只要转角标注
		if bz.DimType != 0 {
			rest = append(rest, bz)
			continue
		}

		var exe = 0.0

		if style, ok := doc.DimStyles[bz.StyleName]; ok {
			exe = style.ExLimit * style.Scale
		}

		// 1. 精度判定：检查被测量的两个端点 (13 和 14) 是否挨着窗户
		// 使用很小的 gap (比如 10-50) 就能精准匹配
		startIn := !utils.IsSeparate(box, bz.BBox2(exe), gap)
		endIn := !utils.IsSeparate(box, bz.BBox2(exe), gap)

		if startIn || endIn {
			// 匹配成功
			near = append(near, bz)

			b := bz.BBox2(exe)

			// 打印标注范围
			// TODO debug
			//fmt.Printf("BZ [%.0f] RECTANG %f,%f %f,%f\n", GetBZValue(doc, bz), b.Min.X, b.Min.Y, b.Max.X, b.Max.Y)

			// 2. 盒子扩充：按照最远的点（10, 11, 13, 14）补全成最大矩形
			// 这样下一轮迭代就能通过“标注线”抓到更外圈的“总尺寸”标注
			points := []core.Point{b.Min, b.Max}
			//points := []core.Point{bz.TextMidPoint, bz.DefPoint, bz.MeasureStart, bz.MeasureEnd}
			for _, p := range points {
				if p.X < newBox.Min.X {
					newBox.Min.X = p.X
				}
				if p.Y < newBox.Min.Y {
					newBox.Min.Y = p.Y
				}
				if p.X > newBox.Max.X {
					newBox.Max.X = p.X
				}
				if p.Y > newBox.Max.Y {
					newBox.Max.Y = p.Y
				}
			}
		} else {
			// 没匹配上的放回池子
			rest = append(rest, bz)
		}
	}
	return
}

// 渲染对错符号
func renderBool(b bool) string {
	if b {
		return "✅"
	}

	return "❌"
}

// 校验表格格式正确性，判断每行","数量是否一致
func checkCSV(filename, header string) {
	// 校验表格文件
	data, err := os.ReadFile(filename)
	if err != nil {
		fmt.Println("[表格检测] 打开文件失败:", err.Error())
		return
	}

	var count = strings.Count(header, ",")
	var lines [][2]int
	for i, line := range strings.Split(string(data), "\n") {
		if line == "" {
			continue
		}

		if curr := strings.Count(line, ","); curr != count {
			lines = append(lines, [2]int{i, curr})
		}
	}

	if len(lines) > 0 {
		fmt.Println("[表格检测] 表头分隔符数:", count, "检测到csv格式不正确，请检查！❌")
		for _, line := range lines {
			fmt.Println(fmt.Sprintf("  [LINE %d] 分隔符数量: %d", line[0]+1, line[1]))
		}
	}
}

// 统一对话框加前缀
func guiTitle(title string) zenity.Option {
	return zenity.Title(fmt.Sprintf("%s - %s", winTitle, title))
}

// 对话框报错，则打印到日志
func showMessage(fn func(text string, options ...zenity.Option) error, text string, options ...zenity.Option) {
	options = append(options, zenity.Modal(), zenity.NoCancel())
	if err := fn(text, options...); err != nil {
		fmt.Println("[GUI错误]", err.Error())
		var name = GetShortFuncName(fn)
		fmt.Println(fmt.Sprintf("[%s]: %s", name, text))
	}
}

// 计算进度百分比
func getPercent(value, total int) int {
	return value * 100 / total
}

// 设置进度条百分比
func setPercent(dialog zenity.ProgressDialog, title string, value, total int) {
	var percent = getPercent(value, total)

	_ = dialog.Value(percent)
	_ = dialog.Text(fmt.Sprintf("%s: %d%% (%d/%d)", title, percent, value, total))
}

// 选择文件对话框
func selectFile() (filename string, err error) {
	// 弹出文件选择框
	return zenity.SelectFile(
		guiTitle("请选择要导出的 CAD 图纸"),
		zenity.Modal(),
		zenity.FileFilters{
			{"CAD图纸", []string{"*.dxf"}, false},
			{"所有文件", []string{"*"}, false},
		},
	)
}

// 进度条处理
func handleProgress(title string, fn func(dialog zenity.ProgressDialog)) {
	dialog, err := zenity.Progress(
		guiTitle(title),
		zenity.EntryText("正在提取 DXF 图层数据，请稍候..."),
		zenity.Pulsate(),
		zenity.MaxValue(100),
		zenity.Modal(),    // 开启模态会阻塞，必须提前执行回调函数
		zenity.NoCancel(), // 如果你不希望用户中途关闭，可以加这一行
	)
	if err != nil {
		showMessage(zenity.Error, err.Error(), guiTitle("打开进度条错误"))
		os.Exit(4)
	}

	go func() {
		defer func() { _ = dialog.Close() }()

		fn(dialog)

		_ = dialog.Complete()
	}()

	<-dialog.Done()
}

// 打开文件，报错则退出
func openFile(filename string) *dxf.Document {
	doc, err := dxf.Open(filename)
	if err != nil {
		showMessage(zenity.Error, err.Error(), guiTitle("打开文件错误"))
		os.Exit(2)
	}

	return doc
}

// 写入文件，报错则退出
func writeFile(fn func(string, []byte, os.FileMode) error, filename string, data []byte, perm os.FileMode) {
	if err := fn(filename, data, perm); err != nil {
		showMessage(zenity.Error, err.Error(), guiTitle("写入文件错误"))
		os.Exit(5)
	}
}

// 获取输入文件: goland 测试、命令行参数、对话框选择
func getInput() string {
	// IDE 运行测试
	if strings.HasPrefix(filepath.Base(os.Args[0]), "___go_build_") {
		return "cmd/testdata/洞口图纸10.dxf"
	}

	// 入参传入文件名
	if len(os.Args) > 1 {
		return os.Args[1]
	}

	// 选择文件
	filename, err := selectFile()
	if err != nil {
		if errors.Is(err, zenity.ErrCanceled) {
			showMessage(zenity.Warning, "取消选择文件，即将退出程序！", guiTitle("选择文件提示"))
		} else {
			showMessage(zenity.Error, err.Error(), guiTitle("选择文件错误"))
		}

		os.Exit(1)
	}

	return filename
}

// 获取输出文件: 默认路径、自定义路径
func getOutput(input string) string {
	// 默认保存文件名
	var defaultOutput = strings.TrimSuffix(input, filepath.Ext(input)) + ".csv"

	if err := zenity.Question(
		fmt.Sprintf("保存到默认路径？\n默认路径: %s", defaultOutput),
		guiTitle("保存表格文件"),
		zenity.Modal(),
		zenity.NoCancel(),
		zenity.OKLabel("默认路径"),
		zenity.CancelLabel("自定义位置"),
	); err != nil {
		// 取消就是自定义位置
		if errors.Is(err, zenity.ErrCanceled) {
			var output string
			if output, err = zenity.SelectFileSave(
				guiTitle("保存到"),
				zenity.Modal(),
				zenity.Filename(defaultOutput), // 默认文件名
				zenity.FileFilters{
					{"表格 CSV", []string{"*.csv"}, false}, // 限制文件类型
				},
			); err == nil {
				if !strings.HasSuffix(output, ".csv") {
					output += ".csv"
				}

				return output
			}

			if errors.Is(err, zenity.ErrCanceled) {
				os.Exit(3)
			}
		}

		var text = fmt.Sprintf("将使用默认路径：%s\n错误信息：%s", defaultOutput, err.Error())
		showMessage(zenity.Error, text, guiTitle("保存文件错误"))
	}

	return defaultOutput
}

// 提取门窗确认单
func getForms(dialog zenity.ProgressDialog, doc *dxf.Document) []Form {
	if dialog == nil || doc == nil {
		return nil
	}

	var (
		pjs []core.BBox
		scs []*entities.Insert
		a4s []*entities.Insert
		bzs []*entities.Dimension
	)

	// 1. 提取所有组件、信息
	// 确认单A4(名称TKA4)、楼号信息(名称SC)、楼号门窗(图层PJ)、门窗标注(图层BZ)
	fmt.Printf("[开始处理]: %d 个实体组件...\n", len(doc.Entities))
	for i, entity := range doc.Entities {
		setPercent(dialog, "解析文档", i+1, len(doc.Entities))
		//time.Sleep(1 * time.Millisecond)

		switch e := entity.(type) {
		case *entities.Insert:
			switch e.BlockName {
			case "SC":
				scs = append(scs, e)
			case "TKA4":
				a4s = append(a4s, e)
			}
		case *entities.Dimension:
			bzs = append(bzs, e)
		}

		pjs = append(pjs, getBox(doc, "PJ", entity, nil)...)
	}

	// 2. 排序确认单A4 TKA4 (按 X 坐标，从左到右，符合人类阅读)
	sort.Slice(a4s, func(i, j int) bool {
		return a4s[i].InsertionPoint.X < a4s[j].InsertionPoint.X
	})

	// 3. 计算包含、相邻关系，划分组件、信息归属
	var forms = make([]Form, 0, len(a4s))
	fmt.Printf("[开始处理]: %d 个门窗数据...\n", len(a4s))
	for i, a4 := range a4s {
		setPercent(dialog, "计算组件", i+1, len(a4s))
		//time.Sleep(100 * time.Millisecond)

		var (
			box   = utils.GetEntityBBoxWCS(doc, a4)
			attrs []*entities.Insert
		)

		// 提取 SC 属性
		for _, sc := range scs {
			if utils.InBox(box, sc.InsertionPoint) {
				attrs = append(attrs, sc)
			}
		}

		// 提取图框内的 PJ 窗户散线
		var innerPJ []core.BBox
		for _, pb := range pjs {
			midX := (pb.Min.X + pb.Max.X) / 2
			midY := (pb.Min.Y + pb.Max.Y) / 2
			if utils.InBox(box, core.Point{X: midX, Y: midY, Z: 0}) {
				innerPJ = append(innerPJ, pb)
			}
		}

		forms = append(forms, Form{
			doc:  doc,
			tka4: a4,
			scs:  attrs,
			pjs:  innerPJ,
			bzs:  bzs,
		})
	}

	return forms
}

// 保存表格文件
func saveFile(dialog zenity.ProgressDialog, input, output string, forms []Form) {
	// 写入表头
	const (
		header    = "序号,楼号,宽度,高度,校验,测量宽度,测量高度,识别宽度,识别高度\n"
		emptyLine = ",,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,,"
	)

	writeFile(os.WriteFile, output, []byte(header), 0644)

	fmt.Println("写入文件:", output)
	fmt.Println()

	// 最后校验文件格式
	defer checkCSV(output, header)

	// 统计信息
	var (
		attrCount int     // 属性数量，理论应该和楼号数量一致
		diffCount int     // 测量和标注不一致的数量
		totalWin  int     // 所有楼号总窗户数
		totalArea float64 // 所有楼号总窗户面积
	)

	// 写入表格，打印输出
	for i, form := range forms {
		setPercent(dialog, "写入文件", i+1, len(forms))

		var (
			box  = form.BBox()
			wins = form.Windows()
		)

		// 打印信息
		fmt.Printf("[TKA4.%02d] | RECTANG %.2f,%.2f %.2f,%.2f | SC=%s\n",
			i+1, box.Min.X, box.Min.Y, box.Max.X, box.Max.Y, renderBool(len(form.scs) == 1),
		)
		for j, sc := range form.scs {
			var (
				area     = utils.GetAttr(sc, "面积")
				amount   = utils.GetAttr(sc, "金额")
				serial   = utils.GetAttr(sc, "序号")
				building = utils.GetAttr(sc, "楼号")
			)
			fmt.Printf("    [SC.%02d] | 序号:%s 金额:%s 面积:%s 楼号:%s\n",
				j+1, serial, amount, area, building,
			)
		}

		// 统计信息
		attrCount += len(form.scs)

		for j, w := range wins {
			// 打印信息
			var width, height = w.Width(), w.Height()
			fmt.Printf("    [窗户%d] | %.1f x %.1f | RECTANG %.2f,%.2f %.2f,%.2f\n",
				j+1, width, height, w.Box.Min.X, w.Box.Min.Y, w.Box.Max.X, w.Box.Max.Y,
			)
			// 识别宽高
			var verifyWidth, verifyHeight = w.VerifyWidth(epsilon), w.VerifyHeight(epsilon)
			fmt.Println("       |-- [识别宽度]:", w.Widths, renderBool(verifyWidth))
			fmt.Println("       |-- [识别高度]:", w.Heights, renderBool(verifyHeight))
			// 最终选区
			fmt.Printf("       |-- [最终范围]: RECTANG %.0f,%.0f %.0f,%.0f\n", w.Area.Min.X, w.Area.Min.Y, w.Area.Max.X, w.Area.Max.Y)

			// 统计信息
			totalWin++
			totalArea += width * height
			if !verifyWidth || !verifyHeight {
				diffCount++
			}

			var (
				valid    = renderBool(verifyWidth && verifyHeight)
				serial   string
				building string
			)

			if j == 0 {
				serial, building = form.Serial(), form.Building()
			}

			var line = fmt.Sprintf("%s,%s,%.0f,%.0f,%s,%.0f,%.0f,%s,%s\n",
				serial, building, w.MaxWidth(), w.MaxHeight(),
				valid, width, height,
				fmt.Sprint(w.Widths), fmt.Sprint(w.Heights),
			)

			writeFile(xos.AppendFile, output, []byte(line), 0644)
		}

		// 填充空行，至少7行
		for j := len(wins); j < 7; j++ {
			var line = emptyLine[:strings.Count(header, ",")] + "\n"

			writeFile(xos.AppendFile, output, []byte(line), 0644)
		}
	}

	// 写入统计信息
	var buf bytes.Buffer
	buf.WriteString("统计信息,总楼号数,总门窗数,总面积,A4页数,误差数,文件名,,\n")
	buf.WriteString(fmt.Sprintf(",%d,%d,%.6f,%d,%d,%s,,\n", attrCount, totalWin, totalArea/1000000, len(forms), diffCount, filepath.Base(input)))
	writeFile(xos.AppendFile, output, buf.Bytes(), 0644)

	fmt.Println()
	fmt.Println("[处理完成] 数据已保存至:", output, renderBool(true))
	fmt.Println("[共识别出]:")
	fmt.Println("    [楼号数]:", attrCount, "[A4页数]:", len(forms), renderBool(attrCount == len(forms)))
	fmt.Println("    [门窗数]:", fmt.Sprintf("%d (%d%s)", totalWin, diffCount, "个窗户测量与标注不一致"), renderBool(diffCount == 0))
	fmt.Println("    [总面积]:", fmt.Sprintf("%.6f (%.6f)", totalArea/1000000, totalArea))
	fmt.Println()
}

// GetFunctionName 获取函数全程，含路径
func GetFunctionName(fn any) string {
	// 获取函数的指针地址
	var pc = reflect.ValueOf(fn).Pointer()

	// 查找函数信息
	return runtime.FuncForPC(pc).Name()
}

// GetShortFuncName 只返回 "包名.函数名"
func GetShortFuncName(fn any) string {
	var fullPath = GetFunctionName(fn)

	// 找到最后一个斜杠的位置
	lastSlash := strings.LastIndex(fullPath, "/")
	if lastSlash == -1 {
		return fullPath // 如果没有斜杠，说明已经是 "包名.函数名" 格式
	}

	// 截取斜杠之后的部分
	return fullPath[lastSlash+1:]
}

// RevealFile 打开资源管理器，并选中该文件
func RevealFile(filename string) error {
	var cmd *exec.Cmd

	switch runtime.GOOS {
	case "windows":
		// Windows: 使用 explorer /select,
		cmd = exec.Command("explorer", "/select,", filename)
	case "darwin":
		// macOS: 使用 open -R
		cmd = exec.Command("open", "-R", filename)
	default:
		// Linux: 通常使用 dbus 发送信号，或者简单打开文件夹
		// 这里简单演示打开文件夹
		// cmd = exec.Command("xdg-open", filepath.Dir(filePath))
		return nil
	}

	return cmd.Run()
}

func main() {
	var (
		forms    []Form
		input    = getInput()      // 获取输入文件
		document = openFile(input) // 打开输入文件
	)

	// 提取数据
	handleProgress("提取数据", func(dialog zenity.ProgressDialog) {
		forms = getForms(dialog, document)
	})

	// 获取输出文件
	var output = getOutput(input)

	// 保存文件
	handleProgress("保存文件", func(dialog zenity.ProgressDialog) {
		saveFile(dialog, input, output, forms)
	})

	showMessage(zenity.Info, "数据导出成功！", guiTitle("导出提示"))

	_ = RevealFile(output)
}
