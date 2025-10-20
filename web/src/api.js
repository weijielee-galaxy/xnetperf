// API 工具函数

const API_BASE = '/api'

// 统一的 fetch 封装
async function fetchJSON(url, options = {}) {
  const response = await fetch(url, options)
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  
  const text = await response.text()
  if (!text || text.trim() === '') {
    throw new Error('Empty response from server')
  }
  
  const result = JSON.parse(text)
  
  if (result.code !== 0) {
    throw new Error(result.message || 'Unknown error')
  }
  
  return result.data
}

// 获取配置列表
export async function fetchConfigs() {
  return await fetchJSON(`${API_BASE}/configs`)
}

// 获取指定配置
export async function fetchConfig(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}`)
}

// 创建配置
export async function createConfig(name, config) {
  return await fetchJSON(`${API_BASE}/configs`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ name, config }),
  })
}

// 更新配置
export async function updateConfig(name, config) {
  return await fetchJSON(`${API_BASE}/configs/${name}`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(config),
  })
}

// 删除配置
export async function deleteConfig(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}`, {
    method: 'DELETE',
  })
}

// 验证配置（特殊处理：返回完整响应包含错误详情）
export async function validateConfig(name) {
  const response = await fetch(`${API_BASE}/configs/${name}/validate`, {
    method: 'POST',
  })
  
  if (!response.ok) {
    throw new Error(`HTTP error! status: ${response.status}`)
  }
  
  const text = await response.text()
  if (!text || text.trim() === '') {
    throw new Error('Empty response from server')
  }
  
  const result = JSON.parse(text)
  
  // 验证接口特殊处理：即使 code !== 0 也返回完整 data
  if (result.code !== 0 && result.data && result.data.errors) {
    const error = new Error(result.message || 'Validation failed')
    error.errors = result.data.errors // 附加详细错误列表
    throw error
  }
  
  if (result.code !== 0) {
    throw new Error(result.message || 'Unknown error')
  }
  
  return result.data
}

// 预览配置
export async function previewConfig(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/preview`)
}

// 获取主机名列表
export async function fetchHostnames() {
  return await fetchJSON(`${API_BASE}/dictionary/hostnames`)
}

// 更新主机名列表
export async function updateHostnames(hostnames) {
  return await fetchJSON(`${API_BASE}/dictionary/hostnames`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hostnames }),
  })
}

// 获取 HCA 列表
export async function fetchHCAs() {
  return await fetchJSON(`${API_BASE}/dictionary/hcas`)
}

// 更新 HCA 列表
export async function updateHCAs(hcas) {
  return await fetchJSON(`${API_BASE}/dictionary/hcas`, {
    method: 'PUT',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ hcas }),
  })
}

// 执行 Precheck 检查
export async function runPrecheck(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/precheck`, {
    method: 'POST',
  })
}

// 运行测试
export async function runTest(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/run`, {
    method: 'POST',
  })
}

// 探测测试状态
export async function probeTest(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/probe`, {
    method: 'POST',
  })
}

// 收集报告
export async function collectReports(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/collect`, {
    method: 'POST',
  })
}

// 获取性能报告
export async function getReport(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/report`)
}
