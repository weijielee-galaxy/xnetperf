import { useState, useEffect } from 'react'
import {
  Box,
  Tabs,
  TabList,
  TabPanels,
  Tab,
  TabPanel,
  Heading,
  useToast,
} from '@chakra-ui/react'
import ConfigPage from './pages/ConfigPage'
import DictionaryPage from './pages/DictionaryPage'
import TrafficTestPage from './pages/TrafficTestPage'
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

      {/* Tabs Navigation */}
      <Tabs colorScheme="blue" h="calc(100vh - 64px)" display="flex" flexDirection="column">
        <TabList px={4} bg="white" borderBottomWidth="1px">
          <Tab>配置管理</Tab>
          <Tab>字典管理</Tab>
          <Tab>流量测试</Tab>
        </TabList>

        <TabPanels flex="1" overflow="hidden">
          <TabPanel p={0} h="100%">
            <ConfigPage
              configs={configs}
              currentConfig={currentConfig}
              configData={configData}
              originalData={originalData}
              loading={loading}
              onConfigSelect={selectConfig}
              onRefresh={loadConfigs}
              onConfigCreate={loadConfigs}
              onConfigDelete={loadConfigs}
              onConfigUpdate={() => {
                loadConfigs()
                selectConfig(currentConfig)
              }}
              onConfigChange={setConfigData}
            />
          </TabPanel>

          <TabPanel p={0} h="100%">
            <DictionaryPage />
          </TabPanel>

          <TabPanel p={0} h="100%">
            <TrafficTestPage configs={configs} />
          </TabPanel>
        </TabPanels>
      </Tabs>
    </Box>
  )
}

export default App
