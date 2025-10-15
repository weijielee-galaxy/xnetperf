import { useState, useRef, useEffect } from 'react'
import {
  Box,
  VStack,
  HStack,
  Select,
  Button,
  Text,
  useToast,
  Card,
  CardBody,
  Heading,
  Divider,
  Badge,
  Progress,
  Alert,
  AlertIcon,
  AlertTitle,
  AlertDescription,
  Spinner,
  Table,
  Thead,
  Tbody,
  Tr,
  Th,
  Td,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalFooter,
  ModalBody,
  ModalCloseButton,
  useDisclosure,
} from '@chakra-ui/react'
import { CheckCircleIcon, WarningIcon, TimeIcon } from '@chakra-ui/icons'
import ProbeResults from '../components/ProbeResults'
import ReportResults from '../components/ReportResults'
import {
  runPrecheck,
  runTest,
  probeTest,
  collectReports,
  getReport,
  fetchConfig,
} from '../api'

const STEPS = {
  IDLE: 'idle',
  PRECHECK: 'precheck',
  RUN: 'run',
  PROBE: 'probe',
  COLLECT: 'collect',
  REPORT: 'report',
  COMPLETED: 'completed',
  ERROR: 'error',
}

const STEP_LABELS = {
  [STEPS.IDLE]: '待执行',
  [STEPS.PRECHECK]: 'PreCheck 检查',
  [STEPS.RUN]: '运行测试',
  [STEPS.PROBE]: '探测状态',
  [STEPS.COLLECT]: '收集报告',
  [STEPS.REPORT]: '生成报告',
  [STEPS.COMPLETED]: '完成',
  [STEPS.ERROR]: '错误',
}

function TrafficTestPage({ configs }) {
  const [selectedConfig, setSelectedConfig] = useState('')
  const [currentStep, setCurrentStep] = useState(STEPS.IDLE)
  const [stepStatus, setStepStatus] = useState({})
  const [precheckData, setPrecheckData] = useState(null)
  const [probeData, setProbeData] = useState(null)
  const [reportData, setReportData] = useState(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [isRunning, setIsRunning] = useState(false)
  const [configPreviewData, setConfigPreviewData] = useState(null)
  const { isOpen: isPreviewOpen, onOpen: onPreviewOpen, onClose: onPreviewClose } = useDisclosure()
  const stepRefs = useRef({})
  const toast = useToast()

  // 自动滚动到当前步骤
  useEffect(() => {
    if (currentStep !== STEPS.IDLE && stepRefs.current[currentStep]) {
      stepRefs.current[currentStep].scrollIntoView({
        behavior: 'smooth',
        block: 'center',
      })
    }
  }, [currentStep])

  // 重置所有状态
  const resetStates = () => {
    setCurrentStep(STEPS.IDLE)
    setStepStatus({})
    setPrecheckData(null)
    setProbeData(null)
    setReportData(null)
    setErrorMessage('')
    setIsRunning(false)
  }

  // 预览配置
  const handlePreviewConfig = async () => {
    if (!selectedConfig) {
      toast({
        title: '请先选择配置',
        description: '需要选择一个配置文件才能预览',
        status: 'warning',
        duration: 3000,
      })
      return
    }

    try {
      const data = await fetchConfig(selectedConfig)
      setConfigPreviewData(data)
      onPreviewOpen()
    } catch (error) {
      toast({
        title: '预览失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  // 更新步骤状态
  const updateStepStatus = (step, status, message = '') => {
    setStepStatus(prev => ({
      ...prev,
      [step]: { status, message },
    }))
  }

  // 步骤 1: PreCheck
  const executePrecheckStep = async () => {
    try {
      setCurrentStep(STEPS.PRECHECK)
      updateStepStatus(STEPS.PRECHECK, 'running', '正在执行 PreCheck...')

      const data = await runPrecheck(selectedConfig)
      setPrecheckData(data)

      // 检查是否有失败项
      // API 返回的数据结构: { check_passed, error_count, unhealthy_count, ... }
      const hasError = !data.check_passed || data.error_count > 0 || data.unhealthy_count > 0

      if (hasError) {
        updateStepStatus(STEPS.PRECHECK, 'warning', 'PreCheck 发现警告或错误')
        toast({
          title: 'PreCheck 完成',
          description: '发现一些警告或错误，请检查结果',
          status: 'warning',
          duration: 3000,
        })
        return false
      }

      updateStepStatus(STEPS.PRECHECK, 'success', 'PreCheck 通过')
      return true
    } catch (error) {
      updateStepStatus(STEPS.PRECHECK, 'error', error.message)
      setErrorMessage(`PreCheck 失败: ${error.message}`)
      setCurrentStep(STEPS.ERROR)
      toast({
        title: 'PreCheck 失败',
        description: error.message,
        status: 'error',
        duration: 5000,
      })
      return false
    }
  }

  // 步骤 2: Run Test
  const executeRunStep = async () => {
    try {
      setCurrentStep(STEPS.RUN)
      updateStepStatus(STEPS.RUN, 'running', '正在分发和运行测试脚本...')

      const result = await runTest(selectedConfig)

      if (!result.success) {
        updateStepStatus(STEPS.RUN, 'error', result.error || '测试运行失败')
        setErrorMessage(`测试运行失败: ${result.error}`)
        setCurrentStep(STEPS.ERROR)
        toast({
          title: '测试运行失败',
          description: result.error,
          status: 'error',
          duration: 5000,
        })
        return false
      }

      updateStepStatus(STEPS.RUN, 'success', result.message)
      return true
    } catch (error) {
      updateStepStatus(STEPS.RUN, 'error', error.message)
      setErrorMessage(`测试运行失败: ${error.message}`)
      setCurrentStep(STEPS.ERROR)
      toast({
        title: '测试运行失败',
        description: error.message,
        status: 'error',
        duration: 5000,
      })
      return false
    }
  }

  // 步骤 3: Probe (轮询直到所有进程完成)
  const executeProbeStep = async () => {
    try {
      setCurrentStep(STEPS.PROBE)
      updateStepStatus(STEPS.PROBE, 'running', '正在探测测试进程状态...')

      let allCompleted = false
      let probeCount = 0
      const maxProbes = 300 // 最多探测 5 分钟 (300 * 2秒)

      while (!allCompleted && probeCount < maxProbes) {
        const data = await probeTest(selectedConfig)
        setProbeData(data)

        allCompleted = data.all_completed

        if (!allCompleted) {
          updateStepStatus(
            STEPS.PROBE,
            'running',
            `探测中... (运行: ${data.running_hosts}, 完成: ${data.completed_hosts}, 错误: ${data.error_hosts})`
          )
          // 等待 2 秒后继续探测
          await new Promise(resolve => setTimeout(resolve, 2000))
          probeCount++
        }
      }

      if (!allCompleted) {
        updateStepStatus(STEPS.PROBE, 'warning', '探测超时，但继续执行后续步骤')
        toast({
          title: '探测超时',
          description: '未能确认所有进程完成，但将继续执行',
          status: 'warning',
          duration: 5000,
        })
        return true // 仍然继续
      }

      if (probeData && probeData.error_hosts > 0) {
        updateStepStatus(STEPS.PROBE, 'warning', '部分主机执行出错')
        toast({
          title: '部分主机出错',
          description: `${probeData.error_hosts} 个主机执行时出错`,
          status: 'warning',
          duration: 3000,
        })
        return true // 仍然继续收集
      }

      updateStepStatus(STEPS.PROBE, 'success', '所有测试进程已完成')
      return true
    } catch (error) {
      updateStepStatus(STEPS.PROBE, 'error', error.message)
      setErrorMessage(`探测失败: ${error.message}`)
      setCurrentStep(STEPS.ERROR)
      toast({
        title: '探测失败',
        description: error.message,
        status: 'error',
        duration: 5000,
      })
      return false
    }
  }

  // 步骤 4: Collect Reports
  const executeCollectStep = async () => {
    try {
      setCurrentStep(STEPS.COLLECT)
      updateStepStatus(STEPS.COLLECT, 'running', '正在收集测试报告...')

      const result = await collectReports(selectedConfig)

      if (!result.success) {
        updateStepStatus(STEPS.COLLECT, 'error', result.error || '报告收集失败')
        setErrorMessage(`报告收集失败: ${result.error}`)
        setCurrentStep(STEPS.ERROR)
        toast({
          title: '报告收集失败',
          description: result.error,
          status: 'error',
          duration: 5000,
        })
        return false
      }

      const totalFiles = Object.values(result.collected_files).reduce(
        (sum, count) => sum + count,
        0
      )
      updateStepStatus(
        STEPS.COLLECT,
        'success',
        `已收集 ${totalFiles} 个报告文件`
      )
      return true
    } catch (error) {
      updateStepStatus(STEPS.COLLECT, 'error', error.message)
      setErrorMessage(`报告收集失败: ${error.message}`)
      setCurrentStep(STEPS.ERROR)
      toast({
        title: '报告收集失败',
        description: error.message,
        status: 'error',
        duration: 5000,
      })
      return false
    }
  }

  // 步骤 5: Generate Report
  const executeReportStep = async () => {
    try {
      setCurrentStep(STEPS.REPORT)
      updateStepStatus(STEPS.REPORT, 'running', '正在生成性能报告...')

      const data = await getReport(selectedConfig)
      setReportData(data)

      updateStepStatus(STEPS.REPORT, 'success', '报告生成完成')
      setCurrentStep(STEPS.COMPLETED)
      toast({
        title: '流量测试完成',
        description: '所有步骤已成功执行',
        status: 'success',
        duration: 5000,
      })
      return true
    } catch (error) {
      updateStepStatus(STEPS.REPORT, 'error', error.message)
      setErrorMessage(`报告生成失败: ${error.message}`)
      setCurrentStep(STEPS.ERROR)
      toast({
        title: '报告生成失败',
        description: error.message,
        status: 'error',
        duration: 5000,
      })
      return false
    }
  }

  // 执行完整工作流
  const executeWorkflow = async () => {
    if (!selectedConfig) {
      toast({
        title: '请选择配置文件',
        status: 'warning',
        duration: 3000,
      })
      return
    }

    resetStates()
    setIsRunning(true)

    try {
      // 步骤 1: PreCheck
      if (!(await executePrecheckStep())) {
        setIsRunning(false)
        return
      }

      // 步骤 2: Run
      if (!(await executeRunStep())) {
        setIsRunning(false)
        return
      }

      // 步骤 3: Probe
      if (!(await executeProbeStep())) {
        setIsRunning(false)
        return
      }

      // 步骤 4: Collect
      if (!(await executeCollectStep())) {
        setIsRunning(false)
        return
      }

      // 步骤 5: Report
      await executeReportStep()
    } finally {
      setIsRunning(false)
    }
  }

  // 渲染步骤状态图标
  const renderStepIcon = step => {
    const status = stepStatus[step]?.status

    if (!status) {
      return <TimeIcon color="gray.400" />
    }

    switch (status) {
      case 'running':
        return <Spinner size="sm" color="blue.500" />
      case 'success':
        return <CheckCircleIcon color="green.500" />
      case 'warning':
        return <WarningIcon color="orange.500" />
      case 'error':
        return <WarningIcon color="red.500" />
      default:
        return <TimeIcon color="gray.400" />
    }
  }

  // 渲染步骤状态徽章
  const renderStepBadge = step => {
    const status = stepStatus[step]?.status

    if (!status) {
      return <Badge colorScheme="gray">待执行</Badge>
    }

    switch (status) {
      case 'running':
        return <Badge colorScheme="blue">执行中</Badge>
      case 'success':
        return <Badge colorScheme="green">成功</Badge>
      case 'warning':
        return <Badge colorScheme="orange">警告</Badge>
      case 'error':
        return <Badge colorScheme="red">失败</Badge>
      default:
        return <Badge colorScheme="gray">待执行</Badge>
    }
  }

  // 渲染步骤详细内容
  const renderStepContent = (step, status) => {
    if (!status || status === 'running') {
      return null
    }

    switch (step) {
      case STEPS.PRECHECK:
        if (precheckData) {
          return (
            <Box mt={4}>
              <Divider mb={4} />
              {/* 汇总信息 */}
              <Alert
                status={precheckData.check_passed ? 'success' : 'warning'}
                variant="subtle"
                mb={4}
              >
                <AlertIcon />
                <Box flex="1">
                  <AlertTitle>
                    {precheckData.check_passed ? '✅ Precheck 通过' : '⚠️ Precheck 发现问题'}
                  </AlertTitle>
                  <AlertDescription>
                    总计: {precheckData.total_hcas} | 健康: {precheckData.healthy_count} | 异常: {precheckData.unhealthy_count} | 错误: {precheckData.error_count}
                  </AlertDescription>
                </Box>
              </Alert>

              {/* 详细结果表格 */}
              {precheckData.results && precheckData.results.length > 0 && (
                <Box overflowX="auto">
                  <Table variant="simple" size="sm">
                    <Thead>
                      <Tr>
                        <Th>主机名</Th>
                        <Th>HCA</Th>
                        <Th>物理状态</Th>
                        <Th>端口状态</Th>
                        <Th>速度</Th>
                        <Th>固件版本</Th>
                        <Th>板卡ID</Th>
                        <Th>状态</Th>
                      </Tr>
                    </Thead>
                    <Tbody>
                      {precheckData.results.map((result, index) => (
                        <Tr key={index} bg={result.error ? 'red.50' : result.is_healthy ? 'green.50' : 'yellow.50'}>
                          <Td>{result.hostname}</Td>
                          <Td>{result.hca}</Td>
                          <Td>
                            <Badge colorScheme={result.phys_state === 'LinkUp' ? 'green' : 'red'}>
                              {result.phys_state || 'N/A'}
                            </Badge>
                          </Td>
                          <Td>
                            <Badge colorScheme={result.state === 'ACTIVE' ? 'green' : 'red'}>
                              {result.state || 'N/A'}
                            </Badge>
                          </Td>
                          <Td>
                            <Text fontWeight={precheckData.all_speeds_same ? 'normal' : 'bold'} 
                                  color={precheckData.all_speeds_same ? 'inherit' : 'orange.600'}>
                              {result.speed || 'N/A'}
                            </Text>
                          </Td>
                          <Td fontSize="xs">{result.fw_ver || 'N/A'}</Td>
                          <Td fontSize="xs">{result.board_id || 'N/A'}</Td>
                          <Td>
                            {result.error ? (
                              <Badge colorScheme="red">ERROR</Badge>
                            ) : result.is_healthy ? (
                              <Badge colorScheme="green">健康</Badge>
                            ) : (
                              <Badge colorScheme="yellow">异常</Badge>
                            )}
                          </Td>
                        </Tr>
                      ))}
                    </Tbody>
                  </Table>
                </Box>
              )}

              {/* 错误信息 */}
              {precheckData.results && precheckData.results.some(r => r.error) && (
                <Box mt={4}>
                  <Text fontWeight="bold" mb={2}>错误详情:</Text>
                  {precheckData.results
                    .filter(r => r.error)
                    .map((result, index) => (
                      <Text key={index} color="red.600" fontSize="sm">
                        • {result.hostname} - {result.hca}: {result.error}
                      </Text>
                    ))}
                </Box>
              )}
            </Box>
          )
        }
        break

      case STEPS.RUN:
        if (status === 'success') {
          return (
            <Box mt={4}>
              <Divider mb={4} />
              <Alert status="success" variant="subtle">
                <AlertIcon />
                <AlertDescription>
                  测试脚本已成功分发到所有节点并启动
                </AlertDescription>
              </Alert>
            </Box>
          )
        }
        break

      case STEPS.PROBE:
        if (probeData) {
          return (
            <Box mt={4}>
              <Divider mb={4} />
              <ProbeResults data={probeData} />
            </Box>
          )
        }
        break

      case STEPS.COLLECT:
        if (status === 'success') {
          return (
            <Box mt={4}>
              <Divider mb={4} />
              <Alert status="success" variant="subtle">
                <AlertIcon />
                <AlertDescription>
                  测试结果已成功从所有节点收集
                </AlertDescription>
              </Alert>
            </Box>
          )
        }
        break

      case STEPS.REPORT:
        if (reportData) {
          return (
            <Box mt={4}>
              <Divider mb={4} />
              <ReportResults data={reportData} />
            </Box>
          )
        }
        break

      default:
        break
    }

    return null
  }

  return (
    <Box p={6} h="100%" overflowY="auto">
      <VStack spacing={6} align="stretch">
        {/* 配置选择区 */}
        <Card>
          <CardBody>
            <VStack spacing={4} align="stretch">
              <Heading size="md">流量测试</Heading>
              <Divider />

              <HStack>
                <Text minW="100px">选择配置:</Text>
                <Select
                  placeholder="请选择配置文件"
                  value={selectedConfig}
                  onChange={e => {
                    setSelectedConfig(e.target.value)
                    resetStates()
                  }}
                  isDisabled={isRunning}
                >
                  {configs.map(cfg => (
                    <option key={cfg.name} value={cfg.name}>
                      {cfg.name}
                    </option>
                  ))}
                </Select>

                <Button
                  variant="outline"
                  onClick={handlePreviewConfig}
                  isDisabled={!selectedConfig || isRunning}
                >
                  预览配置
                </Button>

                <Button
                  colorScheme="blue"
                  onClick={executeWorkflow}
                  isLoading={isRunning}
                  isDisabled={!selectedConfig || isRunning}
                  minW="150px"
                >
                  开始测试
                </Button>

                <Button
                  variant="outline"
                  onClick={resetStates}
                  isDisabled={isRunning}
                >
                  重置
                </Button>
              </HStack>
            </VStack>
          </CardBody>
        </Card>

        {/* 工作流步骤显示 - 时间线样式 */}
        {currentStep !== STEPS.IDLE && (
          <Card>
            <CardBody>
              <VStack spacing={4} align="stretch">
                <Heading size="sm">执行步骤</Heading>
                <Divider />

                {/* 进度条 */}
                <Box>
                  <Progress
                    value={
                      currentStep === STEPS.PRECHECK
                        ? 20
                        : currentStep === STEPS.RUN
                        ? 40
                        : currentStep === STEPS.PROBE
                        ? 60
                        : currentStep === STEPS.COLLECT
                        ? 80
                        : currentStep === STEPS.REPORT
                        ? 90
                        : currentStep === STEPS.COMPLETED
                        ? 100
                        : 0
                    }
                    colorScheme={currentStep === STEPS.ERROR ? 'red' : 'blue'}
                    hasStripe
                    isAnimated={isRunning}
                  />
                </Box>

                {/* 时间线样式的步骤列表 */}
                <Box position="relative" pl={8}>
                  {/* 时间线主线 */}
                  <Box
                    position="absolute"
                    left="15px"
                    top="12px"
                    bottom="12px"
                    width="2px"
                    bg="gray.200"
                  />

                  {/* 步骤节点 */}
                  <VStack spacing={6} align="stretch">
                    {[
                      STEPS.PRECHECK,
                      STEPS.RUN,
                      STEPS.PROBE,
                      STEPS.COLLECT,
                      STEPS.REPORT,
                    ].map((step, index) => {
                      const status = stepStatus[step]?.status
                      const isActive = currentStep === step
                      
                      return (
                        <Box 
                          key={step} 
                          position="relative"
                          ref={el => stepRefs.current[step] = el}
                        >
                          {/* 时间线节点圆圈 */}
                          <Box
                            position="absolute"
                            left="-23px"
                            top="2px"
                            w="32px"
                            h="32px"
                            borderRadius="full"
                            bg={
                              status === 'success'
                                ? 'green.500'
                                : status === 'error'
                                ? 'red.500'
                                : status === 'warning'
                                ? 'orange.500'
                                : status === 'running'
                                ? 'blue.500'
                                : 'gray.300'
                            }
                            border="4px solid white"
                            display="flex"
                            alignItems="center"
                            justifyContent="center"
                            boxShadow={isActive ? '0 0 0 4px rgba(66, 153, 225, 0.3)' : 'sm'}
                            transition="all 0.3s"
                          >
                            {status === 'running' ? (
                              <Spinner size="sm" color="white" />
                            ) : status === 'success' ? (
                              <CheckCircleIcon color="white" boxSize={4} />
                            ) : status === 'error' || status === 'warning' ? (
                              <WarningIcon color="white" boxSize={4} />
                            ) : (
                              <TimeIcon color="white" boxSize={4} />
                            )}
                          </Box>

                          {/* 步骤内容 */}
                          <Box
                            bg={isActive ? 'blue.50' : 'white'}
                            p={4}
                            borderRadius="md"
                            border="1px solid"
                            borderColor={isActive ? 'blue.200' : 'gray.200'}
                            transition="all 0.3s"
                            _hover={{ shadow: 'md' }}
                          >
                            <HStack justify="space-between" mb={2}>
                              <HStack>
                                <Text fontWeight="bold" fontSize="md">
                                  {STEP_LABELS[step]}
                                </Text>
                                {renderStepBadge(step)}
                              </HStack>
                              <Text fontSize="xs" color="gray.500">
                                步骤 {index + 1}/5
                              </Text>
                            </HStack>
                            
                            {stepStatus[step]?.message && (
                              <Text fontSize="sm" color="gray.600" mt={2}>
                                {stepStatus[step].message}
                              </Text>
                            )}

                            {/* 嵌入每个步骤的详细结果 */}
                            {renderStepContent(step, status)}
                          </Box>
                        </Box>
                      )
                    })}
                  </VStack>
                </Box>
              </VStack>
            </CardBody>
          </Card>
        )}

        {/* 错误提示 */}
        {currentStep === STEPS.ERROR && errorMessage && (
          <Alert status="error">
            <AlertIcon />
            <Box flex="1">
              <AlertTitle>执行失败</AlertTitle>
              <AlertDescription>{errorMessage}</AlertDescription>
            </Box>
          </Alert>
        )}
      </VStack>

      {/* 配置预览 Modal */}
      <Modal isOpen={isPreviewOpen} onClose={onPreviewClose} size="4xl" scrollBehavior="inside">
        <ModalOverlay />
        <ModalContent maxH="80vh">
          <ModalHeader>配置预览 - {selectedConfig}</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            {configPreviewData && (
              <VStack spacing={4} align="stretch">
                {/* 基础配置 */}
                <Box>
                  <Heading size="sm" mb={3} color="blue.600">基础配置</Heading>
                  <VStack spacing={2} align="stretch" bg="gray.50" p={4} borderRadius="md">
                    <HStack>
                      <Text fontWeight="medium" minW="150px">流类型:</Text>
                      <Badge colorScheme="blue">{configPreviewData.stream_type}</Badge>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">起始端口:</Text>
                      <Text>{configPreviewData.start_port}</Text>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">理论速度:</Text>
                      <Text>{configPreviewData.speed} Gbps</Text>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">测试时长:</Text>
                      <Text>{configPreviewData.run?.duration_seconds || 10} 秒</Text>
                    </HStack>
                  </VStack>
                </Box>

                {/* 服务端配置 */}
                <Box>
                  <Heading size="sm" mb={3} color="green.600">服务端配置</Heading>
                  <VStack spacing={2} align="stretch" bg="green.50" p={4} borderRadius="md">
                    <Box>
                      <Text fontWeight="medium" mb={2}>主机列表:</Text>
                      <HStack wrap="wrap" spacing={2}>
                        {configPreviewData.server?.hostname?.map((host, index) => (
                          <Badge key={index} colorScheme="green" variant="outline">
                            {host}
                          </Badge>
                        )) || <Text color="gray.500">未配置</Text>}
                      </HStack>
                    </Box>
                    <Box>
                      <Text fontWeight="medium" mb={2}>HCA 设备:</Text>
                      <HStack wrap="wrap" spacing={2}>
                        {configPreviewData.server?.hca?.map((hca, index) => (
                          <Badge key={index} colorScheme="green" variant="solid">
                            {hca}
                          </Badge>
                        )) || <Text color="gray.500">未配置</Text>}
                      </HStack>
                    </Box>
                  </VStack>
                </Box>

                {/* 客户端配置 */}
                <Box>
                  <Heading size="sm" mb={3} color="orange.600">客户端配置</Heading>
                  <VStack spacing={2} align="stretch" bg="orange.50" p={4} borderRadius="md">
                    <Box>
                      <Text fontWeight="medium" mb={2}>主机列表:</Text>
                      <HStack wrap="wrap" spacing={2}>
                        {configPreviewData.client?.hostname?.map((host, index) => (
                          <Badge key={index} colorScheme="orange" variant="outline">
                            {host}
                          </Badge>
                        )) || <Text color="gray.500">未配置</Text>}
                      </HStack>
                    </Box>
                    <Box>
                      <Text fontWeight="medium" mb={2}>HCA 设备:</Text>
                      <HStack wrap="wrap" spacing={2}>
                        {configPreviewData.client?.hca?.map((hca, index) => (
                          <Badge key={index} colorScheme="orange" variant="solid">
                            {hca}
                          </Badge>
                        )) || <Text color="gray.500">未配置</Text>}
                      </HStack>
                    </Box>
                  </VStack>
                </Box>

                {/* 高级配置 */}
                <Box>
                  <Heading size="sm" mb={3} color="purple.600">高级配置</Heading>
                  <VStack spacing={2} align="stretch" bg="purple.50" p={4} borderRadius="md">
                    <HStack>
                      <Text fontWeight="medium" minW="150px">QP 数量:</Text>
                      <Text>{configPreviewData.qp_num || 1}</Text>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">消息大小:</Text>
                      <Text>{configPreviewData.message_size_bytes || 65536} 字节</Text>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">RDMA CM:</Text>
                      <Badge colorScheme={configPreviewData.rdma_cm ? 'green' : 'gray'}>
                        {configPreviewData.rdma_cm ? '启用' : '禁用'}
                      </Badge>
                    </HStack>
                    <HStack>
                      <Text fontWeight="medium" minW="150px">生成报告:</Text>
                      <Badge colorScheme={configPreviewData.report?.enable ? 'green' : 'gray'}>
                        {configPreviewData.report?.enable ? '启用' : '禁用'}
                      </Badge>
                    </HStack>
                    {configPreviewData.report?.enable && (
                      <HStack>
                        <Text fontWeight="medium" minW="150px">报告目录:</Text>
                        <Text fontFamily="mono" fontSize="sm" bg="gray.100" px={2} py={1} borderRadius="sm">
                          {configPreviewData.report?.dir || '/tmp'}
                        </Text>
                      </HStack>
                    )}
                  </VStack>
                </Box>

                {/* 统计信息 */}
                <Box>
                  <Heading size="sm" mb={3} color="gray.600">测试规模</Heading>
                  <HStack spacing={8} bg="gray.100" p={4} borderRadius="md">
                    <VStack>
                      <Text fontSize="2xl" fontWeight="bold" color="green.600">
                        {configPreviewData.server?.hostname?.length || 0}
                      </Text>
                      <Text fontSize="sm" color="gray.600">服务端主机</Text>
                    </VStack>
                    <VStack>
                      <Text fontSize="2xl" fontWeight="bold" color="orange.600">
                        {configPreviewData.client?.hostname?.length || 0}
                      </Text>
                      <Text fontSize="sm" color="gray.600">客户端主机</Text>
                    </VStack>
                    <VStack>
                      <Text fontSize="2xl" fontWeight="bold" color="blue.600">
                        {((configPreviewData.server?.hostname?.length || 0) * (configPreviewData.server?.hca?.length || 0)) +
                         ((configPreviewData.client?.hostname?.length || 0) * (configPreviewData.client?.hca?.length || 0))}
                      </Text>
                      <Text fontSize="sm" color="gray.600">总 HCA 数量</Text>
                    </VStack>
                    {configPreviewData.stream_type === 'fullmesh' && (
                      <VStack>
                        <Text fontSize="2xl" fontWeight="bold" color="purple.600">
                          {((configPreviewData.client?.hostname?.length || 0) * (configPreviewData.client?.hca?.length || 0)) * 
                           ((configPreviewData.server?.hostname?.length || 0) * (configPreviewData.server?.hca?.length || 0))}
                        </Text>
                        <Text fontSize="sm" color="gray.600">预计连接数</Text>
                      </VStack>
                    )}
                  </HStack>
                </Box>
              </VStack>
            )}
          </ModalBody>
          <ModalFooter>
            <Button onClick={onPreviewClose}>关闭</Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}

export default TrafficTestPage
