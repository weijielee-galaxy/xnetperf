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

// 验证配置
export async function validateConfig(name) {
  return await fetchJSON(`${API_BASE}/configs/${name}/validate`, {
    method: 'POST',
  })
}
