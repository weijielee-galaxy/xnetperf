package precheck

// ANSI 颜色代码
const (
	ColorRed    = "\033[31m"
	ColorGreen  = "\033[32m"
	ColorYellow = "\033[33m"
	ColorReset  = "\033[0m"
)

// PrecheckResult 表示预检查结果（数据层DTO）
type PrecheckResult struct {
	Hostname     string `json:"hostname"`
	HCA          string `json:"hca"`
	PhysState    string `json:"phys_state"`
	State        string `json:"state"`
	Speed        string `json:"speed"`
	FwVer        string `json:"fw_ver"`
	BoardId      string `json:"board_id"`
	IsHealthy    bool   `json:"is_healthy"`
	SerialNumber string `json:"serial_number"`
	Error        string `json:"error"`
}
