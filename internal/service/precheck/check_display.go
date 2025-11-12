package precheck

import "sort"

// ColorStyle 颜色样式枚举
type ColorStyle int

const (
	ColorStyleNormal  ColorStyle = iota // 正常，不着色
	ColorStyleSuccess                   // 成功/正常（绿色）
	ColorStyleWarning                   // 警告（黄色）
	ColorStyleError                     // 错误（红色）
)

// FieldColorInfo 单个字段的颜色信息
type FieldColorInfo struct {
	Value      string     // 原始值
	ColorStyle ColorStyle // 颜色样式
}

// ApplyColor 应用颜色（终端）
func (f *FieldColorInfo) ApplyColor() string {
	switch f.ColorStyle {
	case ColorStyleSuccess:
		return ColorGreen + f.Value + ColorReset
	case ColorStyleWarning:
		return ColorYellow + f.Value + ColorReset
	case ColorStyleError:
		return ColorRed + f.Value + ColorReset
	default:
		return f.Value
	}
}

// GetColorClass 获取颜色CSS类名（Web）
func (f *FieldColorInfo) GetColorClass() string {
	switch f.ColorStyle {
	case ColorStyleSuccess:
		return "text-success"
	case ColorStyleWarning:
		return "text-warning"
	case ColorStyleError:
		return "text-danger"
	default:
		return ""
	}
}

// PrecheckDisplayItem 单个 HCA 的展示数据（带颜色信息）
type PrecheckDisplayItem struct {
	Hostname     string         // 主机名（用于分组，不着色）
	HCA          string         // HCA名称（不着色）
	SerialNumber string         // 序列号（不着色）
	PhysState    string         // 物理状态（不着色）
	State        string         // 逻辑状态（不着色）
	Speed        FieldColorInfo // 速度（可能着色）
	FwVer        FieldColorInfo // 固件版本（可能着色）
	BoardId      FieldColorInfo // 板卡ID（可能着色）
	Status       FieldColorInfo // 状态（着色）
}

// PrecheckDisplayData 预检查展示数据集合（包含统计信息和着色规则）
type PrecheckDisplayData struct {
	Items []PrecheckDisplayItem // 展示项列表

	// 统计信息
	HealthyCount   int
	UnhealthyCount int
	ErrorCount     int
	TotalCount     int
}

// NewPrecheckDisplayData 从 PrecheckResult 创建展示数据（应用着色规则）
func NewPrecheckDisplayData(results []PrecheckResult) *PrecheckDisplayData {
	display := &PrecheckDisplayData{
		Items:      make([]PrecheckDisplayItem, 0, len(results)),
		TotalCount: len(results),
	}

	// 1. 统计各字段的出现次数（用于着色规则）
	speedCounts := make(map[string]int)
	fwVerCounts := make(map[string]int)
	boardIdCounts := make(map[string]int)

	for _, result := range results {
		if result.Error == "" {
			if result.Speed != "" {
				speedCounts[result.Speed]++
			}
			if result.FwVer != "" {
				fwVerCounts[result.FwVer]++
			}
			if result.BoardId != "" {
				boardIdCounts[result.BoardId]++
			}
		}
	}

	// 2. 找出最常见的值（用于着色规则）
	maxSpeedCount := 0
	for _, count := range speedCounts {
		if count > maxSpeedCount {
			maxSpeedCount = count
		}
	}

	maxFwVerCount := 0
	for _, count := range fwVerCounts {
		if count > maxFwVerCount {
			maxFwVerCount = count
		}
	}

	maxBoardIdCount := 0
	for _, count := range boardIdCounts {
		if count > maxBoardIdCount {
			maxBoardIdCount = count
		}
	}

	// 3. 应用着色规则，创建展示项
	for _, result := range results {
		item := PrecheckDisplayItem{
			Hostname:     result.Hostname,
			HCA:          result.HCA,
			SerialNumber: result.SerialNumber,
			PhysState:    result.PhysState,
			State:        result.State,
		}

		// 处理 N/A 值
		if item.SerialNumber == "" {
			item.SerialNumber = "N/A"
		}

		// 处理错误情况
		if result.Error != "" {
			item.PhysState = "N/A"
			item.State = "N/A"
			item.SerialNumber = "N/A"

			item.Speed = FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
			item.FwVer = FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
			item.BoardId = FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
			item.Status = FieldColorInfo{Value: "[!] ERROR", ColorStyle: ColorStyleWarning}

			display.ErrorCount++
		} else {
			// 应用 Speed 着色规则
			item.Speed = display.applySpeedColor(result.Speed, speedCounts[result.Speed], maxSpeedCount)

			// 应用 FwVer 着色规则
			item.FwVer = display.applyFwVerColor(result.FwVer, fwVerCounts[result.FwVer], maxFwVerCount)

			// 应用 BoardId 着色规则
			item.BoardId = display.applyBoardIdColor(result.BoardId, boardIdCounts[result.BoardId], maxBoardIdCount)

			// 应用 Status 着色规则
			if result.IsHealthy {
				item.Status = FieldColorInfo{Value: "[+] HEALTHY", ColorStyle: ColorStyleSuccess}
				display.HealthyCount++
			} else {
				item.Status = FieldColorInfo{Value: "[X] UNHEALTHY", ColorStyle: ColorStyleError}
				display.UnhealthyCount++
			}
		}

		display.Items = append(display.Items, item)
	}

	return display
}

// applySpeedColor 应用速度字段着色规则
func (d *PrecheckDisplayData) applySpeedColor(speed string, count, maxCount int) FieldColorInfo {
	if speed == "" {
		return FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
	}

	// 规则：数量最多的标绿色，数量少的标红色
	if count == maxCount && maxCount > 1 {
		return FieldColorInfo{Value: speed, ColorStyle: ColorStyleSuccess}
	} else if count < maxCount {
		return FieldColorInfo{Value: speed, ColorStyle: ColorStyleError}
	}

	return FieldColorInfo{Value: speed, ColorStyle: ColorStyleNormal}
}

// applyFwVerColor 应用固件版本字段着色规则
func (d *PrecheckDisplayData) applyFwVerColor(fwVer string, count, maxCount int) FieldColorInfo {
	if fwVer == "" {
		return FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
	}

	// 规则：数量少的标黄色（版本不一致）
	if count < maxCount {
		return FieldColorInfo{Value: fwVer, ColorStyle: ColorStyleWarning}
	}

	return FieldColorInfo{Value: fwVer, ColorStyle: ColorStyleNormal}
}

// applyBoardIdColor 应用板卡ID字段着色规则
func (d *PrecheckDisplayData) applyBoardIdColor(boardId string, count, maxCount int) FieldColorInfo {
	if boardId == "" {
		return FieldColorInfo{Value: "N/A", ColorStyle: ColorStyleNormal}
	}

	// 规则：数量少的标黄色（型号不一致）
	if count < maxCount {
		return FieldColorInfo{Value: boardId, ColorStyle: ColorStyleWarning}
	}

	return FieldColorInfo{Value: boardId, ColorStyle: ColorStyleNormal}
}

// SortByHostAndHCA 按主机名和HCA排序
func (d *PrecheckDisplayData) SortByHostAndHCA() {
	sort.Slice(d.Items, func(i, j int) bool {
		if d.Items[i].Hostname != d.Items[j].Hostname {
			return d.Items[i].Hostname < d.Items[j].Hostname
		}
		return d.Items[i].HCA < d.Items[j].HCA
	})
}
