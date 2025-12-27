#!/bin/bash

# macOS .app 构建脚本 - 一生足迹数据导入器
# 用于本地测试 macOS .app 包的构建过程

set -e  # 遇到错误立即退出

echo "=========================================="
echo "macOS .app 构建脚本"
echo "=========================================="

# 检查是否在 macOS 上运行
if [[ "$OSTYPE" != "darwin"* ]]; then
    echo "错误: 此脚本只能在 macOS 上运行"
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

echo "目标架构: $ARCH"
echo ""

# 从 consts.go 提取应用名称（兼容 macOS grep）
APP_NAME=$(grep 'AppName = ' internal/const/consts.go | sed 's/.*AppName = "\(.*\)".*/\1/' | head -1)
if [ -z "$APP_NAME" ]; then
    echo "错误: 无法从 consts.go 提取应用名称"
    exit 1
fi

# 从 consts.go 提取版本号（兼容 macOS grep）
VERSION=$(grep 'Version = ' internal/const/consts.go | sed 's/.*Version = "\(.*\)".*/\1/' | head -1)
if [ -z "$VERSION" ]; then
    echo "错误: 无法从 consts.go 提取版本号"
    exit 1
fi

# 从 go.mod 获取模块名，用于生成 Bundle ID
MODULE_NAME=$(grep '^module ' go.mod | awk '{print $2}')
BUNDLE_ID=$(echo "$MODULE_NAME" | tr '/' '.')

# 生成可执行文件名（基于模块名）
EXECUTABLE_NAME=$(basename "$MODULE_NAME")

# 生成 .app 名称（应用名称-架构.app）
APP_BUNDLE_NAME="${APP_NAME}-${ARCH}.app"

# 生成可执行文件输出名称
OUTPUT_NAME="steplife-universal-importer-darwin-${ARCH}"

echo "应用信息:"
echo "  应用名称: $APP_NAME"
echo "  版本号: $VERSION"
echo "  Bundle ID: $BUNDLE_ID"
echo "  可执行文件名: $EXECUTABLE_NAME"
echo "  .app 包名称: $APP_BUNDLE_NAME"
echo ""

# 清理之前的构建
echo "清理之前的构建..."
rm -rf "$APP_BUNDLE_NAME"
rm -f "$OUTPUT_NAME"
echo ""

# 构建可执行文件
echo "构建可执行文件..."
GOOS=darwin GOARCH=$ARCH go build -tags=embed_font -trimpath -ldflags="-s -w" -o "$OUTPUT_NAME" ./cmd

if [ $? -ne 0 ]; then
    echo "错误: 构建失败"
    exit 1
fi

echo "构建完成: $OUTPUT_NAME"
ls -lh "$OUTPUT_NAME"
echo ""

# 创建 .app 目录结构
echo "创建 .app 包结构..."
mkdir -p "${APP_BUNDLE_NAME}/Contents/MacOS"
mkdir -p "${APP_BUNDLE_NAME}/Contents/Resources"
echo ""

# 移动可执行文件到 MacOS 目录
echo "移动可执行文件到 .app 包..."
mv "$OUTPUT_NAME" "${APP_BUNDLE_NAME}/Contents/MacOS/${EXECUTABLE_NAME}"
chmod +x "${APP_BUNDLE_NAME}/Contents/MacOS/${EXECUTABLE_NAME}"
echo ""

# 处理图标（如果存在）
ICON_FILE=""
if [ -f "internal/gui/resources/icon.png" ]; then
    echo "处理图标文件..."
    
    # 尝试创建 .icns 图标（更好的显示效果）
    if command -v iconutil &> /dev/null; then
        echo "尝试创建 .icns 图标..."
        ICONSET_DIR="${APP_BUNDLE_NAME}/Contents/Resources/icon.iconset"
        mkdir -p "$ICONSET_DIR"
        
        # 创建不同尺寸的图标（.icns 需要多个尺寸）
        # 使用 sips 工具（macOS 自带）调整图标尺寸
        if command -v sips &> /dev/null; then
            sips -z 16 16 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_16x16.png" &>/dev/null
            sips -z 32 32 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_16x16@2x.png" &>/dev/null
            sips -z 32 32 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_32x32.png" &>/dev/null
            sips -z 64 64 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_32x32@2x.png" &>/dev/null
            sips -z 128 128 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_128x128.png" &>/dev/null
            sips -z 256 256 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_128x128@2x.png" &>/dev/null
            sips -z 256 256 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_256x256.png" &>/dev/null
            sips -z 512 512 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_256x256@2x.png" &>/dev/null
            sips -z 512 512 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_512x512.png" &>/dev/null
            sips -z 1024 1024 internal/gui/resources/icon.png --out "${ICONSET_DIR}/icon_512x512@2x.png" &>/dev/null
            
            # 转换为 .icns
            iconutil -c icns "$ICONSET_DIR" -o "${APP_BUNDLE_NAME}/Contents/Resources/icon.icns" 2>/dev/null
            if [ $? -eq 0 ] && [ -f "${APP_BUNDLE_NAME}/Contents/Resources/icon.icns" ]; then
                rm -rf "$ICONSET_DIR"
                ICON_FILE="icon"
                echo "已创建 .icns 图标文件"
            else
                # 如果转换失败，回退到 PNG
                rm -rf "$ICONSET_DIR"
                cp "internal/gui/resources/icon.png" "${APP_BUNDLE_NAME}/Contents/Resources/icon.png"
                ICON_FILE="icon"
                echo "已复制 PNG 图标文件（.icns 转换失败）"
            fi
        else
            # 如果没有 sips，直接使用 PNG
            cp "internal/gui/resources/icon.png" "${APP_BUNDLE_NAME}/Contents/Resources/icon.png"
            ICON_FILE="icon"
            echo "已复制 PNG 图标文件"
        fi
    else
        # 如果没有 iconutil，直接使用 PNG
        cp "internal/gui/resources/icon.png" "${APP_BUNDLE_NAME}/Contents/Resources/icon.png"
        ICON_FILE="icon"
        echo "已复制 PNG 图标文件"
    fi
else
    echo "警告: 未找到图标文件 internal/gui/resources/icon.png"
fi
echo ""

# 创建 Info.plist
echo "创建 Info.plist..."
cat > "${APP_BUNDLE_NAME}/Contents/Info.plist" <<EOF
<?xml version="1.0" encoding="UTF-8"?>
<!DOCTYPE plist PUBLIC "-//Apple//DTD PLIST 1.0//EN" "http://www.apple.com/DTDs/PropertyList-1.0.dtd">
<plist version="1.0">
<dict>
  <key>CFBundleExecutable</key>
  <string>${EXECUTABLE_NAME}</string>
  <key>CFBundleIdentifier</key>
  <string>${BUNDLE_ID}</string>
  <key>CFBundleName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleDisplayName</key>
  <string>${APP_NAME}</string>
  <key>CFBundleVersion</key>
  <string>${VERSION}</string>
  <key>CFBundleShortVersionString</key>
  <string>${VERSION}</string>
  <key>CFBundlePackageType</key>
  <string>APPL</string>
  <key>CFBundleSignature</key>
  <string>????</string>
  <key>LSMinimumSystemVersion</key>
  <string>10.13</string>
  <key>NSHighResolutionCapable</key>
  <true/>
EOF

# 如果图标存在，添加图标配置
if [ -n "$ICON_FILE" ]; then
    cat >> "${APP_BUNDLE_NAME}/Contents/Info.plist" <<EOF
  <key>CFBundleIconFile</key>
  <string>${ICON_FILE}</string>
EOF
fi

cat >> "${APP_BUNDLE_NAME}/Contents/Info.plist" <<EOF
</dict>
</plist>
EOF
echo "Info.plist 已创建"
if [ -n "$ICON_FILE" ]; then
    echo "图标已配置: ${ICON_FILE}"
fi
echo ""

# 显示构建结果
echo "=========================================="
echo "构建完成！"
echo "=========================================="
echo ""
echo ".app 包信息:"
ls -lh "${APP_BUNDLE_NAME}"
echo ""
echo ".app 包结构:"
find "${APP_BUNDLE_NAME}" -type f | sort
echo ""
echo ".app 包位置: $(pwd)/${APP_BUNDLE_NAME}"
echo ""
echo "你可以通过以下方式测试:"
echo "  1. 双击 ${APP_BUNDLE_NAME} 运行应用"
echo "  2. 或在终端运行: open ${APP_BUNDLE_NAME}"
echo "  3. 或直接运行: ${APP_BUNDLE_NAME}/Contents/MacOS/${EXECUTABLE_NAME}"

