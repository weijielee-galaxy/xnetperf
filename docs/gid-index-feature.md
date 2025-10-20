# GID Index Configuration Feature

## Overview
Added support for configuring GID (Global Identifier) index for RoCE v2 testing via the `-x` parameter in `ib_write_bw` commands.

## Configuration

### YAML Configuration
Add `gid_index` field to your `config.yaml`:

```yaml
gid_index: 3  # Default value is 3
rdma_cm: false
network_interface: ""
```

### Default Value
- **Default**: `3`
- If not specified or set to `0`, the system will automatically use `3`

## Implementation Details

### Config Structure
```go
type Config struct {
    // ... other fields
    GidIndex          int           `yaml:"gid_index" json:"gid_index"`
    RdmaCm            bool          `yaml:"rdma_cm" json:"rdma_cm"`
    NetworkInterface  string        `yaml:"network_interface" json:"network_interface"`
}
```

### Command Builder Support
The `IBWriteBWCommandBuilder` supports GID index through the `.GidIndex()` method:

```go
cmd := NewIBWriteBWCommandBuilder().
    Device("mlx5_0").
    Port(20000).
    GidIndex(3).
    ServerCommand()
```

### Generated Commands
With `gid_index: 3` in config, generated commands will include `-x 3`:

**Server Command:**
```bash
ib_write_bw -d mlx5_0 -D 20 -q 10 -m 4096 -p 20000 -b -x 3 --report_gbits --out_json --out_json_file /root/report.json
```

**Client Command:**
```bash
ib_write_bw -d mlx5_0 -D 20 -q 10 -m 4096 -p 20001 -b -x 3 192.168.1.100 --report_gbits
```

## Supported Modes
GID index configuration is supported in all three testing modes:
1. **FullMesh** (`stream/stream_fullmesh.go`)
2. **Incast** (`stream/stream_incast.go`)
3. **P2P** (`stream/stream_p2p.go`)

## Relationship with Other Parameters

### Independent Parameters
`gid_index` is **independent** from other RDMA parameters and can be used together:

| Parameter | Flag | Config Field | Purpose |
|-----------|------|--------------|---------|
| GID Index | `-x` | `gid_index` | Specifies GID index for RoCE v2 |
| RDMA CM | `-R` | `rdma_cm` | Use RDMA Connection Manager |
| Bidirectional | `-b` | (always enabled) | Bidirectional testing |

### Example: Using Both gid_index and rdma_cm
```yaml
gid_index: 3
rdma_cm: true
```

Generated command will include both flags:
```bash
ib_write_bw -d mlx5_0 -D 20 -q 10 -m 4096 -p 20000 -b -R -x 3 --report_gbits
```

## Testing

### Unit Tests
Run GID index tests:
```bash
go test -v ./stream -run TestGidIndex
```

### Test Coverage
- ✅ GID index parameter generation (`-x 3`)
- ✅ Zero value handling (no `-x` flag when `gid_index: 0`)
- ✅ Different GID index values
- ✅ Combination with `rdma_cm` parameter
- ✅ Full command validation with SSH and all parameters

## Usage Example

### 1. Configure config.yaml
```yaml
clients:
  - host: server1
    hca_device: mlx5_0
  - host: server2
    hca_device: mlx5_1

servers:
  - host: server1
    hca_device: mlx5_0

test_type: fullmesh
gid_index: 3  # Configure GID index here
duration: 20
queue_pair_num: 10
message_size: 4096
```

### 2. Generate Scripts
```bash
./xnetperf generate
```

### 3. Verify Generated Scripts
Check that scripts contain `-x 3` parameter:
```bash
head generated_scripts_fullmesh/*_server_fullmesh.sh
```

Expected output:
```bash
ib_write_bw -d mlx5_0 -D 10 -q 10 -m 4096 -p 20000 -x 3 >/dev/null 2>&1 &
```

## Technical Notes

### Why GID Index Matters
- GID (Global Identifier) is used in RoCE v2 for network routing
- Different GID indices correspond to different IP addresses on the HCA
- Index 3 is commonly used for RoCE v2 configurations
- Incorrect GID index may cause connection failures

### When to Change Default
Change from default `3` when:
- Your HCA has different GID configuration
- Network topology requires specific GID index
- Troubleshooting RoCE v2 connectivity issues

### Verification
Use `show_gids` to check available GID indices on your HCA:
```bash
show_gids | grep mlx5_0
```

## Files Modified
1. **config/config.go** - Added `GidIndex` field and default value
2. **stream/command_builder.go** - Added GID index support in builder
3. **stream/stream_fullmesh.go** - Applied GID index in fullmesh mode
4. **stream/stream_incast.go** - Applied GID index in incast mode
5. **stream/stream_p2p.go** - Applied GID index in p2p mode
6. **stream/command_builder_gid_test.go** - Unit tests for GID index

## Version
Added in: v0.1.3 (or appropriate version)
