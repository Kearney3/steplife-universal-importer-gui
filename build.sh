#!/bin/bash

# 构建脚本 - 一生足迹数据导入器

# 解析命令行参数
EMBED_FONT=true
for arg in "$@"; do
    case $arg in
        --no-embed-font)
            EMBED_FONT=false
            shift
            ;;
        -h|--help)
            echo "用法: $0 [选项]"
            echo ""
            echo "选项:"
            echo "  --no-embed-font    不嵌入字体，使用文件系统路径加载字体"
            echo "  -h, --help         显示此帮助信息"
            echo ""
            echo "默认情况下会嵌入字体到可执行文件中。"
            exit 0
            ;;
        *)
            echo "未知参数: $arg"
            echo "使用 $0 --help 查看帮助信息"
            exit 1
            ;;
    esac
done

echo "构建一生足迹数据导入器..."

# 检测操作系统
if [[ "$OSTYPE" == "linux-gnu"* ]]; then
    OS="linux"
elif [[ "$OSTYPE" == "darwin"* ]]; then
    OS="darwin"
elif [[ "$OSTYPE" == "msys" ]] || [[ "$OSTYPE" == "win32" ]]; then
    OS="windows"
else
    echo "不支持的操作系统: $OSTYPE"
    exit 1
fi

# 检测架构
ARCH=$(uname -m)
if [[ "$ARCH" == "x86_64" ]]; then
    ARCH="amd64"
elif [[ "$ARCH" == "arm64" ]] || [[ "$ARCH" == "aarch64" ]]; then
    ARCH="arm64"
else
    echo "不支持的架构: $ARCH"
    exit 1
fi

echo "目标平台: $OS/$ARCH"

# 清理之前的构建
rm -f main main.exe rsrc_windows_*.syso cmd/rsrc_windows_*.syso

# 构建参数
# -trimpath: 移除文件系统中的路径信息，使构建更可重现
# -ldflags="-s -w": 减小二进制文件大小
#   -s: 省略符号表和调试信息
#   -w: 省略DWARF符号表

# 根据参数决定是否使用 build tag
BUILD_TAGS=""
if [ "$EMBED_FONT" = true ]; then
    BUILD_TAGS="-tags=embed_font"
    echo "字体嵌入: 启用（字体将嵌入到可执行文件中）"
else
    echo "字体嵌入: 禁用（将从文件系统加载字体）"
fi

# Windows 图标处理
if [[ "$OS" == "windows" ]]; then
    if [ -f "internal/gui/resources/icon.ico" ]; then
        echo "处理 Windows 图标..."
        # 检查 rsrc 工具是否已安装
        if ! command -v rsrc &> /dev/null; then
            echo "安装 rsrc 工具..."
            go install github.com/akavel/rsrc@latest
        fi
        
        # 生成资源文件到 cmd 目录（Go 编译器会在构建包时自动包含同目录下的 .syso 文件）
        rsrc -ico internal/gui/resources/icon.ico -o cmd/rsrc_windows_${ARCH}.syso
        
        if [ -f "cmd/rsrc_windows_${ARCH}.syso" ]; then
            echo "Windows 图标资源文件已创建: cmd/rsrc_windows_${ARCH}.syso"
        else
            echo "警告: 无法创建图标资源文件"
        fi
    else
        echo "警告: 未找到图标文件 internal/gui/resources/icon.ico"
    fi
fi

# 构建
if [[ "$OS" == "windows" ]]; then
    GOOS=$OS GOARCH=$ARCH go build $BUILD_TAGS -trimpath -ldflags="-s -w" -o main.exe ./cmd
    
    # 清理资源文件
    rm -f cmd/rsrc_windows_*.syso 2>/dev/null || true
    if [ $? -eq 0 ]; then
        echo "构建完成: main.exe"
        # 显示文件大小
        if command -v ls &> /dev/null; then
            ls -lh main.exe
        fi
    else
        echo "构建失败！"
        exit 1
    fi
else
    GOOS=$OS GOARCH=$ARCH go build $BUILD_TAGS -trimpath -ldflags="-s -w" -o main ./cmd
    if [ $? -eq 0 ]; then
        echo "构建完成: main"
        # 显示文件大小
        if command -v ls &> /dev/null; then
            ls -lh main
        fi
    else
        echo "构建失败！"
        exit 1
    fi
fi

echo "构建成功！"
