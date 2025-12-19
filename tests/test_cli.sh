#!/bin/bash

# 测试命令行模式脚本

echo "=== 一生足迹数据导入器 - 命令行模式测试 ==="
echo

# 获取项目根目录
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# 检查可执行文件是否存在
if [ ! -f "$PROJECT_ROOT/main" ]; then
    echo "错误：找不到可执行文件 main"
    echo "请先运行以下命令构建程序："
    echo "  cd $PROJECT_ROOT && go build -o main ./cmd"
    exit 1
fi

echo "✅ 找到可执行文件：$PROJECT_ROOT/main"
echo

# 检查测试数据
if [ ! -d "$PROJECT_ROOT/tests/test_data" ]; then
    echo "错误：找不到测试数据目录 $PROJECT_ROOT/tests/test_data"
    exit 1
fi

echo "✅ 找到测试数据目录：$PROJECT_ROOT/tests/test_data"
echo

# 检查配置文件
if [ ! -f "$PROJECT_ROOT/config.ini" ]; then
    echo "错误：找不到配置文件 config.ini"
    exit 1
fi

echo "✅ 找到配置文件：$PROJECT_ROOT/config.ini"
echo

# 备份原始配置
cp "$PROJECT_ROOT/config.ini" "$PROJECT_ROOT/config.ini.backup"

# 修改配置用于测试
cat > "$PROJECT_ROOT/config.ini" << EOF
# 测试配置
enableInsertPointStrategy = 1
insertPointDistance = 100
pathStartTime = 2024-01-01 08:00:00
pathEndTime = 2024-01-01 08:30:00
defaultAltitude = 100.0
speedMode = auto
manualSpeed = 1.5
enableBatchProcessing = 1
EOF

echo "✅ 配置测试参数"
echo

# 创建输出目录和源数据目录
mkdir -p "$PROJECT_ROOT/output" "$PROJECT_ROOT/source_data"

# 复制测试文件到source_data目录
cp test_data/* "$PROJECT_ROOT/source_data/"

echo "✅ 准备测试文件"
echo

# 运行命令行模式测试
echo "🚀 启动命令行模式测试..."
echo

"$PROJECT_ROOT/main" --cli

echo
echo "=== 测试完成 ==="
echo

# 检查输出结果
if [ -f "$PROJECT_ROOT/output.csv" ]; then
    echo "✅ 输出文件生成成功："
    ls -la "$PROJECT_ROOT/output.csv"
    # 复制到tests目录便于验证
    cp "$PROJECT_ROOT/output.csv" "./output/"
else
    echo "❌ 未找到输出文件 $PROJECT_ROOT/output.csv"
fi

echo
echo "📋 测试文件位置："
echo "  测试数据: ./test_data/"
echo "  输出文件: $PROJECT_ROOT/output.csv"

# 恢复原始配置
mv "$PROJECT_ROOT/config.ini.backup" "$PROJECT_ROOT/config.ini"

echo
echo "✅ 测试完成，配置已恢复"

