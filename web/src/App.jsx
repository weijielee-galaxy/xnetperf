import { useState, useEffect } from 'react'
import {
  Box,
  Flex,
  Heading,
  useToast,
} from '@chakra-ui/react'
import ConfigList from './components/ConfigList'
import ConfigEditor from './components/ConfigEditor'
import { fetchConfigs, fetchConfig } from './api'

function App() {
  const [configs, setConfigs] = useState([])
  const [currentConfig, setCurrentConfig] = useState(null)
  const [configData, setConfigData] = useState(null)
  const [originalData, setOriginalData] = useState(null)
  const [loading, setLoading] = useState(false)
  const toast = useToast()

  // 加载配置列表
  const loadConfigs = async () => {
    try {
      const data = await fetchConfigs()
      setConfigs(data)
    } catch (error) {
      toast({
        title: '加载配置列表失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  // 选择配置
  const selectConfig = async (name) => {
    try {
      setLoading(true)
      const data = await fetchConfig(name)
      setCurrentConfig(name)
      setConfigData(data)
      setOriginalData(JSON.parse(JSON.stringify(data)))
    } catch (error) {
      toast({
        title: '加载配置失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  // 初始化
  useEffect(() => {
    loadConfigs()
  }, [])

  return (
    <Box minH="100vh" bg="gray.50">
      {/* Header */}
      <Box bg="blue.500" color="white" px={6} py={4} boxShadow="sm">
        <Heading size="md">xnetperf 配置管理</Heading>
      </Box>

      {/* Main Content */}
      <Flex h="calc(100vh - 64px)">
        {/* Sidebar */}
        <ConfigList
          configs={configs}
          currentConfig={currentConfig}
          onSelect={selectConfig}
          onRefresh={loadConfigs}
        />

        {/* Content Area */}
        <ConfigEditor
          currentConfig={currentConfig}
          configData={configData}
          originalData={originalData}
          loading={loading}
          onSave={() => {
            loadConfigs()
            selectConfig(currentConfig)
          }}
          onCancel={() => {
            setConfigData(JSON.parse(JSON.stringify(originalData)))
          }}
          onChange={setConfigData}
        />
      </Flex>
    </Box>
  )
}

export default App
