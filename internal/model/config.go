package model

type Config struct {
	EnableInsertPointStrategy int     `ini:"enableInsertPointStrategy"`
	InsertPointDistance       int     `ini:"insertPointDistance"`
	PathStartTime             string  `ini:"pathStartTime"`
	PathEndTime               string  `ini:"pathEndTime"`
	TimeInterval              int64   `ini:"timeInterval"` // 时间间隔（秒）
	Timezone                  string  `ini:"timezone"`      // 时区，如 "Asia/Shanghai"，空值表示使用系统本地时区
	PathStartTimestamp        int64
	PathEndTimestamp          int64
	DefaultAltitude           float64 `ini:"defaultAltitude"`
	SpeedMode                 string  `ini:"speedMode"` // "auto" or "manual"
	ManualSpeed               float64 `ini:"manualSpeed"`
	EnableBatchProcessing     int     `ini:"enableBatchProcessing"`
}
