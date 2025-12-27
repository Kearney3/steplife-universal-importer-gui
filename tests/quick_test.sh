#!/bin/bash

# 快速测试脚本

echo "🚀 一生足迹数据导入器 - 快速测试"
echo "=================================="
echo

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 检查Go环境
echo "1. 检查Go环境..."
if ! command -v go &> /dev/null; then
    echo "❌ Go未安装，请访问 https://golang.org/dl/ 下载安装"
    exit 1
fi

GO_VERSION=$(go version | grep -o 'go[0-9]\+\.[0-9]\+\.[0-9]\+')
echo "✅ Go版本：$GO_VERSION"
echo

# 下载依赖
echo "2. 下载项目依赖..."
cd "$PROJECT_ROOT"
go mod tidy
if [ $? -ne 0 ]; then
    echo "❌ 依赖下载失败"
    exit 1
fi
echo "✅ 依赖下载完成"
echo

# 构建程序
echo "3. 构建程序..."
go build -o main ./cmd
if [ $? -ne 0 ]; then
    echo "❌ 构建失败"
    exit 1
fi
echo "✅ 构建完成：main"
echo

# 切换回tests目录
cd "$SCRIPT_DIR"

# 准备测试环境
echo "4. 准备测试环境..."
mkdir -p ../output
echo "✅ 测试目录准备完成"
echo

# 运行命令行测试
echo "5. 运行命令行模式测试..."
./test_cli.sh
echo

# 验证输出
echo "6. 验证输出结果..."
./verify_output.sh
echo

echo "🎉 快速测试完成！"
echo

echo "💡 下一步："
echo "  - 运行 '../main' 启动GUI界面进行可视化测试"
echo "  - 查看 ./TEST_README.md 获取详细测试指南"
echo "  - 使用 ./test_data/ 目录中的文件进行更多测试"







