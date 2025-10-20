import {
  Box,
  VStack,
  HStack,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Badge,
  Text,
  Stat,
  StatLabel,
  StatNumber,
  StatGroup,
} from '@chakra-ui/react'
import { CheckCircleIcon, TimeIcon, WarningIcon } from '@chakra-ui/icons'

function ProbeResults({ data }) {
  if (!data || !data.results) {
    return <Text color="gray.500">暂无探测数据</Text>
  }

  // 渲染状态图标
  const renderStatusIcon = status => {
    switch (status) {
      case 'RUNNING':
        return <TimeIcon color="blue.500" />
      case 'COMPLETED':
        return <CheckCircleIcon color="green.500" />
      case 'ERROR':
        return <WarningIcon color="red.500" />
      default:
        return null
    }
  }

  // 渲染状态徽章
  const renderStatusBadge = status => {
    switch (status) {
      case 'RUNNING':
        return <Badge colorScheme="blue">运行中</Badge>
      case 'COMPLETED':
        return <Badge colorScheme="green">已完成</Badge>
      case 'ERROR':
        return <Badge colorScheme="red">错误</Badge>
      default:
        return <Badge>{status}</Badge>
    }
  }

  return (
    <VStack spacing={4} align="stretch">
      {/* 汇总统计 */}
      <StatGroup>
        <Stat>
          <StatLabel>运行中主机</StatLabel>
          <StatNumber color="blue.500">{data.running_hosts}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel>已完成主机</StatLabel>
          <StatNumber color="green.500">{data.completed_hosts}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel>错误主机</StatLabel>
          <StatNumber color="red.500">{data.error_hosts}</StatNumber>
        </Stat>
        <Stat>
          <StatLabel>总进程数</StatLabel>
          <StatNumber>{data.total_processes}</StatNumber>
        </Stat>
      </StatGroup>

      {/* 时间戳 */}
      <HStack justify="space-between">
        <Text fontSize="sm" color="gray.600">
          探测时间: {data.timestamp}
        </Text>
        {data.all_completed && (
          <Badge colorScheme="green" fontSize="md">
            所有进程已完成
          </Badge>
        )}
      </HStack>

      {/* 主机详情表格 */}
      <Box overflowX="auto">
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>主机名</Th>
              <Th>进程数</Th>
              <Th>状态</Th>
              <Th>错误信息</Th>
            </Tr>
          </Thead>
          <Tbody>
            {data.results.map((result, index) => (
              <Tr key={index}>
                <Td>
                  <HStack>
                    {renderStatusIcon(result.status)}
                    <Text>{result.hostname}</Text>
                  </HStack>
                </Td>
                <Td>{result.process_count}</Td>
                <Td>{renderStatusBadge(result.status)}</Td>
                <Td>
                  {result.error ? (
                    <Text color="red.500" fontSize="sm">
                      {result.error}
                    </Text>
                  ) : (
                    <Text color="gray.400">-</Text>
                  )}
                </Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </VStack>
  )
}

export default ProbeResults
