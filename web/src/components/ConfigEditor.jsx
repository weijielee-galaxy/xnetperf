import {
  Box,
  Button,
  HStack,
  VStack,
  Heading,
  FormControl,
  FormLabel,
  Input,
  Select,
  Switch,
  NumberInput,
  NumberInputField,
  Tag,
  TagLabel,
  TagCloseButton,
  Wrap,
  WrapItem,
  IconButton,
  useToast,
  Spinner,
  Center,
  Text,
} from '@chakra-ui/react'
import { AddIcon } from '@chakra-ui/icons'
import { useState } from 'react'
import { updateConfig, validateConfig } from '../api'

function ConfigEditor({ currentConfig, configData, originalData, loading, onSave, onCancel, onChange }) {
  const [saving, setSaving] = useState(false)
  const [validating, setValidating] = useState(false)
  const toast = useToast()

  // 空状态
  if (!currentConfig) {
    return (
      <Center flex={1} flexDirection="column" color="gray.500">
        <Text fontSize="4xl" mb={4}>📝</Text>
        <Text>请在左侧选择或创建一个配置文件</Text>
      </Center>
    )
  }

  // 加载状态
  if (loading || !configData) {
    return (
      <Center flex={1}>
        <Spinner size="xl" color="blue.500" />
      </Center>
    )
  }

  // 更新字段
  const updateField = (field, value) => {
    onChange({ ...configData, [field]: value })
  }

  // 更新嵌套字段
  const updateNestedField = (parent, field, value) => {
    onChange({
      ...configData,
      [parent]: { ...configData[parent], [field]: value },
    })
  }

  // 添加标签
  const addTag = (parent, field) => {
    const input = prompt(`添加 ${field}:`)
    if (input && input.trim()) {
      const current = configData[parent][field] || []
      onChange({
        ...configData,
        [parent]: {
          ...configData[parent],
          [field]: [...current, input.trim()],
        },
      })
    }
  }

  // 删除标签
  const removeTag = (parent, field, index) => {
    const current = configData[parent][field] || []
    onChange({
      ...configData,
      [parent]: {
        ...configData[parent],
        [field]: current.filter((_, i) => i !== index),
      },
    })
  }

  // 保存配置
  const handleSave = async () => {
    try {
      setSaving(true)
      await updateConfig(currentConfig, configData)
      toast({
        title: '保存成功！',
        status: 'success',
        duration: 2000,
      })
      onSave()
    } catch (error) {
      toast({
        title: '保存失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setSaving(false)
    }
  }

  // 验证配置
  const handleValidate = async () => {
    try {
      setValidating(true)
      await validateConfig(currentConfig)
      toast({
        title: '✓ 配置验证通过！',
        status: 'success',
        duration: 2000,
      })
    } catch (error) {
      toast({
        title: '✗ 配置验证失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setValidating(false)
    }
  }

  return (
    <Box flex={1} display="flex" flexDirection="column">
      {/* Header */}
      <HStack p={4} borderBottom="1px" borderColor="gray.200" justify="space-between" bg="white">
        <Heading size="md">{currentConfig}</Heading>
        <HStack spacing={2}>
          <Button size="sm" onClick={handleValidate} isLoading={validating}>
            ✓ 验证配置
          </Button>
          <Button size="sm" variant="ghost" onClick={onCancel}>
            ✕ 取消
          </Button>
          <Button size="sm" colorScheme="green" onClick={handleSave} isLoading={saving}>
            💾 保存
          </Button>
        </HStack>
      </HStack>

      {/* Form */}
      <Box flex={1} overflowY="auto" p={6}>
        <VStack spacing={8} align="stretch">
          {/* 基础配置 */}
          <Box>
            <Heading size="sm" mb={4} pb={2} borderBottom="2px" borderColor="blue.500">
              基础配置
            </Heading>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>起始端口 (start_port)</FormLabel>
                <NumberInput
                  value={configData.start_port || 0}
                  min={1}
                  max={65535}
                  onChange={(_, val) => updateField('start_port', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel>流类型 (stream_type)</FormLabel>
                <Select
                  value={configData.stream_type || ''}
                  onChange={(e) => updateField('stream_type', e.target.value)}
                >
                  <option value="fullmesh">FullMesh</option>
                  <option value="incast">InCast</option>
                  <option value="p2p">P2P</option>
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel>队列对数量 (qp_num)</FormLabel>
                <NumberInput
                  value={configData.qp_num || 0}
                  min={1}
                  onChange={(_, val) => updateField('qp_num', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel>消息大小/字节 (message_size_bytes)</FormLabel>
                <NumberInput
                  value={configData.message_size_bytes || 0}
                  min={1}
                  onChange={(_, val) => updateField('message_size_bytes', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel>输出目录 (output_base)</FormLabel>
                <Input
                  value={configData.output_base || ''}
                  onChange={(e) => updateField('output_base', e.target.value)}
                />
              </FormControl>

              <FormControl>
                <FormLabel>等待时间/秒 (waiting_time_seconds)</FormLabel>
                <NumberInput
                  value={configData.waiting_time_seconds || 0}
                  min={0}
                  onChange={(_, val) => updateField('waiting_time_seconds', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel>速度/Gbps (speed)</FormLabel>
                <NumberInput
                  value={configData.speed || 0}
                  min={0}
                  onChange={(_, val) => updateField('speed', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>使用 RDMA CM (rdma_cm)</FormLabel>
                <Switch
                  isChecked={configData.rdma_cm || false}
                  onChange={(e) => updateField('rdma_cm', e.target.checked)}
                />
              </FormControl>
            </VStack>
          </Box>

          {/* 报告配置 */}
          <Box>
            <Heading size="sm" mb={4} pb={2} borderBottom="2px" borderColor="blue.500">
              报告配置 (report)
            </Heading>
            <VStack spacing={4} align="stretch">
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>启用报告 (enable)</FormLabel>
                <Switch
                  isChecked={configData.report?.enable || false}
                  onChange={(e) => updateNestedField('report', 'enable', e.target.checked)}
                />
              </FormControl>

              <FormControl>
                <FormLabel>报告目录 (dir)</FormLabel>
                <Input
                  value={configData.report?.dir || ''}
                  onChange={(e) => updateNestedField('report', 'dir', e.target.value)}
                />
              </FormControl>
            </VStack>
          </Box>

          {/* 运行配置 */}
          <Box>
            <Heading size="sm" mb={4} pb={2} borderBottom="2px" borderColor="blue.500">
              运行配置 (run)
            </Heading>
            <VStack spacing={4} align="stretch">
              <FormControl display="flex" alignItems="center">
                <FormLabel mb={0}>无限运行 (infinitely)</FormLabel>
                <Switch
                  isChecked={configData.run?.infinitely || false}
                  onChange={(e) => updateNestedField('run', 'infinitely', e.target.checked)}
                />
              </FormControl>

              <FormControl>
                <FormLabel>运行时长/秒 (duration_seconds)</FormLabel>
                <NumberInput
                  value={configData.run?.duration_seconds || 0}
                  min={0}
                  onChange={(_, val) => updateNestedField('run', 'duration_seconds', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>
            </VStack>
          </Box>

          {/* 服务器配置 */}
          <Box>
            <Heading size="sm" mb={4} pb={2} borderBottom="2px" borderColor="blue.500">
              服务器配置 (server)
            </Heading>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>主机名列表 (hostname)</FormLabel>
                <Wrap spacing={2} mb={2}>
                  {(configData.server?.hostname || []).map((host, index) => (
                    <WrapItem key={index}>
                      <Tag size="md" colorScheme="blue" variant="solid">
                        <TagLabel>{host}</TagLabel>
                        <TagCloseButton onClick={() => removeTag('server', 'hostname', index)} />
                      </Tag>
                    </WrapItem>
                  ))}
                </Wrap>
                <Button
                  size="sm"
                  leftIcon={<AddIcon />}
                  onClick={() => addTag('server', 'hostname')}
                >
                  添加
                </Button>
              </FormControl>

              <FormControl>
                <FormLabel>HCA 列表 (hca)</FormLabel>
                <Wrap spacing={2} mb={2}>
                  {(configData.server?.hca || []).map((hca, index) => (
                    <WrapItem key={index}>
                      <Tag size="md" colorScheme="green" variant="solid">
                        <TagLabel>{hca}</TagLabel>
                        <TagCloseButton onClick={() => removeTag('server', 'hca', index)} />
                      </Tag>
                    </WrapItem>
                  ))}
                </Wrap>
                <Button
                  size="sm"
                  leftIcon={<AddIcon />}
                  onClick={() => addTag('server', 'hca')}
                >
                  添加
                </Button>
              </FormControl>
            </VStack>
          </Box>

          {/* 客户端配置 */}
          <Box>
            <Heading size="sm" mb={4} pb={2} borderBottom="2px" borderColor="blue.500">
              客户端配置 (client)
            </Heading>
            <VStack spacing={4} align="stretch">
              <FormControl>
                <FormLabel>主机名列表 (hostname)</FormLabel>
                <Wrap spacing={2} mb={2}>
                  {(configData.client?.hostname || []).map((host, index) => (
                    <WrapItem key={index}>
                      <Tag size="md" colorScheme="purple" variant="solid">
                        <TagLabel>{host}</TagLabel>
                        <TagCloseButton onClick={() => removeTag('client', 'hostname', index)} />
                      </Tag>
                    </WrapItem>
                  ))}
                </Wrap>
                <Button
                  size="sm"
                  leftIcon={<AddIcon />}
                  onClick={() => addTag('client', 'hostname')}
                >
                  添加
                </Button>
              </FormControl>

              <FormControl>
                <FormLabel>HCA 列表 (hca)</FormLabel>
                <Wrap spacing={2} mb={2}>
                  {(configData.client?.hca || []).map((hca, index) => (
                    <WrapItem key={index}>
                      <Tag size="md" colorScheme="orange" variant="solid">
                        <TagLabel>{hca}</TagLabel>
                        <TagCloseButton onClick={() => removeTag('client', 'hca', index)} />
                      </Tag>
                    </WrapItem>
                  ))}
                </Wrap>
                <Button
                  size="sm"
                  leftIcon={<AddIcon />}
                  onClick={() => addTag('client', 'hca')}
                >
                  添加
                </Button>
              </FormControl>
            </VStack>
          </Box>
        </VStack>
      </Box>
    </Box>
  )
}

export default ConfigEditor
