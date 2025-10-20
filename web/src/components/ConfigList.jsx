import {
  Box,
  Button,
  VStack,
  HStack,
  Text,
  IconButton,
  useDisclosure,
  Modal,
  ModalOverlay,
  ModalContent,
  ModalHeader,
  ModalBody,
  ModalFooter,
  ModalCloseButton,
  Input,
  FormControl,
  FormLabel,
  useToast,
} from '@chakra-ui/react'
import { AddIcon, DeleteIcon, RepeatIcon } from '@chakra-ui/icons'
import { useState } from 'react'
import { createConfig, deleteConfig } from '../api'

function ConfigList({ configs, currentConfig, onSelect, onRefresh }) {
  const { isOpen, onOpen, onClose } = useDisclosure()
  const [newConfigName, setNewConfigName] = useState('')
  const [creating, setCreating] = useState(false)
  const toast = useToast()

  const handleCreate = async () => {
    if (!newConfigName.trim()) {
      toast({
        title: '请输入配置文件名',
        status: 'warning',
        duration: 2000,
      })
      return
    }

    const filename = newConfigName.endsWith('.yaml') ? newConfigName : newConfigName + '.yaml'

    try {
      setCreating(true)
      await createConfig(filename, {
        server: { hostname: [], hca: [] },
        client: { hostname: [], hca: [] },
      })
      toast({
        title: '创建成功！',
        status: 'success',
        duration: 2000,
      })
      onClose()
      setNewConfigName('')
      onRefresh()
      onSelect(filename)
    } catch (error) {
      toast({
        title: '创建失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    } finally {
      setCreating(false)
    }
  }

  const handleDelete = async (name, isDefault, isDeletable) => {
    if (isDefault || !isDeletable) {
      toast({
        title: '无法删除',
        description: '默认配置文件不可删除',
        status: 'warning',
        duration: 2000,
      })
      return
    }

    if (!window.confirm(`确定要删除配置 "${name}" 吗？`)) {
      return
    }

    try {
      await deleteConfig(name)
      toast({
        title: '删除成功！',
        status: 'success',
        duration: 2000,
      })
      onRefresh()
    } catch (error) {
      toast({
        title: '删除失败',
        description: error.message,
        status: 'error',
        duration: 3000,
      })
    }
  }

  return (
    <Box w="280px" bg="gray.100" borderRight="1px" borderColor="gray.200" display="flex" flexDirection="column">
      {/* Header */}
      <Box p={4} borderBottom="1px" borderColor="gray.200">
        <HStack spacing={2}>
          <Button leftIcon={<AddIcon />} colorScheme="blue" size="sm" flex={1} onClick={onOpen}>
            新建配置
          </Button>
          <IconButton
            icon={<RepeatIcon />}
            size="sm"
            onClick={onRefresh}
            aria-label="刷新列表"
          />
        </HStack>
      </Box>

      {/* List */}
      <VStack flex={1} overflowY="auto" spacing={1} p={2} align="stretch">
        {configs.map((config) => (
          <HStack
            key={config.name}
            p={3}
            bg={currentConfig === config.name ? 'blue.500' : 'white'}
            color={currentConfig === config.name ? 'white' : 'gray.800'}
            borderRadius="md"
            cursor="pointer"
            onClick={() => onSelect(config.name)}
            _hover={{ bg: currentConfig === config.name ? 'blue.600' : 'gray.50' }}
            justify="space-between"
          >
            <HStack spacing={2} flex={1}>
              <Text fontSize="lg">{config.is_default ? '⭐' : '📄'}</Text>
              <Text fontSize="sm" fontWeight="medium" isTruncated>
                {config.name}
              </Text>
            </HStack>
            {config.is_deletable && (
              <IconButton
                icon={<DeleteIcon />}
                size="xs"
                colorScheme="red"
                variant="ghost"
                aria-label="删除"
                onClick={(e) => {
                  e.stopPropagation()
                  handleDelete(config.name, config.is_default, config.is_deletable)
                }}
              />
            )}
          </HStack>
        ))}
      </VStack>

      {/* Create Modal */}
      <Modal isOpen={isOpen} onClose={onClose}>
        <ModalOverlay />
        <ModalContent>
          <ModalHeader>创建配置文件</ModalHeader>
          <ModalCloseButton />
          <ModalBody>
            <FormControl>
              <FormLabel>文件名</FormLabel>
              <Input
                placeholder="例如: my-config.yaml"
                value={newConfigName}
                onChange={(e) => setNewConfigName(e.target.value)}
              />
            </FormControl>
          </ModalBody>
          <ModalFooter>
            <Button variant="ghost" mr={3} onClick={onClose}>
              取消
            </Button>
            <Button colorScheme="blue" onClick={handleCreate} isLoading={creating}>
              创建
            </Button>
          </ModalFooter>
        </ModalContent>
      </Modal>
    </Box>
  )
}

export default ConfigList
