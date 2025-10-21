# Latency Color Marking Feature

## Overview
为延迟测试结果显示添加了视觉增强功能，使用红色标记异常延迟值和缺失数据，帮助用户快速识别网络问题。

## Feature Description

### Visual Indicators
延迟矩阵中的数据根据以下规则进行颜色标记：

1. **高延迟标记（红色）**：
   - 当延迟 > 4.0μs 时，该值会以红色显示
   - 适用于 fullmesh 和 incast 两种模式
   - 阈值可通过 `latencyThreshold` 常量配置

2. **自己到自己的连接（显示 `-`）**：
   - 在 fullmesh 模式中，同一个 host:hca 之间的连接（对角线位置）
   - 显示 `-` 符号，**不标红色**（这是正常现象，因为延迟测试不测自己）

3. **测试失败/缺失数据标记（红色 `*`）**：
   - 当测试执行了但未能获取到数据时，显示红色的 `*`
   - 表示网络连通性问题或测试失败
   - 在 incast 模式中，所有缺失数据都标记为红色 `*`（因为 client 和 server 是分开的，不存在"自己到自己"的情况）

### Implementation Details

#### Constants and Configuration
```go
const (
    colorRed         = "\033[31m"
    colorReset       = "\033[0m"
    latencyThreshold = 4.0  // Threshold in microseconds
)
```

#### Color Marking Logic

**Fullmesh Mode (displayLatencyMatrix)**:
```go
if latency > 0 {
    valueStr := fmt.Sprintf("%.2f μs", latency)
    if latency > latencyThreshold {
        // High latency - mark in red
        fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, valueStr, colorReset)
    } else {
        // Normal latency
        fmt.Printf(" %*s │", valueColWidth, valueStr)
    }
} else {
    // Check if this is self-to-self (diagonal)
    if sourceHost == targetHost && sourceHCA == targetHCA {
        // Self-to-self: display "-" without red color
        fmt.Printf(" %*s │", valueColWidth, "-")
    } else {
        // Missing data: display red "*" to indicate test failure/unreachable
        fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, "*", colorReset)
    }
}
```

**Incast Mode (displayLatencyMatrixIncast)**:
```go
if latency > 0 {
    valueStr := fmt.Sprintf("%.2f μs", latency)
    if latency > latencyThreshold {
        // High latency - mark in red
        fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, valueStr, colorReset)
    } else {
        // Normal latency
        fmt.Printf(" %*s │", valueColWidth, valueStr)
    }
} else {
    // In incast mode, client and server are separate, so missing data is always a failure
    // Display red "*" to indicate test failure/unreachable
    fmt.Printf(" %s%*s%s │", colorRed, valueColWidth, "*", colorReset)
}
```

## Test Coverage

### Unit Tests
创建了完整的单元测试套件 (`cmd/lat_color_test.go`)，包含以下测试用例：

#### 1. Fullmesh Mode Tests (`TestDisplayLatencyMatrixWithColorMarking`)
- **Mixed latencies**: 测试混合正常和高延迟值，以及缺失数据的 `*` 标记
- **All high latencies**: 测试所有延迟都高于阈值的情况
- **All normal latencies**: 测试所有延迟都正常的情况
- **Threshold boundary test**: 测试阈值边界情况（3.99μs, 4.00μs, 4.01μs）
- **Self-to-self connections**: 测试对角线位置显示 `-` 而不是红色标记

#### 2. Incast Mode Tests (`TestDisplayLatencyMatrixIncastWithColorMarking`)
- **Mixed latencies**: 测试 client-server 连接中的混合延迟
- **Missing data**: 测试缺失数据的红色 `*` 标记
- **Extreme latency values**: 测试极端高延迟值（50μs, 100μs）

#### 3. Visual Tests
- **TestColorConstants**: 验证颜色常量定义
- **TestLatencyValueFormatting**: 可视化显示不同延迟值的格式化效果

### Test Execution
```bash
# Run all color-related tests
go test ./cmd -run "Color|Formatting" -v

# Run specific test suites
go test ./cmd -run "TestDisplayLatencyMatrixWithColorMarking" -v
go test ./cmd -run "TestDisplayLatencyMatrixIncastWithColorMarking" -v
```

## Example Output

### Fullmesh Mode Example
```
================================================================================
📊 Latency Matrix (Average Latency in microseconds)
================================================================================
┌────────────┬────────────┬──────────────┬──────────────┐
│            │            │ host1        │ host2        │
│            │            ├──────────────┼──────────────┤
│            │            │ mlx5_0       │ mlx5_0       │
├────────────┼────────────┼──────────────┼──────────────┤
│ host1      │ mlx5_0     │            - │      2.50 μs │  ← "-" (self-to-self)
├────────────┼────────────┼──────────────┼──────────────┤
│ host2      │ mlx5_0     │      5.80 μs │            - │  ← Red (>4μs), "-" (self)
└────────────┴────────────┴──────────────┴──────────────┘

说明：
- "5.80 μs" 显示为红色（超过 4μs 阈值）
- "-" 显示为普通颜色（自己到自己，正常现象）
- 如果有测试失败的连接，会显示红色的 "*"
```

### Incast Mode Example
```
================================================================================
📊 Latency Matrix - INCAST Mode (Client → Server)
   Average Latency in microseconds
================================================================================
┌────────────┬────────────┬──────────────┬──────────────┐
│            │            │ server1      │ server2      │
│            │            ├──────────────┼──────────────┤
│            │            │ mlx5_0       │ mlx5_1       │
├────────────┼────────────┼──────────────┼──────────────┤
│ client1    │ mlx5_0     │      2.80 μs │            * │  ← Red "*" (test failed)
├────────────┼────────────┼──────────────┼──────────────┤
│ client2    │ mlx5_0     │      6.50 μs │     15.20 μs │  ← Both Red (>4μs)
└────────────┴────────────┴──────────────┴──────────────┘

说明：
- "6.50 μs" 和 "15.20 μs" 显示为红色（超过 4μs 阈值）
- 红色 "*" 表示测试失败或网络不可达
```

## Benefits

1. **快速识别问题**：红色标记让用户一眼就能看到问题区域
2. **减少分析时间**：无需手动扫描数值，异常值自动突出显示
3. **区分正常和异常缺失**：
   - `-` 表示自己到自己（正常，不需要测试）
   - 红色 `*` 表示测试失败或网络不可达（需要关注）
4. **可配置阈值**：通过 `latencyThreshold` 常量可调整标记阈值

## Future Enhancements

可能的改进方向：

1. **多级颜色标记**：
   - 黄色：警告级别（如 >3μs）
   - 红色：严重级别（如 >5μs）

2. **配置文件支持**：
   - 允许在配置文件中自定义阈值
   - 支持不同场景的不同阈值

3. **统计信息增强**：
   - 在统计区域显示异常延迟的数量
   - 提供异常连接的详细列表

## Related Files

- **Implementation**: `cmd/lat.go`
- **Tests**: `cmd/lat_color_test.go`
- **Documentation**: This file

## Version History

- **v0.2.1**: Initial implementation of color marking feature
  - Added red marking for latencies > 4μs
  - Added `-` for self-to-self connections (no color)
  - Added red `*` marking for test failures/unreachable connections
  - Comprehensive unit test coverage
  - Support for both fullmesh and incast modes
