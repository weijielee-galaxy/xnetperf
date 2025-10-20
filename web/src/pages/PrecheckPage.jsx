import { useState } from 'react'
import {
  Box,
  Button,
  VStack,
  Heading,
  useToast,
  Text,
  Select,
  HStack,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  TableContainer,
  Badge,
  Spinner,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  SimpleGrid,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
} from '@chakra-ui/react'
import { runPrecheck } from '../api'

function PrecheckPage({ configs }) {
  const [selectedConfig, setSelectedConfig] = useState('')
  const [loading, setLoading] = useState(false)
  const [result, setResult] = useState(null)
  const toast = useToast()

  const handleRunPrecheck = async () => {
    if (!selectedConfig) {
      toast({
        title: '请选择配置文件',
        status: 'warning',
        duration: 2000,
      })
      return
    }

    try {
      setLoading(true)
      setResult(null)
      const data = await runPrecheck(selectedConfig)
      setResult(data)
      
      if (data.check_passed) {
        toast({
          title: '✅ Precheck 通过！',
          description: '所有 HCA 状态正常',
          status: 'success',
          duration: 3000,
        })
      } else {
        toast({
          title: '❌ Precheck 失败',
          description: '部分 HCA 状态异常',
          status: 'error',
          duration: 3000,
        })
      }
    } catch (error) {
      toast({
        title: 'Precheck 执行失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  // 获取状态颜色
  const getStatusColor = (isHealthy, error) => {
    if (error) return 'red'
    return isHealthy ? 'green' : 'yellow'
  }

  // 获取状态文本
  const getStatusText = (isHealthy, error) => {
    if (error) return 'ERROR'
    return isHealthy ? 'HEALTHY' : 'UNHEALTHY'
  }

  // 获取值的频率颜色（用于 speed, fw_ver, board_id）
  const getFrequencyColor = (value, stats) => {
    if (!stats || Object.keys(stats).length <= 1) return undefined
    
    const count = stats[value] || 0
    const maxCount = Math.max(...Object.values(stats))
    const minCount = Math.min(...Object.values(stats))
    
    if (count === minCount && count < maxCount) {
      return 'yellow'
    } else if (count === maxCount) {
      return 'green'
    }
    return undefined
  }

  return (
    <Box p={6} bg="gray.50" minH="100vh">
      <VStack spacing={6} align="stretch" maxW="1400px" mx="auto">
        {/* Header */}
        <Box>
          <Heading size="lg" mb={2}>PreCheck 检查</Heading>
          <Text color="gray.600">检查所有配置的 InfiniBand HCA 状态</Text>
        </Box>

        {/* 配置选择 */}
        <Box bg="white" p={6} borderRadius="lg" shadow="sm">
          <HStack spacing={4}>
            <Box flex="1">
              <Text mb={2} fontWeight="medium">选择配置文件</Text>
              <Select
                placeholder="请选择配置文件"
                value={selectedConfig}
                onChange={(e) => setSelectedConfig(e.target.value)}
              >
                {configs.map((config) => (
                  <option key={config.name} value={config.name}>
                    {config.name} {config.is_default ? '(默认)' : ''}
                  </option>
                ))}
              </Select>
            </Box>
            <Box pt={8}>
              <Button
                colorScheme="blue"
                onClick={handleRunPrecheck}
                isLoading={loading}
                loadingText="检查中..."
                size="lg"
              >
                开始检查
              </Button>
            </Box>
          </HStack>
        </Box>

        {/* 加载中 */}
        {loading && (
          <Box textAlign="center" py={10}>
            <Spinner size="xl" color="blue.500" thickness="4px" />
            <Text mt={4} color="gray.600">正在执行 Precheck，请稍候...</Text>
          </Box>
        )}

        {/* 检查结果 */}
        {result && !loading && (
          <VStack spacing={6} align="stretch">
            {/* 总体状态 */}
            <Alert
              status={result.check_passed ? 'success' : 'error'}
              variant="subtle"
              borderRadius="lg"
            >
              <AlertIcon boxSize="40px" mr={4} />
              <Box flex="1">
                <AlertTitle fontSize="lg">
                  {result.check_passed ? '✅ Precheck 通过' : '❌ Precheck 失败'}
                </AlertTitle>
                <AlertDescription>
                  {result.check_passed 
                    ? '所有 HCA 状态正常，速度一致，可以进行性能测试'
                    : '部分 HCA 状态异常或速度不一致，请检查网络环境'}
                </AlertDescription>
              </Box>
            </Alert>

            {/* 统计信息 */}
            <SimpleGrid columns={{ base: 2, md: 4 }} spacing={4}>
              <Stat bg="white" p={4} borderRadius="lg" shadow="sm">
                <StatLabel>总计 HCA</StatLabel>
                <StatNumber>{result.total_hcas}</StatNumber>
                <StatHelpText>检查的 HCA 数量</StatHelpText>
              </Stat>
              
              <Stat bg="white" p={4} borderRadius="lg" shadow="sm">
                <StatLabel>健康</StatLabel>
                <StatNumber color="green.500">{result.healthy_count}</StatNumber>
                <StatHelpText>LinkUp & ACTIVE</StatHelpText>
              </Stat>
              
              <Stat bg="white" p={4} borderRadius="lg" shadow="sm">
                <StatLabel>异常</StatLabel>
                <StatNumber color="yellow.500">{result.unhealthy_count}</StatNumber>
                <StatHelpText>状态不正常</StatHelpText>
              </Stat>
              
              <Stat bg="white" p={4} borderRadius="lg" shadow="sm">
                <StatLabel>错误</StatLabel>
                <StatNumber color="red.500">{result.error_count}</StatNumber>
                <StatHelpText>无法获取信息</StatHelpText>
              </Stat>
            </SimpleGrid>

            {/* Speed 一致性检查 */}
            {!result.all_speeds_same && (
              <Alert status="warning" borderRadius="lg">
                <AlertIcon />
                <AlertTitle>速度不一致！</AlertTitle>
                <AlertDescription>
                  检测到不同的速度配置：
                  {Object.entries(result.speed_stats).map(([speed, count]) => (
                    <Badge key={speed} ml={2} colorScheme={count === Math.max(...Object.values(result.speed_stats)) ? 'green' : 'red'}>
                      {speed} ({count}个)
                    </Badge>
                  ))}
                </AlertDescription>
              </Alert>
            )}

            {/* 详细结果表格 */}
            <Box bg="white" p={6} borderRadius="lg" shadow="sm">
              <Heading size="md" mb={4}>详细检查结果</Heading>
              <TableContainer>
                <Table variant="simple" size="sm">
                  <Thead>
                    <Tr>
                      <Th>主机名</Th>
                      <Th>HCA</Th>
                      <Th>物理状态</Th>
                      <Th>逻辑状态</Th>
                      <Th>速度</Th>
                      <Th>固件版本</Th>
                      <Th>板卡 ID</Th>
                      <Th>状态</Th>
                    </Tr>
                  </Thead>
                  <Tbody>
                    {result.results.map((item, index) => {
                      const prevHostname = index > 0 ? result.results[index - 1].hostname : null
                      const showHostname = item.hostname !== prevHostname
                      
                      return (
                        <Tr key={index}>
                          <Td fontWeight={showHostname ? 'bold' : 'normal'}>
                            {showHostname ? item.hostname : ''}
                          </Td>
                          <Td fontFamily="monospace">{item.hca}</Td>
                          <Td fontSize="xs">{item.phys_state}</Td>
                          <Td fontSize="xs">{item.state}</Td>
                          <Td>
                            <Badge
                              colorScheme={getFrequencyColor(item.speed, result.speed_stats)}
                              fontSize="xs"
                            >
                              {item.speed}
                            </Badge>
                          </Td>
                          <Td>
                            <Badge
                              colorScheme={getFrequencyColor(item.fw_ver, result.fw_ver_stats)}
                              fontSize="xs"
                            >
                              {item.fw_ver}
                            </Badge>
                          </Td>
                          <Td>
                            <Badge
                              colorScheme={getFrequencyColor(item.board_id, result.board_id_stats)}
                              fontSize="xs"
                            >
                              {item.board_id}
                            </Badge>
                          </Td>
                          <Td>
                            <Badge colorScheme={getStatusColor(item.is_healthy, item.error)}>
                              {getStatusText(item.is_healthy, item.error)}
                            </Badge>
                          </Td>
                        </Tr>
                      )
                    })}
                  </Tbody>
                </Table>
              </TableContainer>
            </Box>
          </VStack>
        )}
      </VStack>
    </Box>
  )
}

export default PrecheckPage
