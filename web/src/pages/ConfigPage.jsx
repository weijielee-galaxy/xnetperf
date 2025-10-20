import { Box, Flex } from '@chakra-ui/react'
import ConfigList from '../components/ConfigList'
import ConfigEditor from '../components/ConfigEditor'

function ConfigPage({ 
  configs, 
  currentConfig, 
  configData, 
  originalData,
  loading,
  onConfigSelect,
  onRefresh,
  onConfigCreate,
  onConfigDelete,
  onConfigUpdate,
  onConfigChange
}) {
  return (
    <Flex h="100vh" overflow="hidden">
      <ConfigList
        configs={configs}
        currentConfig={currentConfig}
        onSelect={onConfigSelect}
        onRefresh={onRefresh}
      />
      <Box flex="1" overflow="auto">
        <ConfigEditor
          configData={configData}
          originalData={originalData}
          currentConfig={currentConfig}
          loading={loading}
          onSave={onConfigUpdate}
          onCancel={() => onConfigChange(JSON.parse(JSON.stringify(originalData)))}
          onChange={onConfigChange}
        />
      </Box>
    </Flex>
  )
}

export default ConfigPage
