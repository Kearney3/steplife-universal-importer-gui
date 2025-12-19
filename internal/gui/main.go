package gui

import (
	"fmt"
	"image/color"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	consts "steplife-universal-importer/internal/const"
	"steplife-universal-importer/internal/model"
	"steplife-universal-importer/internal/server"
	"steplife-universal-importer/internal/utils/logx"
	timeUtils "steplife-universal-importer/internal/utils/time"

	"fyne.io/fyne/v2"
	"fyne.io/fyne/v2/app"
	"fyne.io/fyne/v2/container"
	"fyne.io/fyne/v2/dialog"
	"fyne.io/fyne/v2/layout"
	"fyne.io/fyne/v2/theme"
	"fyne.io/fyne/v2/widget"
	"github.com/pkg/errors"
	"gopkg.in/ini.v1"
)

// hiddenFileFilter 自定义文件过滤器，隐藏以点开头的文件和文件夹
type hiddenFileFilter struct {
	extensions []string
}

func (f *hiddenFileFilter) Matches(uri fyne.URI) bool {
	path := uri.Path()
	baseName := filepath.Base(path)

	// 隐藏以点开头的文件和文件夹
	if strings.HasPrefix(baseName, ".") {
		return false
	}

	// 如果指定了扩展名，检查文件扩展名
	if len(f.extensions) > 0 {
		ext := strings.ToLower(filepath.Ext(path))
		for _, allowedExt := range f.extensions {
			if ext == strings.ToLower(allowedExt) {
				return true
			}
		}
		return false
	}

	return true
}

func (f *hiddenFileFilter) Extensions() []string {
	return f.extensions
}

type GUI struct {
	app             fyne.App
	window          fyne.Window
	config          model.Config
	sourceDir       string
	outputDir       string
	createOutputDir bool // 是否创建output文件夹
	isFileMode      bool // 是否为文件选择模式（true=文件，false=文件夹）
	showLog         bool // 是否显示处理日志
	statusLabel     *widget.Label
	progressBar     *widget.ProgressBar
	logText         *widget.Entry
	logScroll       *container.Scroll
	fontRegular     fyne.Resource
	customTheme     fyne.Theme
}

// loadCustomFont 加载自定义字体
func (g *GUI) loadCustomFont() error {
	fontPath := "./resource/MiSans-Regular.otf"

	// 检查字体文件是否存在
	if _, err := os.Stat(fontPath); os.IsNotExist(err) {
		logx.InfoF("字体文件不存在 %s，使用系统默认字体: %v", fontPath, err)
		return err
	}

	fontData, err := ioutil.ReadFile(fontPath)
	if err != nil {
		logx.InfoF("无法读取字体文件 %s，使用系统默认字体: %v", fontPath, err)
		return err
	}

	if len(fontData) == 0 {
		logx.InfoF("字体文件为空 %s，使用系统默认字体", fontPath)
		return fmt.Errorf("font file is empty")
	}

	g.fontRegular = fyne.NewStaticResource("MiSans-Regular", fontData)
	logx.InfoF("成功加载自定义字体: MiSans-Regular.otf (大小: %d bytes)", len(fontData))
	return nil
}

// customTheme 实现自定义主题
type customTheme struct {
	fontRegular fyne.Resource
}

func (c *customTheme) Font(style fyne.TextStyle) fyne.Resource {
	if c.fontRegular != nil {
		return c.fontRegular
	}
	return theme.DefaultTheme().Font(style)
}

func (c *customTheme) Color(name fyne.ThemeColorName, variant fyne.ThemeVariant) color.Color {
	return theme.DefaultTheme().Color(name, variant)
}

func (c *customTheme) Icon(name fyne.ThemeIconName) fyne.Resource {
	return theme.DefaultTheme().Icon(name)
}

func (c *customTheme) Size(name fyne.ThemeSizeName) float32 {
	return theme.DefaultTheme().Size(name)
}

// NewGUI 创建GUI实例
func NewGUI() *GUI {
	gui := &GUI{
		app:             app.New(),
		createOutputDir: true,  // 默认创建output文件夹
		isFileMode:      false, // 默认文件夹模式
		showLog:         true,  // 默认显示日志
		config: model.Config{
			EnableInsertPointStrategy: 1,
			InsertPointDistance:       100,
			DefaultAltitude:           0.0,
			SpeedMode:                 "auto",
			ManualSpeed:               1.5,
			EnableBatchProcessing:     1,
		},
	}

	// 加载自定义字体
	gui.loadCustomFont()

	// 设置自定义主题
	if gui.fontRegular != nil {
		gui.customTheme = &customTheme{fontRegular: gui.fontRegular}
		gui.app.Settings().SetTheme(gui.customTheme)
	}

	gui.window = gui.app.NewWindow(fmt.Sprintf("一生足迹数据导入器 v%s", consts.Version))
	gui.window.SetMaster()

	return gui
}

// Run 运行GUI
func (g *GUI) Run() {
	g.loadConfig()
	g.createMainWindow()
	g.window.ShowAndRun()
}

// loadConfig 加载配置文件
func (g *GUI) loadConfig() {
	cfg, err := ini.Load("config.ini")
	if err != nil {
		logx.Info("配置文件不存在，使用默认配置")
		return
	}

	err = cfg.MapTo(&g.config)
	if err != nil {
		logx.ErrorF("加载配置失败：%v", err)
		return
	}

	// 处理时间戳
	if g.config.PathStartTime != "" {
		g.config.PathStartTimestamp, _ = timeUtils.ToTimestamp(g.config.PathStartTime)
	}
	if g.config.PathEndTime != "" {
		g.config.PathEndTimestamp, _ = timeUtils.ToTimestamp(g.config.PathEndTime)
	}
}

// saveConfig 保存配置到文件
func (g *GUI) saveConfig() error {
	cfg := ini.Empty()

	section := cfg.Section("")

	section.Key("enableInsertPointStrategy").SetValue(fmt.Sprintf("%d", g.config.EnableInsertPointStrategy))
	section.Key("insertPointDistance").SetValue(fmt.Sprintf("%d", g.config.InsertPointDistance))
	section.Key("pathStartTime").SetValue(g.config.PathStartTime)
	section.Key("pathEndTime").SetValue(g.config.PathEndTime)
	section.Key("defaultAltitude").SetValue(fmt.Sprintf("%.2f", g.config.DefaultAltitude))
	section.Key("speedMode").SetValue(g.config.SpeedMode)
	section.Key("manualSpeed").SetValue(fmt.Sprintf("%.2f", g.config.ManualSpeed))
	section.Key("enableBatchProcessing").SetValue(fmt.Sprintf("%d", g.config.EnableBatchProcessing))

	return cfg.SaveTo("config.ini")
}

// createMainWindow 创建主界面
func (g *GUI) createMainWindow() {
	g.window.SetContent(g.createMainLayout())
	// 设置最小窗口大小，而不是固定大小
	g.window.SetFixedSize(false)
	g.window.Resize(fyne.NewSize(900, 1000))
}

// createMainLayout 创建主界面布局
func (g *GUI) createMainLayout() fyne.CanvasObject {
	// 文件选择区域
	sourceDirLabel := widget.NewLabel("源文件:")
	sourceDirEntry := widget.NewEntry()
	sourceDirEntry.SetPlaceHolder("选择轨迹文件或包含轨迹文件的目录")
	sourceDirEntry.Resize(fyne.NewSize(350, sourceDirEntry.MinSize().Height))
	sourceDirButton := widget.NewButton("选择文件/目录", func() {
		g.selectSource(sourceDirEntry)
	})

	// 文件/文件夹选择模式
	modeSelect := widget.NewSelect([]string{"文件夹模式", "单文件模式"}, nil)

	outputDirLabel := widget.NewLabel("输出目录:")
	outputDirEntry := widget.NewEntry()
	outputDirEntry.SetPlaceHolder("选择输出CSV文件的目录")
	outputDirEntry.Resize(fyne.NewSize(300, outputDirEntry.MinSize().Height))

	// 创建output文件夹开关
	createOutputDirCheck := widget.NewCheck("创建 output 文件夹", func(checked bool) {
		g.createOutputDir = checked
		if checked && sourceDirEntry.Text != "" {
			// 自动设置输出目录
			g.updateOutputDir(sourceDirEntry.Text, outputDirEntry)
		}
	})
	createOutputDirCheck.SetChecked(true) // 默认开启

	outputDirButton := widget.NewButton("选择目录", func() {
		g.selectOutputDirectory(outputDirEntry)
	})

	// 参数设置区域
	paramsCard := widget.NewCard("参数设置", "",
		container.NewVBox(
			g.createTimeSettings(),
			widget.NewSeparator(),
			g.createAltitudeSettings(),
			widget.NewSeparator(),
			g.createSpeedSettings(),
			widget.NewSeparator(),
			g.createInsertPointSettings(),
		),
	)

	// 状态显示区域
	g.statusLabel = widget.NewLabel("就绪")
	g.progressBar = widget.NewProgressBar()
	g.progressBar.Hide()

	statusCard := widget.NewCard("处理状态", "",
		container.NewVBox(
			g.statusLabel,
			g.progressBar,
		),
	)

	// 初始化日志显示区域（只在第一次调用时）
	if g.logText == nil {
		g.logText = widget.NewMultiLineEntry()
		g.logText.Disable()                           // 设置为只读（禁用编辑）
		g.logText.Wrapping = fyne.TextWrapWord        // 启用自动换行
		g.logScroll = container.NewVScroll(g.logText) // 使用垂直滚动，优化滚动体验
		// 设置最小尺寸，宽度设为0以允许随窗口宽度变化
		g.logScroll.SetMinSize(fyne.NewSize(0, 100))
	}

	// 创建日志卡片（始终显示）
	logCard := widget.NewCard("处理日志", "",
		g.logScroll,
	)

	// 文件选择区域布局
	sourceDirContainer := container.NewBorder(
		nil, nil, nil, sourceDirButton, sourceDirEntry,
	)
	sourceDirRow := container.NewVBox(
		container.NewHBox(sourceDirLabel, modeSelect),
		sourceDirContainer,
	)

	outputDirContainer := container.NewBorder(
		nil, nil, nil, outputDirButton, outputDirEntry,
	)
	outputDirRow := container.NewVBox(
		container.NewHBox(outputDirLabel, createOutputDirCheck),
		outputDirContainer,
	)

	// 状态和日志区域 - 日志区域会自动扩展填充可用空间
	statusAndLogArea := container.NewVBox(
		statusCard,
		widget.NewSeparator(),
		logCard,
	)

	fileSelectionArea := container.NewVBox(
		sourceDirRow,
		outputDirRow,
	)

	// 可滚动的主内容区域（不包含按钮）
	scrollableContent := container.NewVBox(
		fileSelectionArea,
		paramsCard,
		statusAndLogArea,
	)

	// 添加滚动容器 - 使用垂直滚动，优化滚动体验
	scrollContainer := container.NewVScroll(scrollableContent)
	scrollContainer.SetMinSize(fyne.NewSize(800, 600)) // 设置最小滚动区域大小

	// 操作按钮（始终可见，位于底部）
	processButton := widget.NewButtonWithIcon("开始处理", theme.MediaPlayIcon(), func() {
		g.startProcessing(sourceDirEntry.Text, outputDirEntry.Text)
	})
	processButton.Importance = widget.HighImportance

	saveConfigButton := widget.NewButtonWithIcon("保存配置", theme.DocumentSaveIcon(), func() {
		g.saveConfigDialog()
	})

	buttons := container.NewHBox(
		layout.NewSpacer(),
		processButton,
		saveConfigButton,
		layout.NewSpacer(),
	)

	// 按钮容器，紧凑布局
	buttonContainer := container.NewVBox(
		widget.NewSeparator(),
		container.NewPadded(buttons), // 给按钮添加内边距
	)
	// 设置最小高度，确保按钮区域可见
	buttonContainer.Resize(fyne.NewSize(buttonContainer.MinSize().Width, 50))

	// 设置模式选择的回调函数（现在所有变量都已定义）
	modeSelect.OnChanged = func(selected string) {
		previousMode := g.isFileMode
		g.isFileMode = (selected == "单文件模式")

		// 检查当前选择是否与新模式兼容
		if g.sourceDir != "" {
			fileInfo, err := os.Stat(g.sourceDir)
			if err == nil {
				isDir := fileInfo.IsDir()

				// 如果从文件夹模式切换到单文件模式，且当前选择了文件夹
				if !previousMode && g.isFileMode && isDir {
					g.addLog("模式切换：检测到当前选择了文件夹，但单文件模式需要选择具体文件")
					dialog.ShowInformation("模式切换提示",
						"您当前选择了文件夹，但单文件模式需要选择具体的轨迹文件。\n请重新选择文件。",
						g.window)
					// 清除当前选择
					sourceDirEntry.SetText("")
					g.sourceDir = ""
					outputDirEntry.SetText("")
					g.outputDir = ""
				} else if previousMode && !g.isFileMode && !isDir {
					// 从单文件模式切换到文件夹模式，且当前选择了文件
					g.addLog("模式切换：从单文件模式切换到文件夹模式")
				}
			}
		}

		// 更新UI显示
		if g.isFileMode {
			sourceDirEntry.SetPlaceHolder("选择轨迹文件")
			sourceDirButton.SetText("选择文件")
		} else {
			sourceDirEntry.SetPlaceHolder("选择包含轨迹文件的目录")
			sourceDirButton.SetText("选择目录")
		}
	}
	modeSelect.SetSelected("文件夹模式") // 默认文件夹模式

	// 整体布局：滚动内容在上，按钮固定在底部
	return container.NewBorder(
		nil,             // 顶部无内容
		buttonContainer, // 按钮区域固定在底部，有固定高度
		nil,             // 左侧无内容
		nil,             // 右侧无内容
		scrollContainer, // 主要内容区域可滚动
	)
}

// createTimeSettings 创建时间设置组件
func (g *GUI) createTimeSettings() fyne.CanvasObject {
	// 开始时间输入框和选择按钮
	startTimeEntry := widget.NewEntry()
	startTimeEntry.SetPlaceHolder("格式: 2024-01-01 08:00:00 (默认为系统时间)")
	startTimeEntry.SetText(g.config.PathStartTime)
	startTimeEntry.Resize(fyne.NewSize(250, startTimeEntry.MinSize().Height))
	startTimeEntry.OnChanged = func(text string) {
		g.config.PathStartTime = text
	}

	startTimeButton := widget.NewButton("选择时间", func() {
		g.showDateTimePicker(startTimeEntry, "选择开始时间")
	})

	startTimeContainer := container.NewBorder(nil, nil, nil, startTimeButton, startTimeEntry)

	// 结束时间输入框和选择按钮
	endTimeEntry := widget.NewEntry()
	endTimeEntry.SetPlaceHolder("格式: 2024-01-01 18:00:00 (可选)")
	endTimeEntry.SetText(g.config.PathEndTime)
	endTimeEntry.Resize(fyne.NewSize(250, endTimeEntry.MinSize().Height))
	endTimeEntry.OnChanged = func(text string) {
		g.config.PathEndTime = text
	}

	endTimeButton := widget.NewButton("选择时间", func() {
		g.showDateTimePicker(endTimeEntry, "选择结束时间")
	})

	endTimeContainer := container.NewBorder(nil, nil, nil, endTimeButton, endTimeEntry)

	return container.New(layout.NewFormLayout(),
		widget.NewLabel("开始时间:"), startTimeContainer,
		widget.NewLabel("结束时间:"), endTimeContainer,
	)
}

// createAltitudeSettings 创建海拔设置组件
func (g *GUI) createAltitudeSettings() fyne.CanvasObject {
	altitudeEntry := widget.NewEntry()
	altitudeEntry.SetPlaceHolder("默认海拔高度(米)")
	altitudeEntry.SetText(fmt.Sprintf("%.2f", g.config.DefaultAltitude))
	altitudeEntry.Resize(fyne.NewSize(200, altitudeEntry.MinSize().Height))
	altitudeEntry.OnChanged = func(text string) {
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			g.config.DefaultAltitude = val
		}
	}

	return container.New(layout.NewFormLayout(),
		widget.NewLabel("默认海拔(米):"), altitudeEntry,
	)
}

// createSpeedSettings 创建速度设置组件
func (g *GUI) createSpeedSettings() fyne.CanvasObject {
	speedModeSelect := widget.NewSelect([]string{"自动计算", "手动指定"}, func(selected string) {
		if selected == "自动计算" {
			g.config.SpeedMode = "auto"
		} else {
			g.config.SpeedMode = "manual"
		}
	})

	if g.config.SpeedMode == "auto" {
		speedModeSelect.SetSelected("自动计算")
	} else {
		speedModeSelect.SetSelected("手动指定")
	}

	speedEntry := widget.NewEntry()
	speedEntry.SetPlaceHolder("手动指定速度(m/s)")
	speedEntry.SetText(fmt.Sprintf("%.2f", g.config.ManualSpeed))
	speedEntry.Resize(fyne.NewSize(200, speedEntry.MinSize().Height))
	speedEntry.OnChanged = func(text string) {
		if val, err := strconv.ParseFloat(text, 64); err == nil {
			g.config.ManualSpeed = val
		}
	}

	return container.NewVBox(
		widget.NewLabel("速度设置:"),
		speedModeSelect,
		container.New(layout.NewFormLayout(),
			widget.NewLabel("指定速度(m/s):"), speedEntry,
		),
	)
}

// createInsertPointSettings 创建插点设置组件
func (g *GUI) createInsertPointSettings() fyne.CanvasObject {
	enableInsertCheck := widget.NewCheck("启用轨迹插点", func(checked bool) {
		if checked {
			g.config.EnableInsertPointStrategy = 1
		} else {
			g.config.EnableInsertPointStrategy = 0
		}
	})
	enableInsertCheck.SetChecked(g.config.EnableInsertPointStrategy == 1)

	distanceEntry := widget.NewEntry()
	distanceEntry.SetPlaceHolder("插点距离阈值(米)")
	distanceEntry.SetText(fmt.Sprintf("%d", g.config.InsertPointDistance))
	distanceEntry.Resize(fyne.NewSize(200, distanceEntry.MinSize().Height))
	distanceEntry.OnChanged = func(text string) {
		if val, err := strconv.Atoi(text); err == nil && val >= consts.MinInsertPointDistance {
			g.config.InsertPointDistance = val
		}
	}

	return container.NewVBox(
		enableInsertCheck,
		container.New(layout.NewFormLayout(),
			widget.NewLabel("插点距离(米):"), distanceEntry,
		),
	)
}

// selectSource 选择源文件或目录
func (g *GUI) selectSource(entry *widget.Entry) {
	if g.isFileMode {
		// 单文件模式
		fileDialog := dialog.NewFileOpen(func(reader fyne.URIReadCloser, err error) {
			if err != nil || reader == nil {
				return
			}
			uri := reader.URI()
			path := uri.Path()
			g.addLog(fmt.Sprintf("选择文件 - URI: %s, Path: %s", uri.String(), path))

			// 验证路径是否存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				g.addLog(fmt.Sprintf("警告：文件路径不存在: %s", path))
			} else {
				g.addLog(fmt.Sprintf("文件路径验证通过: %s", path))
			}

			entry.SetText(path)
			g.sourceDir = path
			// 自动更新输出目录
			g.updateOutputDir(path, nil)
		}, g.window)
		// 使用自定义过滤器，隐藏以点开头的文件
		fileFilter := &hiddenFileFilter{extensions: []string{".gpx", ".kml", ".ovjsn"}}
		fileDialog.SetFilter(fileFilter)
		fileDialog.Show()
	} else {
		// 文件夹模式
		folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
			if err != nil || uri == nil {
				return
			}
			path := uri.Path()

			// 检查是否为隐藏文件夹（以点开头）
			baseName := filepath.Base(path)
			if strings.HasPrefix(baseName, ".") {
				g.addLog(fmt.Sprintf("跳过隐藏文件夹: %s", path))
				dialog.ShowInformation("提示", "不能选择隐藏文件夹（以点开头的文件夹）", g.window)
				return
			}

			g.addLog(fmt.Sprintf("选择文件夹 - URI: %s, Path: %s", uri.String(), path))

			// 验证路径是否存在
			if _, err := os.Stat(path); os.IsNotExist(err) {
				g.addLog(fmt.Sprintf("警告：文件夹路径不存在: %s", path))
			} else {
				g.addLog(fmt.Sprintf("文件夹路径验证通过: %s", path))
			}

			entry.SetText(path)
			g.sourceDir = path
			// 自动更新输出目录
			g.updateOutputDir(path, nil)
		}, g.window)
		folderDialog.Show()
	}
}

// updateOutputDir 自动更新输出目录
func (g *GUI) updateOutputDir(sourcePath string, outputEntry *widget.Entry) {
	if sourcePath == "" {
		return
	}

	var outputPath string
	if g.createOutputDir {
		if g.isFileMode {
			// 单文件模式：在文件所在目录创建output文件夹
			sourcePath = filepath.Dir(sourcePath)
		}
		outputPath = filepath.Join(sourcePath, "output")
	} else {
		// 不创建output文件夹：直接使用源目录
		if g.isFileMode {
			outputPath = filepath.Dir(sourcePath)
		} else {
			outputPath = sourcePath
		}
	}

	if outputEntry != nil {
		outputEntry.SetText(outputPath)
	}
	g.outputDir = outputPath
}

// selectOutputDirectory 选择输出目录
func (g *GUI) selectOutputDirectory(entry *widget.Entry) {
	folderDialog := dialog.NewFolderOpen(func(uri fyne.ListableURI, err error) {
		if err != nil || uri == nil {
			return
		}
		path := uri.Path()

		// 检查是否为隐藏文件夹（以点开头）
		baseName := filepath.Base(path)
		if strings.HasPrefix(baseName, ".") {
			g.addLog(fmt.Sprintf("跳过隐藏文件夹: %s", path))
			dialog.ShowInformation("提示", "不能选择隐藏文件夹（以点开头的文件夹）", g.window)
			return
		}

		entry.SetText(path)
		g.outputDir = path
	}, g.window)
	folderDialog.Show()
}

// saveConfigDialog 保存配置对话框
func (g *GUI) saveConfigDialog() {
	if err := g.saveConfig(); err != nil {
		dialog.ShowError(errors.Wrap(err, "保存配置失败"), g.window)
		return
	}
	dialog.ShowInformation("成功", "配置已保存", g.window)
}

// startProcessing 开始处理文件
func (g *GUI) startProcessing(sourcePath, outputDir string) {
	if sourcePath == "" {
		if g.isFileMode {
			dialog.ShowError(errors.New("请选择源文件"), g.window)
		} else {
			dialog.ShowError(errors.New("请选择源文件目录"), g.window)
		}
		return
	}

	// 检查源路径是否存在
	g.addLog(fmt.Sprintf("验证源路径: %s", sourcePath))
	if _, err := os.Stat(sourcePath); os.IsNotExist(err) {
		g.addLog(fmt.Sprintf("路径不存在: %s", sourcePath))
		dialog.ShowError(fmt.Errorf("路径不存在: %s", sourcePath), g.window)
		return
	}
	g.addLog(fmt.Sprintf("源路径验证通过: %s", sourcePath))

	if outputDir == "" {
		if g.isFileMode {
			outputDir = filepath.Dir(sourcePath)
		} else {
			outputDir = sourcePath
		}
		if g.createOutputDir {
			outputDir = filepath.Join(outputDir, "output")
		}
	}

	g.sourceDir = sourcePath
	g.outputDir = outputDir

	// 验证和处理时间配置
	if g.config.PathStartTime != "" {
		timestamp, err := timeUtils.ToTimestamp(g.config.PathStartTime)
		if err != nil {
			dialog.ShowError(errors.Wrap(err, "开始时间格式错误"), g.window)
			return
		}
		g.config.PathStartTimestamp = timestamp
	} else {
		g.config.PathStartTimestamp = time.Now().Unix()
	}

	if g.config.PathEndTime != "" {
		timestamp, err := timeUtils.ToTimestamp(g.config.PathEndTime)
		if err != nil {
			dialog.ShowError(errors.Wrap(err, "结束时间格式错误"), g.window)
			return
		}
		g.config.PathEndTimestamp = timestamp
	}

	// 保存配置
	g.saveConfig()

	// 开始处理
	go g.processFiles()
}

// processFiles 处理文件
func (g *GUI) processFiles() {
	defer func() {
		// 确保在任何情况下都恢复GUI状态
		if r := recover(); r != nil {
			g.addLog(fmt.Sprintf("处理过程中发生严重错误: %v", r))
			g.showError(fmt.Sprintf("处理过程中发生严重错误: %v", r))
			g.progressBar.Hide()
			g.statusLabel.SetText("错误：处理失败")
		}
	}()

	g.progressBar.Show()
	g.statusLabel.SetText("正在扫描文件...")
	g.addLog("开始处理文件...")

	// 确保输出目录存在
	g.addLog("创建输出目录: " + g.outputDir)
	if err := os.MkdirAll(g.outputDir, 0755); err != nil {
		g.addLog("创建输出目录失败: " + err.Error())
		g.showError("创建输出目录失败: " + err.Error())
		return
	}
	g.addLog("输出目录创建成功: " + g.outputDir)

	// 验证输出目录是否可写
	testFile := filepath.Join(g.outputDir, ".test_write")
	if err := os.WriteFile(testFile, []byte("test"), 0644); err != nil {
		g.addLog("输出目录不可写: " + err.Error())
		g.showError("输出目录不可写，请选择其他目录")
		return
	}
	os.Remove(testFile) // 清理测试文件
	g.addLog("输出目录权限验证通过")

	g.progressBar.SetValue(0.1)

	var filePaths []string

	// 检查是文件还是目录
	fileInfo, err := os.Stat(g.sourceDir)
	if err != nil {
		g.showError("访问路径失败: " + err.Error())
		return
	}

	if fileInfo.IsDir() {
		// 文件夹模式：扫描目录
		g.addLog("扫描源目录: " + g.sourceDir)
		filePathMap, err := g.scanSourceDirectory()
		if err != nil {
			g.showError("扫描文件失败: " + err.Error())
			return
		}

		// 收集所有文件路径
		for _, paths := range filePathMap {
			filePaths = append(filePaths, paths...)
		}
	} else {
		// 单文件模式：直接处理单个文件
		g.addLog("检测到单文件模式，源路径: " + g.sourceDir)

		// 验证文件是否存在且可读
		if _, err := os.Stat(g.sourceDir); os.IsNotExist(err) {
			g.showError("文件不存在: " + g.sourceDir)
			return
		}

		ext := strings.ToLower(filepath.Ext(g.sourceDir))
		g.addLog("文件扩展名: " + ext)

		if ext != ".gpx" && ext != ".kml" && ext != ".ovjsn" {
			g.showError("不支持的文件格式，仅支持 .gpx, .kml, .ovjsn 文件")
			return
		}
		filePaths = []string{g.sourceDir}
		g.addLog("准备处理单个文件: " + filepath.Base(g.sourceDir))
	}

	totalFiles := len(filePaths)
	g.addLog(fmt.Sprintf("找到 %d 个文件待处理", totalFiles))

	if totalFiles == 0 {
		g.showError("未找到支持的文件格式(.kml, .gpx, .ovjsn)")
		return
	}

	g.statusLabel.SetText(fmt.Sprintf("找到 %d 个文件，开始处理...", totalFiles))
	g.addLog("开始处理文件...")
	g.progressBar.SetValue(0.2)

	processed := 0
	g.addLog(fmt.Sprintf("开始处理 %d 个文件", totalFiles))

	for _, filePath := range filePaths {
		fileName := filepath.Base(filePath)
		g.statusLabel.SetText(fmt.Sprintf("处理文件: %s", fileName))
		g.addLog(fmt.Sprintf("正在处理文件: %s (路径: %s)", fileName, filePath))

		// 验证文件是否存在
		if _, err := os.Stat(filePath); os.IsNotExist(err) {
			g.addLog(fmt.Sprintf("文件不存在，跳过: %s", filePath))
			processed++
			continue
		}

		// 根据文件扩展名确定文件类型
		ext := strings.ToLower(filepath.Ext(filePath))
		var fileType string
		switch ext {
		case ".gpx", ".kml", ".ovjsn":
			fileType = consts.FileTypeCommon
			g.addLog(fmt.Sprintf("文件类型: %s", fileType))
		default:
			g.addLog(fmt.Sprintf("跳过不支持的文件类型 %s: %s", ext, fileName))
			processed++
			continue
		}

		// 生成输出路径
		outputPath := g.generateOutputPath(filePath)
		g.addLog(fmt.Sprintf("输出路径: %s", outputPath))

		err := g.processSingleFile(fileType, filePath)
		if err != nil {
			g.addLog(fmt.Sprintf("处理文件失败 %s: %s", fileName, err.Error()))
			g.showError(fmt.Sprintf("处理文件失败 %s: %s", fileName, err.Error()))
			continue
		}

		g.addLog(fmt.Sprintf("文件处理完成: %s", fileName))
		processed++
		g.progressBar.SetValue(0.2 + 0.8*float64(processed)/float64(totalFiles))
	}

	g.progressBar.SetValue(1.0)
	if processed > 0 {
		g.statusLabel.SetText(fmt.Sprintf("处理完成！成功处理 %d 个文件", processed))
		g.addLog(fmt.Sprintf("处理完成！成功处理 %d 个文件", processed))
	} else {
		g.statusLabel.SetText("处理完成，但没有成功处理任何文件")
		g.addLog("处理完成，但没有成功处理任何文件")
	}

	// 完成后隐藏进度条
	time.Sleep(2 * time.Second)
	g.progressBar.Hide()
	g.statusLabel.SetText("就绪")
}

// scanSourceDirectory 扫描源目录
func (g *GUI) scanSourceDirectory() (map[string][]string, error) {
	filePathMap := make(map[string][]string)

	err := filepath.Walk(g.sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		ext := strings.ToLower(filepath.Ext(path))
		switch ext {
		case ".kml", ".gpx", ".ovjsn":
			fileType := consts.FileTypeCommon
			filePathMap[fileType] = append(filePathMap[fileType], path)
		}

		return nil
	})

	return filePathMap, err
}

// processSingleFile 处理单个文件
func (g *GUI) processSingleFile(fileType, filePath string) error {
	// 为每个文件创建独立的配置副本
	config := g.config

	// 如果设置了结束时间，重新计算时间戳分配
	if config.PathEndTime != "" {
		// 这里需要先读取文件获取点数量，然后重新分配时间
		// 暂时使用简化逻辑，后续完善
	}

	csvFilePath := g.generateOutputPath(filePath)

	err := server.ProcessSingleFile(fileType, filePath, csvFilePath, config)
	if err != nil {
		return err
	}

	return nil
}

// generateOutputPath 生成输出文件路径
func (g *GUI) generateOutputPath(sourcePath string) string {
	baseName := strings.TrimSuffix(filepath.Base(sourcePath), filepath.Ext(sourcePath))
	outputPath := filepath.Join(g.outputDir, baseName+"_steplife.csv")
	return outputPath
}

// showDateTimePicker 显示日期时间选择器
func (g *GUI) showDateTimePicker(entry *widget.Entry, title string) {
	currentTime := time.Now()
	if entry.Text != "" {
		if parsedTime, err := time.Parse("2006-01-02 15:04:05", entry.Text); err == nil {
			currentTime = parsedTime
		}
	}

	// Fyne没有内置的日期时间选择器，使用自定义的数字输入框实现
	// 优点：跨平台一致性，不依赖系统组件
	yearEntry := g.createNumberEntry(currentTime.Year(), 1900, 2100)
	monthEntry := g.createNumberEntry(int(currentTime.Month()), 1, 12)
	dayEntry := g.createNumberEntry(currentTime.Day(), 1, 31)
	hourEntry := g.createNumberEntry(currentTime.Hour(), 0, 23)
	minuteEntry := g.createNumberEntry(currentTime.Minute(), 0, 59)
	secondEntry := g.createNumberEntry(currentTime.Second(), 0, 59)

	// 设置输入框宽度，便于输入
	yearEntry.Resize(fyne.NewSize(80, yearEntry.MinSize().Height))
	monthEntry.Resize(fyne.NewSize(60, monthEntry.MinSize().Height))
	dayEntry.Resize(fyne.NewSize(60, dayEntry.MinSize().Height))
	hourEntry.Resize(fyne.NewSize(60, hourEntry.MinSize().Height))
	minuteEntry.Resize(fyne.NewSize(60, minuteEntry.MinSize().Height))
	secondEntry.Resize(fyne.NewSize(60, secondEntry.MinSize().Height))

	// 快捷按钮
	nowButton := widget.NewButton("现在", func() {
		now := time.Now()
		yearEntry.SetText(strconv.Itoa(now.Year()))
		monthEntry.SetText(strconv.Itoa(int(now.Month())))
		dayEntry.SetText(strconv.Itoa(now.Day()))
		hourEntry.SetText(strconv.Itoa(now.Hour()))
		minuteEntry.SetText(strconv.Itoa(now.Minute()))
		secondEntry.SetText(strconv.Itoa(now.Second()))
	})

	todayStartButton := widget.NewButton("今天0点", func() {
		now := time.Now()
		yearEntry.SetText(strconv.Itoa(now.Year()))
		monthEntry.SetText(strconv.Itoa(int(now.Month())))
		dayEntry.SetText(strconv.Itoa(now.Day()))
		hourEntry.SetText("0")
		minuteEntry.SetText("0")
		secondEntry.SetText("0")
	})

	tomorrowStartButton := widget.NewButton("明天0点", func() {
		tomorrow := time.Now().AddDate(0, 0, 1)
		yearEntry.SetText(strconv.Itoa(tomorrow.Year()))
		monthEntry.SetText(strconv.Itoa(int(tomorrow.Month())))
		dayEntry.SetText(strconv.Itoa(tomorrow.Day()))
		hourEntry.SetText("0")
		minuteEntry.SetText("0")
		secondEntry.SetText("0")
	})

	shortcutButtons := container.NewHBox(
		nowButton,
		todayStartButton,
		tomorrowStartButton,
	)

	content := container.NewVBox(
		widget.NewLabel("请选择日期和时间："),
		shortcutButtons,
		widget.NewSeparator(),
		container.NewGridWithColumns(2,
			widget.NewLabel("年:"), yearEntry,
			widget.NewLabel("月:"), monthEntry,
			widget.NewLabel("日:"), dayEntry,
			widget.NewLabel("时:"), hourEntry,
			widget.NewLabel("分:"), minuteEntry,
			widget.NewLabel("秒:"), secondEntry,
		),
	)

	dialog.NewCustomConfirm(title, "确定", "取消", content, func(ok bool) {
		if ok {
			// 获取输入值，如果为空或无效则使用默认值
			yearStr := strings.TrimSpace(yearEntry.Text)
			monthStr := strings.TrimSpace(monthEntry.Text)
			dayStr := strings.TrimSpace(dayEntry.Text)
			hourStr := strings.TrimSpace(hourEntry.Text)
			minuteStr := strings.TrimSpace(minuteEntry.Text)
			secondStr := strings.TrimSpace(secondEntry.Text)

			// 使用当前时间作为默认值
			defaultTime := time.Now()
			year := defaultTime.Year()
			month := int(defaultTime.Month())
			day := defaultTime.Day()
			hour := defaultTime.Hour()
			minute := defaultTime.Minute()
			second := defaultTime.Second()

			// 解析用户输入，如果有效则覆盖默认值
			if y, err := strconv.Atoi(yearStr); err == nil && y >= 1900 && y <= 2100 {
				year = y
			}
			if m, err := strconv.Atoi(monthStr); err == nil && m >= 1 && m <= 12 {
				month = m
			}
			if d, err := strconv.Atoi(dayStr); err == nil && d >= 1 && d <= 31 {
				day = d
			}
			if h, err := strconv.Atoi(hourStr); err == nil && h >= 0 && h <= 23 {
				hour = h
			}
			if mm, err := strconv.Atoi(minuteStr); err == nil && mm >= 0 && mm <= 59 {
				minute = mm
			}
			if s, err := strconv.Atoi(secondStr); err == nil && s >= 0 && s <= 59 {
				second = s
			}

			// 验证日期有效性
			if !g.validateDate(year, month, day) {
				dialog.ShowError(fmt.Errorf("无效的日期：%d年%d月%d日", year, month, day), g.window)
				return
			}

			// 验证时间有效性
			if hour < 0 || hour > 23 || minute < 0 || minute > 59 || second < 0 || second > 59 {
				dialog.ShowError(fmt.Errorf("无效的时间：%02d:%02d:%02d", hour, minute, second), g.window)
				return
			}

			selectedTime := time.Date(year, time.Month(month), day, hour, minute, second, 0, time.Local)
			entry.SetText(selectedTime.Format("2006-01-02 15:04:05"))
		}
	}, g.window).Show()
}

// validateDate 验证日期是否有效
func (g *GUI) validateDate(year, month, day int) bool {
	if month < 1 || month > 12 {
		return false
	}

	daysInMonth := []int{31, 28, 31, 30, 31, 30, 31, 31, 30, 31, 30, 31}

	// 检查闰年
	if year%4 == 0 && (year%100 != 0 || year%400 == 0) {
		daysInMonth[1] = 29 // 2月29日
	}

	return day >= 1 && day <= daysInMonth[month-1]
}

// createNumberEntry 创建数字输入框
func (g *GUI) createNumberEntry(value, min, max int) *widget.Entry {
	entry := widget.NewEntry()
	entry.SetText(strconv.Itoa(value))
	entry.SetPlaceHolder(fmt.Sprintf("%d-%d", min, max))

	// 存储上一次的有效值，用于在输入无效时恢复
	validValue := value
	// 防止递归调用
	isUpdating := false

	entry.OnChanged = func(text string) {
		// 防止递归调用
		if isUpdating {
			return
		}

		if text == "" {
			// 允许清空输入框
			return
		}

		// 检查输入是否为纯数字
		if _, err := strconv.Atoi(text); err != nil {
			// 输入包含非数字字符，恢复到上一个有效值
			isUpdating = true
			entry.SetText(strconv.Itoa(validValue))
			isUpdating = false
			return
		}

		// 输入是有效数字，更新validValue（无论是否在范围内）
		if val, _ := strconv.Atoi(text); val >= min && val <= max {
			validValue = val
		}
		// 如果超出范围，允许用户继续输入，不立即纠正
	}

	return entry
}

// addLog 添加日志消息到GUI日志显示区域
func (g *GUI) addLog(message string) {
	if g.logText == nil {
		return
	}

	timestamp := time.Now().Format("15:04:05")
	logLine := fmt.Sprintf("[%s] %s\n", timestamp, message)

	// 追加新内容到日志
	currentText := g.logText.Text
	newText := currentText + logLine
	g.logText.SetText(newText)

	// 自动滚动到底部
	g.logScroll.ScrollToBottom()
}

// showError 显示错误信息
func (g *GUI) showError(message string) {
	g.statusLabel.SetText("处理失败: " + message)
	g.addLog("错误: " + message)
	g.progressBar.Hide()
	dialog.ShowError(errors.New(message), g.window)
}
