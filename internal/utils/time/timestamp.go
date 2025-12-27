package timeUtils

import (
	"fmt"
	"strings"
	"time"
)

// ToTimestamp 尝试解析常见时间字符串为 Unix 时间戳（秒）
//
//	@Description:
//	@param timeStr	时间字符串
//	@return int64	时间戳
//	@return error
func ToTimestamp(timeStr string) (int64, error) {
	return ToTimestampWithTimezone(timeStr, "")
}

// ToTimestampWithTimezone 尝试解析常见时间字符串为 Unix 时间戳（秒），支持指定时区
//
//	@Description:
//	@param timeStr	时间字符串
//	@param timezone	时区名称，如 "Asia/Shanghai"，空字符串表示使用系统本地时区
//	@return int64	时间戳
//	@return error
func ToTimestampWithTimezone(timeStr string, timezone string) (int64, error) {
	// 常见时间格式列表，从最精确到最简单
	layouts := []string{
		time.RFC3339,          // 2024-01-03T03:53:22Z
		"2006-01-02 15:04:05", // 2020-10-20 16:49:00
		"2006-01-02 15:04",    // 2020-10-20 16:49
		"2006-01-02",          // 2020-10-20
		"2006/01/02 15:04:05", // 2020/10/20 16:49:00
		"2006/01/02 15:04",    // 2020/10/20 16:49
		"2006/01/02",          // 2020/10/20
	}

	timeStr = strings.TrimSpace(timeStr)
	var t time.Time
	var err error
	var loc *time.Location

	// 确定使用的时区
	if timezone != "" {
		loc, err = time.LoadLocation(timezone)
		if err != nil {
			return 0, fmt.Errorf("无效的时区: %s", timezone)
		}
	} else {
		loc = time.Local
	}

	for _, layout := range layouts {
		// 如果时间中含有Z或T等 UTC 标记，默认使用 time.Parse（UTC）
		if strings.ContainsAny(timeStr, "TZ") {
			t, err = time.Parse(layout, timeStr)
		} else {
			// 使用指定的时区或本地时区
			t, err = time.ParseInLocation(layout, timeStr, loc)
		}
		if err == nil {
			return t.Unix(), nil
		}
	}

	return 0, fmt.Errorf("无法解析时间字符串: %s", timeStr)
}
