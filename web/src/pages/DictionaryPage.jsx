import { useState, useEffect } from 'react'
import {
  Box,
  Button,
  VStack,
  Heading,
  Textarea,
  useToast,
  Text,
  SimpleGrid,
  HStack,
} from '@chakra-ui/react'
import { fetchHostnames, updateHostnames, fetchHCAs, updateHCAs } from '../api'

function DictionaryPage() {
  const [hostnamesText, setHostnamesText] = useState('')
  const [hcasText, setHcasText] = useState('')
  const [loading, setLoading] = useState(false)
  const toast = useToast()

  // 加载字典
  const loadDictionaries = async () => {
    try {
      const [hostnames, hcas] = await Promise.all([
        fetchHostnames(),
        fetchHCAs()
      ])
      setHostnamesText(hostnames.join('\n'))
      setHcasText(hcas.join('\n'))
    } catch (error) {
      toast({
        title: '加载字典失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  useEffect(() => {
    loadDictionaries()
  }, [])

  // 保存主机名
  const handleSaveHostnames = async () => {
    try {
      setLoading(true)
      const hostnames = hostnamesText
        .split('\n')
        .map(line => line.trim())
        .filter(line => line !== '')
      
      await updateHostnames(hostnames)
      toast({
        title: '保存成功！',
        description: `已保存 ${hostnames.length} 个主机名`,
        status: 'success',
        duration: 2000,
      })
      await loadDictionaries()
    } catch (error) {
      toast({
        title: '保存失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  // 保存 HCA
  const handleSaveHCAs = async () => {
    try {
      setLoading(true)
      const hcas = hcasText
        .split('\n')
        .map(line => line.trim())
        .filter(line => line !== '')
      
      await updateHCAs(hcas)
      toast({
        title: '保存成功！',
        description: `已保存 ${hcas.length} 个 HCA`,
        status: 'success',
        duration: 2000,
      })
      await loadDictionaries()
    } catch (error) {
      toast({
        title: '保存失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setLoading(false)
    }
  }

  return (
    <Box p={6} bg="gray.50" minH="100vh">
      <VStack spacing={6} align="stretch" maxW="1200px" mx="auto">
        <Box>
          <Heading size="lg" mb={2}>字典管理</Heading>
          <Text color="gray.600">批量管理主机名和 HCA 列表，每行一条数据</Text>
        </Box>

        <SimpleGrid columns={{ base: 1, lg: 2 }} spacing={6}>
          {/* 主机名字典 */}
          <Box bg="white" p={6} borderRadius="lg" shadow="sm">
            <VStack align="stretch" spacing={4}>
              <HStack justify="space-between">
                <Heading size="md" color="cyan.600">主机名字典</Heading>
                <Text fontSize="sm" color="gray.500">
                  {hostnamesText.split('\n').filter(l => l.trim()).length} 条
                </Text>
              </HStack>
              
              <Textarea
                value={hostnamesText}
                onChange={(e) => setHostnamesText(e.target.value)}
                placeholder="每行输入一个主机名，例如：&#10;cetus-g88-061&#10;cetus-g88-062&#10;cetus-g88-065"
                rows={15}
                fontFamily="monospace"
                fontSize="sm"
              />
              
              <Button
                colorScheme="cyan"
                onClick={handleSaveHostnames}
                isLoading={loading}
              >
                保存主机名
              </Button>
            </VStack>
          </Box>

          {/* HCA 字典 */}
          <Box bg="white" p={6} borderRadius="lg" shadow="sm">
            <VStack align="stretch" spacing={4}>
              <HStack justify="space-between">
                <Heading size="md" color="orange.600">HCA 字典</Heading>
                <Text fontSize="sm" color="gray.500">
                  {hcasText.split('\n').filter(l => l.trim()).length} 条
                </Text>
              </HStack>
              
              <Textarea
                value={hcasText}
                onChange={(e) => setHcasText(e.target.value)}
                placeholder="每行输入一个 HCA 名称，例如：&#10;mlx5_0&#10;mlx5_1"
                rows={15}
                fontFamily="monospace"
                fontSize="sm"
              />
              
              <Button
                colorScheme="orange"
                onClick={handleSaveHCAs}
                isLoading={loading}
              >
                保存 HCA 列表
              </Button>
            </VStack>
          </Box>
        </SimpleGrid>
      </VStack>
    </Box>
  )
}

export default DictionaryPage
