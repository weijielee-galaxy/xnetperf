# Latency Testing Guide

## Overview

The `xnetperf lat` command provides comprehensive network latency testing capabilities for InfiniBand/RDMA networks. Unlike bandwidth testing (which uses `ib_write_bw`), latency testing uses `ib_write_lat` to measure the time it takes for a single message to travel between network endpoints.

## Key Features

- **N×N Full-Mesh Testing**: Measures latency between all HCA pairs across all hosts
- **Automated Workflow**: Generate → Run → Monitor → Collect → Analyze in one command
- **JSON Report Format**: Structured output for easy parsing and analysis
- **Statistical Analysis**: Provides min/max/avg latency statistics
- **Matrix Display**: Shows latency measurements in an easy-to-read format

## Quick Start

### Basic Usage

```bash
# Run latency test with default config
xnetperf lat

# Run with custom config file
xnetperf lat -c /path/to/config.yaml
```

## Configuration Requirements

### Recommended Settings for Latency Testing

In your `config.yaml`, configure the following settings:

```yaml
stream_type: fullmesh  # Required: latency testing only supports fullmesh mode

run:
  infinitely: false
  duration_seconds: 5  # Recommended: 5 seconds for latency tests

report:
  enable: true
  dir: reports

# Other parameters (queue_pair_num, message_size) are ignored for latency tests
# ib_write_lat measures single-message latency without these parameters
```

### Important Notes

1. **Stream Type**: Currently, latency testing only supports `fullmesh` mode. If your config specifies a different `stream_type` (e.g., `incast`, `p2p`), a warning will be displayed, but testing will continue.

2. **Duration**: For latency tests, a shorter duration (5-10 seconds) is typically sufficient to get stable measurements, unlike bandwidth tests which may run for 20+ seconds.

3. **Test Parameters**: The following parameters used in bandwidth testing are automatically excluded for latency tests:
   - `queue_pair_num` (-q): Not applicable for latency
   - `message_size` (-m): Not applicable for latency

## Workflow Steps

The `xnetperf lat` command executes the following automated workflow:

### Step 1: Generate Scripts
Generates bash scripts for N×N latency testing between all HCA pairs.

**What happens:**
- Creates server scripts (ib_write_lat in server mode)
- Creates client scripts (ib_write_lat connecting to servers)
- Calculates port allocation for N×N connections
- Validates port availability

**Output:**
```
📋 Step 1/5: Generating latency test scripts...
Total ports needed for latency testing: 24
Host host1 IP: 192.168.1.10
✅ Generated latency scripts for host1:mlx5_0
...
✅ Successfully generated latency test scripts in generated_scripts_latency
```

### Step 2: Run Tests
Executes the generated scripts using a two-phase startup:
1. Start all server processes (with initialization delay)
2. Start all client processes (after servers are ready)

**What happens:**
- SSH to each host and execute server scripts
- Wait for servers to initialize
- SSH to each host and execute client scripts
- Tests run for the configured duration

### Step 3: Monitor Progress
Monitors `ib_write_lat` processes across all hosts.

**What happens:**
- Checks process status every 5 seconds
- Displays progress information
- Waits until all tests complete

### Step 4: Collect Reports
Collects JSON report files from all remote hosts.

**What happens:**
- SCP report files from remote hosts to local `reports/` directory
- Cleans up remote report files
- Validates collected files

### Step 5: Analyze Results
Parses JSON reports and displays latency matrix.

**What happens:**
- Parses `t_avg` (average latency in microseconds) from each report
- Groups measurements by source host and HCA
- Calculates statistics (min/max/avg)
- Displays formatted matrix output

## Understanding the Output

### Latency Matrix Example

```
================================================================================
📊 Latency Matrix (Average Latency in microseconds)
================================================================================
┌──────────────────────┬──────────────┬──────────────┬──────────────┐
│ Source → Target      │ host1:mlx5_0 │ host2:mlx5_0 │ host3:mlx5_0 │
├──────────────────────┼──────────────┼──────────────┼──────────────┤
│ host1:mlx5_0         │ -            │      1.45 μs │      1.48 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┤
│ host2:mlx5_0         │      1.46 μs │ -            │      1.50 μs │
├──────────────────────┼──────────────┼──────────────┼──────────────┤
│ host3:mlx5_0         │      1.47 μs │      1.51 μs │ -            │
└──────────────────────┴──────────────┴──────────────┴──────────────┘

================================================================================
📈 Latency Statistics:
  Minimum Latency: 1.45 μs
  Maximum Latency: 1.51 μs
  Average Latency: 1.48 μs
  Total Measurements: 6
================================================================================
```

**Note:** The table shows a symmetric N×N matrix where:
- Each row represents a source HCA
- Each column represents a target HCA
- Values show the average latency in microseconds
- Self-connections (diagonal) are marked with `-`
- Missing measurements are also shown as `-`

### Understanding Latency Values

- **Typical InfiniBand Latency**: 1-2 microseconds for NDR (200 Gbps)
- **Good Latency**: < 2 μs
- **Acceptable Latency**: 2-5 μs  
- **High Latency**: > 5 μs (may indicate network issues)

If you see high latency values:
1. Check for network congestion
2. Verify cable quality and connections
3. Check HCA firmware versions
4. Ensure proper network configuration (MTU, flow control, etc.)

## Command Architecture

### Generated Commands

**Server Command Example:**
```bash
ssh host1 'ib_write_lat -d mlx5_0 -D 5 -p 20000 \
  --output-format json --out_json_file /path/latency_s_host1_mlx5_0_20000.json \
  >/dev/null 2>&1 &'; sleep 0.02
```

**Client Command Example:**
```bash
ssh host2 'ib_write_lat -d mlx5_0 -D 5 -p 20000 192.168.1.10 \
  --output-format json --out_json_file /path/latency_c_host2_mlx5_0_20000.json \
  >/dev/null 2>&1 &'; sleep 0.06
```

### Key Differences from Bandwidth Testing

| Aspect | Bandwidth (`ib_write_bw`) | Latency (`ib_write_lat`) |
|--------|---------------------------|--------------------------|
| Command | `ib_write_bw` | `ib_write_lat` |
| Metric | Throughput (Gbps) | Latency (μs) |
| Queue Pairs | Multiple (-q flag) | Single (no -q flag) |
| Message Size | Configurable (-m flag) | Single message (no -m flag) |
| Duration | 20-30 seconds typical | 5-10 seconds sufficient |
| Output | Bandwidth matrix | Latency matrix |
| JSON Field | `bw_avg_value` | `t_avg` |

## Port Calculation

For N hosts with H HCAs each, the total ports needed:
```
Total Ports = N × H × (N-1) × H
```

**Example:**
- 3 hosts, 2 HCAs each: 3 × 2 × 2 × 2 = 24 ports
- 10 hosts, 4 HCAs each: 10 × 4 × 9 × 4 = 1440 ports

The command automatically validates that sufficient ports are available (default range: 20000-65535).

## Troubleshooting

### Error: Not enough ports available

**Problem:**
```
Error: Not enough ports available. Required: 1500, Available: 1000
```

**Solution:**
- Reduce the number of hosts or HCAs in your config
- Increase the available port range by lowering `start_port` in config

### Warning: Stream type not fullmesh

**Problem:**
```
⚠️  Warning: Latency testing currently only supports fullmesh mode. Current mode: incast
```

**Solution:**
This is a warning, not an error. Testing will continue, but results may not be comprehensive. To fix:
```yaml
stream_type: fullmesh  # Change to fullmesh for proper latency testing
```

### No latency reports found

**Problem:**
```
❌ Error collecting latency report data: no latency reports found in reports
```

**Solution:**
1. Check that `report.enable: true` in config
2. Verify tests actually ran: `ps aux | grep ib_write_lat`
3. Check remote hosts have `ib_write_lat` installed
4. Review SSH connectivity and permissions

### High or inconsistent latency values

**Problem:**
Latency values > 10 μs or highly variable

**Solution:**
1. Check for network congestion: `ibv_devinfo` and `ibstat`
2. Verify MTU settings match across all devices
3. Check for packet loss: `ibdiagnet` or similar tools
4. Ensure no other traffic during testing
5. Verify CPU affinity and interrupt handling

## Advanced Usage

### Manual Step-by-Step Execution

If you prefer to run each step manually instead of using the automated workflow:

```bash
# Step 1: Generate scripts only (not yet implemented as standalone command)
# Note: Use 'xnetperf lat' for now

# Step 2: Review generated scripts
ls generated_scripts_latency/
cat generated_scripts_latency/host1_mlx5_0_server_latency.sh

# Step 3: Run tests manually
bash generated_scripts_latency/host1_mlx5_0_server_latency.sh
bash generated_scripts_latency/host1_mlx5_0_client_latency.sh

# Step 4: Collect reports
xnetperf collect --cleanup

# Step 5: Analyze results (not yet implemented as standalone for latency)
# Note: Use 'xnetperf lat' for now to get full workflow
```

### Integration with CI/CD

For automated latency testing in CI/CD pipelines:

```bash
#!/bin/bash
# latency_ci_test.sh

set -e

# Run latency test
xnetperf lat -c production_config.yaml

# Check if latency exceeds threshold (5 μs)
MAX_LATENCY=$(grep "Maximum Latency:" output.log | awk '{print $3}')
if (( $(echo "$MAX_LATENCY > 5.0" | bc -l) )); then
    echo "❌ Latency test failed: $MAX_LATENCY μs exceeds 5.0 μs threshold"
    exit 1
fi

echo "✅ Latency test passed: $MAX_LATENCY μs within acceptable range"
```

## JSON Report Format

Example `latency_c_host1_mlx5_0_20000.json`:

```json
{
  "results": [
    {
      "t_min": 0.85,
      "t_max": 2.15,
      "t_avg": 1.45,
      "t_stdev": 0.12,
      "t_typical": 1.43,
      "t_99percentile": 1.98,
      "t_99_9percentile": 2.10
    }
  ]
}
```

Currently, `xnetperf lat` extracts the `t_avg` field, but the JSON contains rich statistical data that can be used for advanced analysis.

## Comparison with Bandwidth Testing

| Use Case | Command | Purpose |
|----------|---------|---------|
| **Throughput Testing** | `xnetperf execute` | Measure maximum bandwidth (Gbps) |
| **Latency Testing** | `xnetperf lat` | Measure message latency (μs) |
| **Complete Testing** | Both commands | Full network characterization |

**Best Practice:** Run both bandwidth and latency tests for comprehensive network validation:

```bash
# First, test bandwidth
xnetperf execute -c config.yaml

# Then, test latency
xnetperf lat -c config.yaml
```

## Version History

- **v0.2.0** (2024): Initial latency testing implementation
  - N×N full-mesh latency testing
  - Automated workflow
  - JSON report parsing
  - Matrix display with statistics

## Future Enhancements

Planned improvements for latency testing:

1. **Target Identification**: Currently, target host/HCA info is marked as "unknown" in some displays. Future versions will extract this from test metadata.

2. **Additional Stream Types**: Support for `incast` and `p2p` latency testing modes.

3. **Bidirectional Testing**: Measure latency in both directions simultaneously.

4. **Histogram Display**: Visualize latency distribution with histograms.

5. **Comparison Mode**: Compare latency results across multiple test runs.

6. **Alert Thresholds**: Configurable thresholds with automatic pass/fail detection.

7. **Web UI Integration**: Display latency matrix in the web interface.

## See Also

- [Traffic Test Guide](traffic-test-guide.md) - Bandwidth testing guide
- [Config Validation](config-validation.md) - Configuration file guide
- [Generate Command](generate-command.md) - Script generation details
