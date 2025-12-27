package logx

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os"
	"sync"
	"time"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

var sugar *zap.SugaredLogger
var guiLogCallback func(string) // GUI日志回调函数
var guiLogMutex sync.Mutex      // 保护GUI日志回调的互斥锁

func init() {
	NewLogger()
	//log.Println("zap log init success")
}

// SetGUILogger 设置GUI日志回调函数
func SetGUILogger(callback func(string)) {
	guiLogMutex.Lock()
	defer guiLogMutex.Unlock()
	guiLogCallback = callback
}

func NewLogger() {
	core := newCore(zap.DebugLevel)
	caller := zap.AddCaller()
	// 向上跳一层，否则日志中caller显示不正确
	callerSkip := zap.AddCallerSkip(1)
	// 构造日志
	sugar = zap.New(core, caller, callerSkip).Sugar()
}

// guiLogWriter 实现zapcore.WriteSyncer接口，将日志转发到GUI
type guiLogWriter struct{}

func (w *guiLogWriter) Write(p []byte) (n int, err error) {
	guiLogMutex.Lock()
	callback := guiLogCallback
	guiLogMutex.Unlock()

	if callback != nil {
		// 解析JSON日志，提取消息
		var logEntry map[string]interface{}
		if err := json.Unmarshal(p, &logEntry); err == nil {
			// 提取关键信息
			level := ""
			message := ""
			caller := ""

			if l, ok := logEntry["level"].(string); ok {
				level = l
			}
			if m, ok := logEntry["message"].(string); ok {
				message = m
			}
			if c, ok := logEntry["caller"].(string); ok {
				caller = c
			}

			// 格式化日志消息
			if caller != "" {
				callback(fmt.Sprintf("[%s] %s (%s)", level, message, caller))
			} else {
				callback(fmt.Sprintf("[%s] %s", level, message))
			}
		} else {
			// 如果不是JSON格式，直接使用原始内容
			callback(string(bytes.TrimSpace(p)))
		}
	}
	return len(p), nil
}

func (w *guiLogWriter) Sync() error {
	return nil
}

func newCore(level zapcore.Level) zapcore.Core {

	//日志文件路径配置
	hook := lumberjack.Logger{
		Filename: ".cache/local.log", // 日志文件路径
		MaxSize:  1024,               // 每个日志文件保存的最大尺寸 单位：M
		Compress: true,               // 是否压缩
	}

	// 设置日志级别
	atomicLevel := zap.NewAtomicLevel()
	atomicLevel.SetLevel(level)

	//公用编码器
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",                        // json时时间键
		LevelKey:       "level",                       // json时日志等级键
		NameKey:        "name",                        // json时日志记录器名
		CallerKey:      "caller",                      // json时日志文件信息键
		MessageKey:     "message",                     // json时日志消息键
		StacktraceKey:  "stackTrace",                  // json时堆栈键
		LineEnding:     zapcore.DefaultLineEnding,     // 友好日志换行符
		EncodeLevel:    zapcore.CapitalLevelEncoder,   // 友好日志等级名大小写（info INFO）
		EncodeTime:     timeEncoder,                   // 友好日志时日期格式化
		EncodeDuration: zapcore.StringDurationEncoder, // 时间序列化
		EncodeCaller:   zapcore.ShortCallerEncoder,    // 日志文件信息（包/文件.go:行号）
		EncodeName:     zapcore.FullNameEncoder,
	}

	// 创建写入器列表（总是包含GUI写入器，内部会检查回调是否存在）
	writers := []zapcore.WriteSyncer{
		zapcore.AddSync(os.Stdout),
		zapcore.AddSync(&hook),
		&guiLogWriter{}, // GUI日志写入器（内部会检查回调是否存在）
	}

	return zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),   // 编码器配置
		zapcore.NewMultiWriteSyncer(writers...), // 打印到控制台、文件和GUI
		atomicLevel,                             // 日志级别
	)
}

/**
 * @Description: 格式化时间
 * @param t
 * @param enc
 */
func timeEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
	enc.AppendString(t.Format("2006-01-02 15:04:05.000"))
}
