package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
)

// ConvertDWGToDXFNative 通过本地安装的 AutoCAD Core Console 原生转换图纸
// acCorePath: accoreconsole.exe 的绝对路径，例如 "C:\Program Files\Autodesk\AutoCAD 2024\accoreconsole.exe"
// inputDWG: 输入的 DWG 文件绝对路径
// outputDXF: 输出的 DXF 文件绝对路径
func ConvertDWGToDXFNative(acCorePath, inputDWG, outputDXF string) error {
	// 1. 路径校验与清理
	absInput, err := filepath.Abs(inputDWG)
	if err != nil {
		return fmt.Errorf("解析输入路径失败: %v", err)
	}
	absOutput, err := filepath.Abs(outputDXF)
	if err != nil {
		return fmt.Errorf("解析输出路径失败: %v", err)
	}

	if _, err := os.Stat(absInput); os.IsNotExist(err) {
		return fmt.Errorf("DWG文件不存在: %s", absInput)
	}

	// 如果输出文件已存在，先删除，防止 AutoCAD 遇到同名文件弹窗确认阻塞进程
	if _, err := os.Stat(absOutput); err == nil {
		os.Remove(absOutput)
	}

	// 2. 动态生成 AutoCAD Script (.scr) 批处理文件
	// FILEDIA 0: 禁用所有文件对话框，强制使用命令行交互
	// _DXFOUT: 执行导出 DXF 命令
	// 空行: 代表在 AutoCAD 命令行中按下 Enter 键（接受默认的 16 位精度）
	// _QUIT 和 Y: 退出且不保存当前 DWG 的修改
	scriptContent := fmt.Sprintf(`FILEDIA
0
_DXFOUT
"%s"

_QUIT
Y
`, absOutput)

	// 将脚本写入临时文件
	scriptFile, err := os.CreateTemp("", "autocad_convert_*.scr")
	if err != nil {
		return fmt.Errorf("创建临时脚本文件失败: %v", err)
	}
	scriptPath := scriptFile.Name()
	defer os.Remove(scriptPath) // 执行完毕后清理临时文件

	if _, err := scriptFile.WriteString(scriptContent); err != nil {
		scriptFile.Close()
		return fmt.Errorf("写入脚本文件失败: %v", err)
	}
	scriptFile.Close()

	// 3. 组装并调用 accoreconsole.exe
	// 参数 /i 指定输入图纸，/s 指定要执行的 script 脚本，/l 指定语言包(可选)
	cmd := exec.Command(acCorePath, "/i", absInput, "/s", scriptPath)

	// 捕获控制台输出，便于排查 AutoCAD 内部报错（如由于文件损坏导致的转换失败）
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("AutoCAD 进程执行失败: %v\n控制台输出: %s", err, string(output))
	}

	// 4. 验证结果
	if _, err := os.Stat(absOutput); os.IsNotExist(err) {
		return fmt.Errorf("命令执行完毕，但未生成 DXF 文件。可能是 AutoCAD 内部错误，控制台输出: %s", string(output))
	}

	return nil
}

func main() {
	// 示例调用 (请根据实际环境替换 AutoCAD 安装路径)
	autoCadPath := `C:\Program Files\Autodesk\AutoCAD 2024\accoreconsole.exe`
	inputPath := `C:\temp\sample_window_layout.dwg`
	outputPath := `C:\temp\sample_window_layout.dxf`

	fmt.Printf("开始调用原生 AutoCAD 进行转换...\n")
	err := ConvertDWGToDXFNative(autoCadPath, inputPath, outputPath)
	if err != nil {
		fmt.Printf("转换失败: %v\n", err)
		return
	}
	fmt.Printf("转换成功！文件已保存至: %s\n", outputPath)
}
