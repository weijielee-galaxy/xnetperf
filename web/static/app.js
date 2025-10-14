// 纯 JavaScript 实现的配置管理应用
const app = {
    configs: [],
    currentConfig: null,
    formData: {},
    originalData: {},
    
    // 初始化
    init() {
        this.loadConfigs();
    },
    
    // 加载配置列表
    async loadConfigs() {
        try {
            const response = await fetch('/api/configs');
            
            // 检查响应状态
            if (!response.ok) {
                throw new Error(`HTTP error! status: ${response.status}`);
            }
            
            // 获取响应文本用于调试
            const text = await response.text();
            console.log('API Response:', text);
            
            // 解析 JSON
            const result = JSON.parse(text);
            
            // API 返回格式: {code: 0, message: "success", data: [...]}
            if (result.code === 0 && result.data) {
                // data 是配置文件信息数组，提取文件名
                this.configs = result.data.map(item => item.name);
            } else {
                this.configs = [];
            }
            
            this.renderConfigList();
        } catch (error) {
            console.error('Load configs error:', error);
            this.showMessage('加载配置列表失败: ' + error.message, 'error');
        }
    },
    
    // 渲染配置列表
    renderConfigList() {
        const list = document.getElementById('configList');
        if (this.configs.length === 0) {
            list.innerHTML = '<div style="text-align: center; padding: 20px; color: #909399;">暂无配置文件</div>';
            return;
        }
        
        list.innerHTML = '';
        this.configs.forEach(config => {
            const item = document.createElement('div');
            item.className = 'config-item' + (config === this.currentConfig ? ' active' : '');
            item.onclick = () => this.selectConfig(config);
            
            item.innerHTML = `
                <div class="config-item-name">
                    <span>📄</span>
                    <span>${config}</span>
                </div>
                <button class="btn btn-danger" onclick="event.stopPropagation(); app.deleteConfig('${config}')">
                    🗑️
                </button>
            `;
            
            list.appendChild(item);
        });
    },
    
    // 选择配置
    async selectConfig(name) {
        try {
            const response = await fetch(`/api/configs/${name}`);
            const result = await response.json();
            
            // API 返回格式: {code: 0, message: "success", data: {...}}
            if (result.code !== 0) {
                this.showMessage('加载配置失败: ' + result.message, 'error');
                return;
            }
            
            console.log('Loaded config data:', result.data); // 调试输出
            
            this.currentConfig = name;
            this.formData = result.data;
            this.originalData = JSON.parse(JSON.stringify(result.data));
            
            // 显示编辑器，隐藏空状态
            document.getElementById('emptyState').classList.add('hidden');
            document.getElementById('configEditor').classList.remove('hidden');
            document.getElementById('currentConfigName').textContent = name;
            
            // 填充表单
            this.fillForm();
            this.renderConfigList();
        } catch (error) {
            this.showMessage('加载配置失败: ' + error.message, 'error');
        }
    },
    
    // 填充表单
    fillForm() {
        const data = this.formData;
        
        console.log('Filling form with data:', data); // 调试输出
        
        // 基础字段 - 统一使用 snake_case
        document.getElementById('start_port').value = data.start_port ?? '';
        document.getElementById('stream_type').value = data.stream_type || 'fullmesh';
        document.getElementById('qp_num').value = data.qp_num ?? '';
        document.getElementById('message_size_bytes').value = data.message_size_bytes ?? '';
        document.getElementById('output_base').value = data.output_base || '';
        document.getElementById('waiting_time_seconds').value = data.waiting_time_seconds ?? '';
        document.getElementById('speed').value = data.speed ?? '';
        document.getElementById('rdma_cm').checked = data.rdma_cm || false;
        
        // 报告配置
        document.getElementById('report_enable').checked = data.report?.enable || false;
        document.getElementById('report_dir').value = data.report?.dir || '';
        
        // 运行配置
        document.getElementById('run_infinitely').checked = data.run?.infinitely || false;
        document.getElementById('run_duration_seconds').value = data.run?.duration_seconds ?? '';
        
        // 服务器配置
        this.renderTagList('server_hostname_list', data.server?.hostname || [], 'server_hostname');
        this.renderTagList('server_hca_list', data.server?.hca || [], 'server_hca');
        
        // 客户端配置
        this.renderTagList('client_hostname_list', data.client?.hostname || [], 'client_hostname');
        this.renderTagList('client_hca_list', data.client?.hca || [], 'client_hca');
    },
    
    // 渲染标签列表
    renderTagList(containerId, items, type) {
        const container = document.getElementById(containerId);
        container.innerHTML = '';
        
        items.forEach((item, index) => {
            const tag = document.createElement('span');
            tag.className = 'tag';
            tag.innerHTML = `
                ${item}
                <span class="tag-close" onclick="app.removeTag('${type}', ${index})">✕</span>
            `;
            container.appendChild(tag);
        });
        
        // 添加输入框和按钮
        const input = document.createElement('input');
        input.type = 'text';
        input.className = 'tag-input';
        input.placeholder = '添加...';
        input.onkeypress = (e) => {
            if (e.key === 'Enter') {
                e.preventDefault();
                this.addTag(type, input.value);
                input.value = '';
            }
        };
        
        const btn = document.createElement('button');
        btn.type = 'button';
        btn.className = 'tag-add-btn';
        btn.textContent = '添加';
        btn.onclick = () => {
            this.addTag(type, input.value);
            input.value = '';
        };
        
        container.appendChild(input);
        container.appendChild(btn);
    },
    
    // 添加标签
    addTag(type, value) {
        if (!value || !value.trim()) return;
        
        // type 格式: 'server_hostname' 或 'client_hca'
        const [section, field] = type.split('_');
        
        if (!this.formData[section]) {
            this.formData[section] = {};
        }
        if (!this.formData[section][field]) {
            this.formData[section][field] = [];
        }
        
        this.formData[section][field].push(value.trim());
        this.renderTagList(`${type}_list`, this.formData[section][field], type);
    },
    
    // 删除标签
    removeTag(type, index) {
        const [section, field] = type.split('_');
        this.formData[section][field].splice(index, 1);
        this.renderTagList(`${type}_list`, this.formData[section][field], type);
    },
    
    // 收集表单数据
    collectFormData() {
        return {
            start_port: parseInt(document.getElementById('start_port').value) || 0,
            stream_type: document.getElementById('stream_type').value,
            qp_num: parseInt(document.getElementById('qp_num').value) || 0,
            message_size_bytes: parseInt(document.getElementById('message_size_bytes').value) || 0,
            output_base: document.getElementById('output_base').value,
            waiting_time_seconds: parseInt(document.getElementById('waiting_time_seconds').value) || 0,
            speed: parseFloat(document.getElementById('speed').value) || 0,
            rdma_cm: document.getElementById('rdma_cm').checked,
            report: {
                enable: document.getElementById('report_enable').checked,
                dir: document.getElementById('report_dir').value
            },
            run: {
                infinitely: document.getElementById('run_infinitely').checked,
                duration_seconds: parseInt(document.getElementById('run_duration_seconds').value) || 0
            },
            server: {
                hostname: this.formData.server?.hostname || [],
                hca: this.formData.server?.hca || []
            },
            client: {
                hostname: this.formData.client?.hostname || [],
                hca: this.formData.client?.hca || []
            }
        };
    },
    
    // 保存配置
    async saveConfig() {
        if (!this.currentConfig) return;
        
        const data = this.collectFormData();
        
        try {
            const response = await fetch(`/api/configs/${this.currentConfig}`, {
                method: 'PUT',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify(data)
            });
            
            const result = await response.json();
            
            if (result.code === 0) {
                this.showMessage('保存成功！', 'success');
                this.originalData = JSON.parse(JSON.stringify(data));
            } else {
                this.showMessage('保存失败: ' + result.message, 'error');
            }
        } catch (error) {
            this.showMessage('保存失败: ' + error.message, 'error');
        }
    },
    
    // 取消编辑
    cancelEdit() {
        if (this.currentConfig) {
            this.formData = JSON.parse(JSON.stringify(this.originalData));
            this.fillForm();
            this.showMessage('已取消修改', 'warning');
        }
    },
    
    // 验证配置
    async validateConfig() {
        if (!this.currentConfig) return;
        
        try {
            const response = await fetch(`/api/configs/${this.currentConfig}/validate`, {
                method: 'POST'
            });
            const result = await response.json();
            
            if (result.code === 0) {
                this.showMessage('✓ 配置验证通过！', 'success');
            } else {
                this.showMessage('✗ 配置验证失败: ' + result.message, 'error');
            }
        } catch (error) {
            this.showMessage('验证失败: ' + error.message, 'error');
        }
    },
    
    // 预览配置
    previewConfig() {
        if (!this.currentConfig) return;
        
        const data = this.collectFormData();
        const yaml = this.toYAML(data);
        
        document.getElementById('previewCode').textContent = yaml;
        document.getElementById('previewModal').classList.add('show');
    },
    
    // 隐藏预览对话框
    hidePreviewDialog() {
        document.getElementById('previewModal').classList.remove('show');
    },
    
    // 转换为 YAML 格式
    toYAML(obj, indent = 0) {
        const spaces = '    '.repeat(indent);
        let yaml = '';
        
        for (const [key, value] of Object.entries(obj)) {
            if (value === null || value === undefined) {
                continue;
            }
            
            if (typeof value === 'object' && !Array.isArray(value)) {
                yaml += `${spaces}${key}:\n`;
                yaml += this.toYAML(value, indent + 1);
            } else if (Array.isArray(value)) {
                yaml += `${spaces}${key}:\n`;
                if (value.length === 0) {
                    yaml += `${spaces}    []\n`;
                } else {
                    value.forEach(item => {
                        yaml += `${spaces}    - ${item}\n`;
                    });
                }
            } else if (typeof value === 'boolean') {
                yaml += `${spaces}${key}: ${value}\n`;
            } else if (typeof value === 'number') {
                yaml += `${spaces}${key}: ${value}\n`;
            } else {
                yaml += `${spaces}${key}: ${value}\n`;
            }
        }
        
        return yaml;
    },
    
    // 显示创建对话框
    showCreateDialog() {
        document.getElementById('createModal').classList.add('show');
        document.getElementById('newConfigName').value = '';
        document.getElementById('newConfigName').focus();
    },
    
    // 隐藏创建对话框
    hideCreateDialog() {
        document.getElementById('createModal').classList.remove('show');
    },
    
    // 创建配置
    async createConfig() {
        const name = document.getElementById('newConfigName').value.trim();
        if (!name) {
            this.showMessage('请输入配置文件名', 'warning');
            return;
        }
        
        const filename = name.endsWith('.yaml') ? name : name + '.yaml';
        
        try {
            const response = await fetch('/api/configs', {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                    name: filename,
                    config: {
                        start_port: 20000,
                        stream_type: 'fullmesh',
                        qp_num: 1,
                        message_size_bytes: 8192,
                        output_base: './output',
                        waiting_time_seconds: 5,
                        speed: 0,
                        rdma_cm: false,
                        report: { enable: true, dir: './reports' },
                        run: { infinitely: false, duration_seconds: 60 },
                        server: { hostname: [], hca: [] },
                        client: { hostname: [], hca: [] }
                    }
                })
            });
            
            const result = await response.json();
            
            if (result.code === 0) {
                this.showMessage('创建成功！', 'success');
                this.hideCreateDialog();
                await this.loadConfigs();
                await this.selectConfig(filename);
            } else {
                this.showMessage('创建失败: ' + result.message, 'error');
            }
        } catch (error) {
            this.showMessage('创建失败: ' + error.message, 'error');
        }
    },
    
    // 删除配置
    async deleteConfig(name) {
        if (!confirm(`确定要删除配置 "${name}" 吗？`)) {
            return;
        }
        
        try {
            const response = await fetch(`/api/configs/${name}`, {
                method: 'DELETE'
            });
            
            const result = await response.json();
            
            if (result.code === 0) {
                this.showMessage('删除成功！', 'success');
                if (this.currentConfig === name) {
                    this.currentConfig = null;
                    document.getElementById('emptyState').classList.remove('hidden');
                    document.getElementById('configEditor').classList.add('hidden');
                }
                await this.loadConfigs();
            } else {
                this.showMessage('删除失败: ' + result.message, 'error');
            }
        } catch (error) {
            this.showMessage('删除失败: ' + error.message, 'error');
        }
    },
    
    // 显示消息
    showMessage(text, type = 'success') {
        const message = document.createElement('div');
        message.className = `message message-${type}`;
        message.textContent = text;
        document.body.appendChild(message);
        
        setTimeout(() => {
            message.remove();
        }, 3000);
    }
};

// 页面加载完成后初始化
document.addEventListener('DOMContentLoaded', () => {
    app.init();
});
