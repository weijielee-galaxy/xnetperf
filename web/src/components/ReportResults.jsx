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
  Heading,
  Divider,
  Stat,
  StatLabel,
  StatNumber,
  StatHelpText,
  StatGroup,
  Card,
  CardBody,
} from '@chakra-ui/react'

function ReportResults({ data }) {
  if (!data) {
    return <Text color="gray.500">暂无报告数据</Text>
  }

  // 渲染 P2P 报告
  if (data.stream_type === 'p2p') {
    return renderP2PReport(data)
  }

  // 渲染传统 FullMesh/InCast 报告
  return renderTraditionalReport(data)
}

// 渲染 P2P 报告
function renderP2PReport(data) {
  const { p2p_data, p2p_summary } = data

  if (!p2p_data) {
    return <Text color="gray.500">P2P 报告数据为空</Text>
  }

  // 转换数据为数组以便排序
  const dataArray = []
  Object.keys(p2p_data).forEach(hostname => {
    Object.keys(p2p_data[hostname]).forEach(device => {
      dataArray.push(p2p_data[hostname][device])
    })
  })

  // 按主机名和设备排序
  dataArray.sort((a, b) => {
    if (a.hostname !== b.hostname) {
      return a.hostname.localeCompare(b.hostname)
    }
    return a.device.localeCompare(b.device)
  })

  return (
    <VStack spacing={4} align="stretch">
      {/* P2P 汇总 */}
      {p2p_summary && (
        <StatGroup>
          <Stat>
            <StatLabel>总连接对数</StatLabel>
            <StatNumber>{p2p_summary.total_pairs}</StatNumber>
          </Stat>
          <Stat>
            <StatLabel>平均速度</StatLabel>
            <StatNumber>{p2p_summary.avg_speed.toFixed(2)} Gbps</StatNumber>
          </Stat>
        </StatGroup>
      )}

      <Divider />

      {/* P2P 详细数据表 */}
      <Box overflowX="auto">
        <Table variant="simple" size="sm">
          <Thead>
            <Tr>
              <Th>主机名</Th>
              <Th>设备</Th>
              <Th isNumeric>平均速度 (Gbps)</Th>
              <Th isNumeric>连接数</Th>
            </Tr>
          </Thead>
          <Tbody>
            {dataArray.map((item, index) => (
              <Tr key={index}>
                <Td>{item.hostname}</Td>
                <Td>{item.device}</Td>
                <Td isNumeric>{item.avg_speed.toFixed(2)}</Td>
                <Td isNumeric>{item.count}</Td>
              </Tr>
            ))}
          </Tbody>
        </Table>
      </Box>
    </VStack>
  )
}

// 渲染传统 FullMesh/InCast 报告
function renderTraditionalReport(data) {
  const { client_data, server_data, theoretical_bw_per_client, total_server_bw, client_count } = data

  return (
    <VStack spacing={6} align="stretch">
      {/* 带宽汇总 */}
      <Card>
        <CardBody>
          <StatGroup>
            <Stat>
              <StatLabel>服务端总带宽</StatLabel>
              <StatNumber>{total_server_bw?.toFixed(2)} Gbps</StatNumber>
            </Stat>
            <Stat>
              <StatLabel>客户端数量</StatLabel>
              <StatNumber>{client_count}</StatNumber>
            </Stat>
            <Stat>
              <StatLabel>理论单客户端带宽</StatLabel>
              <StatNumber>{theoretical_bw_per_client?.toFixed(2)} Gbps</StatNumber>
              <StatHelpText>
                {total_server_bw?.toFixed(2)} Gbps ÷ {client_count}
              </StatHelpText>
            </Stat>
          </StatGroup>
        </CardBody>
      </Card>

      {/* 客户端数据 (TX) */}
      <Box>
        <Heading size="sm" mb={3}>
          客户端数据 (TX)
        </Heading>
        <Box overflowX="auto">
          <Table variant="simple" size="sm">
            <Thead>
              <Tr>
                <Th>主机名</Th>
                <Th>设备</Th>
                <Th isNumeric>实际带宽 (Gbps)</Th>
                <Th isNumeric>理论带宽 (Gbps)</Th>
                <Th isNumeric>差值 (Gbps)</Th>
                <Th isNumeric>差值百分比</Th>
                <Th>状态</Th>
              </Tr>
            </Thead>
            <Tbody>
              {client_data &&
                Object.keys(client_data).map((hostname, hIndex) =>
                  Object.keys(client_data[hostname]).map((device, dIndex) => {
                    const item = client_data[hostname][device]
                    return (
                      <Tr key={`${hIndex}-${dIndex}`}>
                        <Td>{item.hostname}</Td>
                        <Td>{item.device}</Td>
                        <Td isNumeric>{item.actual_bw.toFixed(2)}</Td>
                        <Td isNumeric>{item.theoretical_bw.toFixed(2)}</Td>
                        <Td isNumeric>
                          <Text color={item.delta >= 0 ? 'green.600' : 'red.600'}>
                            {item.delta.toFixed(2)}
                          </Text>
                        </Td>
                        <Td isNumeric>
                          <Text color={Math.abs(item.delta_percent) > 20 ? 'red.600' : 'green.600'}>
                            {item.delta_percent.toFixed(1)}%
                          </Text>
                        </Td>
                        <Td>
                          <Badge colorScheme={item.status === 'OK' ? 'green' : 'red'}>
                            {item.status}
                          </Badge>
                        </Td>
                      </Tr>
                    )
                  })
                )}
            </Tbody>
          </Table>
        </Box>
      </Box>

      <Divider />

      {/* 服务端数据 (RX) */}
      <Box>
        <Heading size="sm" mb={3}>
          服务端数据 (RX)
        </Heading>
        <Box overflowX="auto">
          <Table variant="simple" size="sm">
            <Thead>
              <Tr>
                <Th>主机名</Th>
                <Th>设备</Th>
                <Th isNumeric>接收带宽 (Gbps)</Th>
                <Th isNumeric>理论带宽 (Gbps)</Th>
                <Th isNumeric>差值 (Gbps)</Th>
                <Th isNumeric>差值百分比</Th>
                <Th>状态</Th>
              </Tr>
            </Thead>
            <Tbody>
              {server_data &&
                Object.keys(server_data).map((hostname, hIndex) =>
                  Object.keys(server_data[hostname]).map((device, dIndex) => {
                    const item = server_data[hostname][device]
                    return (
                      <Tr key={`${hIndex}-${dIndex}`}>
                        <Td>{item.hostname}</Td>
                        <Td>{item.device}</Td>
                        <Td isNumeric>{item.rx_bw.toFixed(2)}</Td>
                        <Td isNumeric>{item.theoretical_bw.toFixed(2)}</Td>
                        <Td isNumeric>
                          <Text color={item.delta >= 0 ? 'green.600' : 'red.600'}>
                            {item.delta.toFixed(2)}
                          </Text>
                        </Td>
                        <Td isNumeric>
                          <Text color={Math.abs(item.delta_percent) > 20 ? 'red.600' : 'green.600'}>
                            {item.delta_percent.toFixed(1)}%
                          </Text>
                        </Td>
                        <Td>
                          <Badge colorScheme={item.status === 'OK' ? 'green' : 'red'}>
                            {item.status}
                          </Badge>
                        </Td>
                      </Tr>
                    )
                  })
                )}
            </Tbody>
          </Table>
        </Box>
      </Box>
    </VStack>
  )
}

export default ReportResults
