package server

import (
	"fmt"
	"math"
	"path"
	consts "steplife-universal-importer/internal/const"
	"steplife-universal-importer/internal/model"
	"steplife-universal-importer/internal/parser"
	"steplife-universal-importer/internal/utils"
	"steplife-universal-importer/internal/utils/logx"
	"steplife-universal-importer/internal/utils/pointcalc"
	"time"
)

// Run
//
//	@Description: 	执行
//	@return error
func Run(config model.Config) error {
	// 尝试多个可能的source_data目录位置
	sourceDataPaths := []string{
		"./source_data",  // 当前目录
		"../source_data", // 父目录（从tests目录运行时）
	}

	var directory string
	var filePathMap map[string][]string
	var err error

	for _, path := range sourceDataPaths {
		filePathMap, err = utils.GetAllFilePath(path)
		if err == nil {
			directory = path
			logx.InfoF("找到source_data目录：%s", path)
			break
		}
	}

	if err != nil {
		return fmt.Errorf("failed to read directory from any location: %w", err)
	}

	// 根据source_data目录位置确定output.csv路径
	var csvFilePath string
	if directory == "../source_data" {
		csvFilePath = "../output.csv"
	} else {
		csvFilePath = "./output.csv"
	}

	csvExisted, err := utils.CreateCSVFile(csvFilePath)
	if err != nil {
		return err
	}

	// 如果文件曾经不存在，则写入CSV文件头
	if !csvExisted {
		sl := model.NewStepLife()
		err = utils.WriteCSV(csvFilePath, sl.CSVHeader)
		if err != nil {
			logx.ErrorF("写入CSV文件头失败：%s", csvFilePath)
			return err
		}
	}

	for fileType, paths := range filePathMap {
		for i, filePath := range paths {
			logx.InfoF("处理第%d个文件（%s）", i, filePath)

			sl, err := parseOne(fileType, filePath, config)
			if err != nil {
				logx.ErrorF("处理第%d个文件（%s）失败：%s", i, filePath, err)
				return err
			}

			err = utils.WriteCSV(csvFilePath, sl.CSVData)
			if err != nil {
				logx.ErrorF("写入CSV文件失败：%s", csvFilePath)
				return err
			}

			// 更新起始时间戳
			config.PathStartTimestamp += int64(len(sl.CSVData))
		}
	}

	return nil
}

// ProcessSingleFile 处理单个文件
func ProcessSingleFile(fileType, filePath, csvFilePath string, config model.Config) error {
	// 创建CSV文件
	csvExisted, err := utils.CreateCSVFile(csvFilePath)
	if err != nil {
		return err
	}

	// 如果文件不存在，写入CSV文件头
	if !csvExisted {
		sl := model.NewStepLife()
		err = utils.WriteCSV(csvFilePath, sl.CSVHeader)
		if err != nil {
			logx.ErrorF("写入CSV文件头失败：%s", csvFilePath)
			return err
		}
	}

	logx.InfoF("处理文件：%s", filePath)

	sl, err := processOneFile(fileType, filePath, config)
	if err != nil {
		logx.ErrorF("处理文件失败：%s", filePath)
		return err
	}

	err = utils.WriteCSV(csvFilePath, sl.CSVData)
	if err != nil {
		logx.ErrorF("写入CSV文件失败：%s", csvFilePath)
		return err
	}

	return nil
}

func processOneFile(fileType, filePath string, config model.Config) (*model.StepLife, error) {
	var adaptor parser.FileAdaptor

	switch fileType {
	case consts.FileTypeCommon:
		adaptor = parser.CreateAdaptor(path.Ext(filePath))
	case consts.FileTypeVariFlight:
		logx.ErrorF("飞常准数据暂不支持......")
		return nil, nil
	default:
		logx.ErrorF("不支持的文件类型：%s", fileType)
		return nil, fmt.Errorf("不支持的文件类型：%s", fileType)
	}

	if adaptor == nil {
		return nil, fmt.Errorf("不支持的结构解析（%s）", fileType)
	}

	content, err := utils.ReadFile(filePath)
	if err != nil {
		logx.ErrorF("读取文件失败：%s", filePath)
		return nil, err
	}

	latLngData, err := adaptor.Parse(content)
	if err != nil {
		logx.ErrorF("解析文件失败：%s", filePath)
		return nil, err
	}

	sl, err := convertToStepLifeWithAdvancedOptions(config, latLngData)
	if err != nil {
		logx.ErrorF("转换文件失败：%s", filePath)
		return nil, err
	}

	return sl, nil
}

func convertToStepLifeWithAdvancedOptions(config model.Config, points []model.Point) (*model.StepLife, error) {
	sl := model.NewStepLife()
	logx.Info("处理经纬度坐标（高级模式）")

	startTimestamp := config.PathStartTimestamp
	if startTimestamp == 0 {
		startTimestamp = time.Now().Unix()
	}

	// 如果开始时间大于结束时间，反转轨迹点顺序并交换时间戳
	if config.PathEndTimestamp > 0 && startTimestamp > config.PathEndTimestamp {
		logx.Info("检测到开始时间大于结束时间，自动反转轨迹顺序")
		// 反转点数组
		for i, j := 0, len(points)-1; i < j; i, j = i+1, j-1 {
			points[i], points[j] = points[j], points[i]
		}
		// 交换时间戳
		startTimestamp, config.PathEndTimestamp = config.PathEndTimestamp, startTimestamp
	}

	// 如果设置了结束时间，需要先计算总点数（包括插值点）以正确计算时间间隔
	var totalPoints int64 = int64(len(points))
	if config.EnableInsertPointStrategy == 1 {
		// 计算插值后的总点数
		totalPoints = 1 // 第一个点
		for i := 1; i < len(points); i++ {
			interpolatedPoints := pointcalc.Calculate(points[i-1], points[i], config.InsertPointDistance)
			totalPoints += int64(len(interpolatedPoints))
		}
	}

	// 计算时间间隔（如果设置了结束时间）
	var timeInterval int64 = 1 // 默认1秒间隔
	useEndTime := config.PathEndTimestamp > 0 && startTimestamp > 0 && totalPoints > 1
	if useEndTime {
		totalDuration := config.PathEndTimestamp - startTimestamp
		timeInterval = totalDuration / (totalPoints - 1)
		if timeInterval < 1 {
			timeInterval = 1
		}
	}

	// 使用点索引来计算时间戳，确保最后一个点的时间戳等于结束时间
	pointIndex := int64(0)

	for i, point := range points {
		// 第0个坐标或者不需要插入值，不需要计算中间点，直接写入
		if i == 0 || config.EnableInsertPointStrategy == 0 {
			// 计算当前点的时间戳
			var currentTimestamp int64
			if useEndTime {
				currentTimestamp = startTimestamp + pointIndex*timeInterval
				// 如果是最后一个点，使用精确的结束时间
				if i == len(points)-1 {
					currentTimestamp = config.PathEndTimestamp
				}
			} else {
				currentTimestamp = startTimestamp + pointIndex*timeInterval
			}

			row := model.NewRow()
			row.DataTime = currentTimestamp
			row.Altitude = config.DefaultAltitude         // 使用配置的海拔高度
			row.Speed = calculateSpeed(config, points, i) // 计算速度
			row.Latitude = point.Latitude
			row.Longitude = point.Longitude
			sl.AddCSVRow(*row)
			pointIndex++
		} else {
			interpolatedPoints := pointcalc.Calculate(points[i-1], point, config.InsertPointDistance)
			for j, interpolatedPoint := range interpolatedPoints {
				// 计算当前点的时间戳
				var currentTimestamp int64
				if useEndTime {
					currentTimestamp = startTimestamp + pointIndex*timeInterval
					// 如果是最后一个点，使用精确的结束时间
					if i == len(points)-1 && j == len(interpolatedPoints)-1 {
						currentTimestamp = config.PathEndTimestamp
					}
				} else {
					currentTimestamp = startTimestamp + pointIndex*timeInterval
				}

				row := model.NewRow()
				row.Point = interpolatedPoint
				row.DataTime = currentTimestamp
				row.Altitude = config.DefaultAltitude
				row.Speed = calculateSpeed(config, points, i)
				sl.AddCSVRow(*row)
				pointIndex++
			}
		}
	}

	// 确保最后一个点的时间戳等于结束时间（如果设置了结束时间）
	if useEndTime && len(sl.CSVData) > 0 {
		// CSVData 的第一个元素是 DataTime
		sl.CSVData[len(sl.CSVData)-1][0] = fmt.Sprintf("%d", config.PathEndTimestamp)
	}

	logx.InfoF("处理经纬度完成，原始坐标%d个，插点后坐标%d个", len(points), len(sl.CSVData))
	return sl, nil
}

// calculateSpeed 计算速度
func calculateSpeed(config model.Config, points []model.Point, currentIndex int) float64 {
	if config.SpeedMode == "manual" {
		return config.ManualSpeed
	}

	// 自动计算速度
	if currentIndex == 0 || currentIndex >= len(points) {
		return 0.0
	}

	// 计算两点间的距离和时间差来估算速度
	prevPoint := points[currentIndex-1]
	currPoint := points[currentIndex]

	// 使用Haversine公式计算距离（米）
	distance := calculateHaversineDistance(prevPoint.Latitude, prevPoint.Longitude, currPoint.Latitude, currPoint.Longitude)

	// 估算时间差（假设平均速度）
	estimatedTimeDiff := 1.0 // 1秒
	if distance > 0 {
		// 假设平均步行速度为1.5 m/s，计算合理的时间差
		estimatedTimeDiff = distance / 1.5
		if estimatedTimeDiff < 1 {
			estimatedTimeDiff = 1
		}
	}

	return distance / estimatedTimeDiff
}

// calculateHaversineDistance 计算两点间的球面距离（米）
func calculateHaversineDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const earthRadius = 6371000.0 // 地球半径（米）

	// 转换为弧度
	lat1Rad := lat1 * math.Pi / 180
	lon1Rad := lon1 * math.Pi / 180
	lat2Rad := lat2 * math.Pi / 180
	lon2Rad := lon2 * math.Pi / 180

	deltaLat := lat2Rad - lat1Rad
	deltaLon := lon2Rad - lon1Rad

	a := math.Sin(deltaLat/2)*math.Sin(deltaLat/2) +
		math.Cos(lat1Rad)*math.Cos(lat2Rad)*
			math.Sin(deltaLon/2)*math.Sin(deltaLon/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return earthRadius * c
}

func parseOne(fileType, filePath string, config model.Config) (*model.StepLife, error) {

	var adaptor parser.FileAdaptor

	switch fileType {
	case consts.FileTypeCommon:
		adaptor = parser.CreateAdaptor(path.Ext(filePath))
	case consts.FileTypeVariFlight:
		// TODO
		logx.ErrorF("飞常准数据后续支持......")
		return nil, nil
	default:
		logx.ErrorF("不支持的文件类型：%s", fileType)
		return nil, fmt.Errorf("不支持的文件类型：%s", fileType)
	}

	if adaptor == nil {
		return nil, fmt.Errorf("不支持的结构解析（%s）", fileType)
	}

	content, err := utils.ReadFile(filePath)
	if err != nil {
		logx.ErrorF("读取文件失败：%s", filePath)
		return nil, err
	}

	latLngData, err := adaptor.Parse(content)
	if err != nil {
		logx.ErrorF("解析文件失败：%s", filePath)
		return nil, err
	}

	sl, err := adaptor.Convert2StepLife(config, latLngData)
	if err != nil {
		logx.ErrorF("转换文件失败：%s", filePath)
		return nil, err
	}

	return sl, nil
}
