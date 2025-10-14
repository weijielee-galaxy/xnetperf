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
  SimpleGrid,
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
      <HStack p={4} borderBottom="1px" borderColor="gray.200" justify="space-between" bg="white" shadow="sm">
        <HStack spacing={2}>
          <Text fontSize="lg" fontWeight="semibold" color="gray.700">📄</Text>
          <Heading size="md" color="gray.700">{currentConfig}</Heading>
        </HStack>
        <HStack spacing={2}>
          <Button size="sm" colorScheme="blue" variant="outline" onClick={handleValidate} isLoading={validating}>
            验证
          </Button>
          <Button size="sm" variant="ghost" onClick={onCancel}>
            取消
          </Button>
          <Button size="sm" colorScheme="green" onClick={handleSave} isLoading={saving}>
            保存
          </Button>
        </HStack>
      </HStack>

      {/* Form */}
      <Box flex={1} overflowY="auto" p={6} bg="gray.50">
        <VStack spacing={6} align="stretch">
          {/* 基础配置 */}
          <Box bg="white" p={6} borderRadius="lg" shadow="sm">
            <Heading size="md" mb={4} color="blue.600">
              基础配置
            </Heading>
            <SimpleGrid columns={{ base: 1, md: 2, lg: 3 }} spacing={4}>
              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">起始端口</FormLabel>
                <NumberInput
                  size="sm"
                  value={configData.start_port || 0}
                  min={1}
                  max={65535}
                  onChange={(_, val) => updateField('start_port', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">流类型</FormLabel>
                <Select
                  size="sm"
                  value={configData.stream_type || ''}
                  onChange={(e) => updateField('stream_type', e.target.value)}
                >
                  <option value="fullmesh">FullMesh</option>
                  <option value="incast">InCast</option>
                  <option value="p2p">P2P</option>
                </Select>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">队列对数量</FormLabel>
                <NumberInput
                  size="sm"
                  value={configData.qp_num || 0}
                  min={1}
                  onChange={(_, val) => updateField('qp_num', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">消息大小 (字节)</FormLabel>
                <NumberInput
                  size="sm"
                  value={configData.message_size_bytes || 0}
                  min={1}
                  onChange={(_, val) => updateField('message_size_bytes', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">等待时间 (秒)</FormLabel>
                <NumberInput
                  size="sm"
                  value={configData.waiting_time_seconds || 0}
                  min={0}
                  onChange={(_, val) => updateField('waiting_time_seconds', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl>
                <FormLabel fontSize="sm" fontWeight="medium">速度 (Gbps)</FormLabel>
                <NumberInput
                  size="sm"
                  value={configData.speed || 0}
                  min={0}
                  onChange={(_, val) => updateField('speed', val)}
                >
                  <NumberInputField />
                </NumberInput>
              </FormControl>

              <FormControl gridColumn={{ md: "span 2" }}>
                <FormLabel fontSize="sm" fontWeight="medium">输出目录</FormLabel>
                <Input
                  size="sm"
                  value={configData.output_base || ''}
                  onChange={(e) => updateField('output_base', e.target.value)}
                />
              </FormControl>

              <FormControl display="flex" alignItems="center" pt={6}>
                <Switch
                  isChecked={configData.rdma_cm || false}
                  onChange={(e) => updateField('rdma_cm', e.target.checked)}
                  colorScheme="blue"
                />
                <FormLabel mb={0} ml={3} fontSize="sm">使用 RDMA CM</FormLabel>
              </FormControl>
            </SimpleGrid>
          </Box>

          {/* 报告配置 & 运行配置 */}
          <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6}>
            {/* 报告配置 */}
            <Box bg="white" p={6} borderRadius="lg" shadow="sm">
              <Heading size="md" mb={4} color="green.600">
                报告配置
              </Heading>
              <VStack spacing={4} align="stretch">
                <FormControl display="flex" alignItems="center">
                  <Switch
                    isChecked={configData.report?.enable || false}
                    onChange={(e) => updateNestedField('report', 'enable', e.target.checked)}
                    colorScheme="green"
                  />
                  <FormLabel mb={0} ml={3} fontSize="sm">启用报告</FormLabel>
                </FormControl>

                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">报告目录</FormLabel>
                  <Input
                    size="sm"
                    value={configData.report?.dir || ''}
                    onChange={(e) => updateNestedField('report', 'dir', e.target.value)}
                  />
                </FormControl>
              </VStack>
            </Box>

            {/* 运行配置 */}
            <Box bg="white" p={6} borderRadius="lg" shadow="sm">
              <Heading size="md" mb={4} color="purple.600">
                运行配置
              </Heading>
              <VStack spacing={4} align="stretch">
                <FormControl display="flex" alignItems="center">
                  <Switch
                    isChecked={configData.run?.infinitely || false}
                    onChange={(e) => updateNestedField('run', 'infinitely', e.target.checked)}
                    colorScheme="purple"
                  />
                  <FormLabel mb={0} ml={3} fontSize="sm">无限运行</FormLabel>
                </FormControl>

                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">运行时长 (秒)</FormLabel>
                  <NumberInput
                    size="sm"
                    value={configData.run?.duration_seconds || 0}
                    min={0}
                    onChange={(_, val) => updateNestedField('run', 'duration_seconds', val)}
                  >
                    <NumberInputField />
                  </NumberInput>
                </FormControl>
              </VStack>
            </Box>
          </SimpleGrid>

          {/* 服务器配置 & 客户端配置 */}
          <SimpleGrid columns={{ base: 1, md: 2 }} spacing={6}>
            {/* 服务器配置 */}
            <Box bg="white" p={6} borderRadius="lg" shadow="sm">
              <Heading size="md" mb={4} color="cyan.600">
                服务器配置
              </Heading>
              <VStack spacing={4} align="stretch">
                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">主机名</FormLabel>
                  <Wrap spacing={2} mb={2} minH="40px" p={2} bg="gray.50" borderRadius="md">
                    {(configData.server?.hostname || []).map((host, index) => (
                      <WrapItem key={index}>
                        <Tag size="md" colorScheme="cyan" variant="subtle">
                          <TagLabel>{host}</TagLabel>
                          <TagCloseButton onClick={() => removeTag('server', 'hostname', index)} />
                        </Tag>
                      </WrapItem>
                    ))}
                  </Wrap>
                  <Button
                    size="xs"
                    leftIcon={<AddIcon />}
                    onClick={() => addTag('server', 'hostname')}
                    colorScheme="cyan"
                    variant="outline"
                  >
                    添加主机名
                  </Button>
                </FormControl>

                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">HCA 列表</FormLabel>
                  <Wrap spacing={2} mb={2} minH="40px" p={2} bg="gray.50" borderRadius="md">
                    {(configData.server?.hca || []).map((hca, index) => (
                      <WrapItem key={index}>
                        <Tag size="md" colorScheme="teal" variant="subtle">
                          <TagLabel>{hca}</TagLabel>
                          <TagCloseButton onClick={() => removeTag('server', 'hca', index)} />
                        </Tag>
                      </WrapItem>
                    ))}
                  </Wrap>
                  <Button
                    size="xs"
                    leftIcon={<AddIcon />}
                    onClick={() => addTag('server', 'hca')}
                    colorScheme="teal"
                    variant="outline"
                  >
                    添加 HCA
                  </Button>
                </FormControl>
              </VStack>
            </Box>

            {/* 客户端配置 */}
            <Box bg="white" p={6} borderRadius="lg" shadow="sm">
              <Heading size="md" mb={4} color="orange.600">
                客户端配置
              </Heading>
              <VStack spacing={4} align="stretch">
                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">主机名</FormLabel>
                  <Wrap spacing={2} mb={2} minH="40px" p={2} bg="gray.50" borderRadius="md">
                    {(configData.client?.hostname || []).map((host, index) => (
                      <WrapItem key={index}>
                        <Tag size="md" colorScheme="orange" variant="subtle">
                          <TagLabel>{host}</TagLabel>
                          <TagCloseButton onClick={() => removeTag('client', 'hostname', index)} />
                        </Tag>
                      </WrapItem>
                    ))}
                  </Wrap>
                  <Button
                    size="xs"
                    leftIcon={<AddIcon />}
                    onClick={() => addTag('client', 'hostname')}
                    colorScheme="orange"
                    variant="outline"
                  >
                    添加主机名
                  </Button>
                </FormControl>

                <FormControl>
                  <FormLabel fontSize="sm" fontWeight="medium">HCA 列表</FormLabel>
                  <Wrap spacing={2} mb={2} minH="40px" p={2} bg="gray.50" borderRadius="md">
                    {(configData.client?.hca || []).map((hca, index) => (
                      <WrapItem key={index}>
                        <Tag size="md" colorScheme="pink" variant="subtle">
                          <TagLabel>{hca}</TagLabel>
                          <TagCloseButton onClick={() => removeTag('client', 'hca', index)} />
                        </Tag>
                      </WrapItem>
                    ))}
                  </Wrap>
                  <Button
                    size="xs"
                    leftIcon={<AddIcon />}
                    onClick={() => addTag('client', 'hca')}
                    colorScheme="pink"
                    variant="outline"
                  >
                    添加 HCA
                  </Button>
                </FormControl>
              </VStack>
            </Box>
          </SimpleGrid>
        </VStack>
      </Box>
    </Box>
  )
}

export default ConfigEditor
