<script>
	import { onMount } from 'svelte';
	import { Lock, Play, Square, RefreshCw, Settings, Users, Activity, Shield, Globe, Plus, Download, Copy, Eye, EyeOff, Trash2, Edit } from 'lucide-svelte';
	import { metaBaseAPI } from '$lib/api';

	// 响应式状态
	let status = {
		enabled: false,
		running: false,
		config: {},
		stats: {}
	};

	let clients = [];
	let connections = [];
	let loading = true;
	let error = null;
	let showConfigForm = false;
	let showClientForm = false;
	let editingClient = null;
	let newClient = {
		id: '',
		name: '',
		password: '',
		status: 'active',
		expires_at: null,
		data_limit: 0,
		ip_whitelist: [],
		tags: []
	};

	let showPassword = {};
	let configForm = {
		enabled: false,
		host: '0.0.0.0',
		port: 8443,
		password: '',
		server_name: 'localhost',
		auto_cert: true,
		max_clients: 1000,
		log_level: 'info'
	};

	// 页面挂载时加载数据
	onMount(async () => {
		await loadStatus();
		await loadClients();
		await loadConnections();
		loading = false;
	});

	// 加载服务状态
	async function loadStatus() {
		try {
			const response = await metaBaseAPI.get('/admin/trojan/status');
			status = response;
			configForm = { ...configForm, ...response.config };
		} catch (err) {
			error = 'Failed to load Trojan status: ' + err.message;
		}
	}

	// 加载客户端列表
	async function loadClients() {
		try {
			const response = await metaBaseAPI.get('/admin/trojan/clients');
			clients = response.clients || [];
		} catch (err) {
			error = 'Failed to load clients: ' + err.message;
		}
	}

	// 加载连接列表
	async function loadConnections() {
		try {
			const response = await metaBaseAPI.get('/admin/trojan/connections');
			connections = response.connections || [];
		} catch (err) {
			error = 'Failed to load connections: ' + err.message;
		}
	}

	// 启动服务
	async function startService() {
		loading = true;
		try {
			await metaBaseAPI.post('/admin/trojan/start', { config: configForm });
			await loadStatus();
			error = null;
		} catch (err) {
			error = 'Failed to start Trojan service: ' + err.message;
		}
		loading = false;
	}

	// 停止服务
	async function stopService() {
		loading = true;
		try {
			await metaBaseAPI.post('/admin/trojan/stop');
			await loadStatus();
			error = null;
		} catch (err) {
			error = 'Failed to stop Trojan service: ' + err.message;
		}
		loading = false;
	}

	// 重启服务
	async function restartService() {
		loading = true;
		try {
			await metaBaseAPI.post('/admin/trojan/restart', { config: configForm });
			await loadStatus();
			error = null;
		} catch (err) {
			error = 'Failed to restart Trojan service: ' + err.message;
		}
		loading = false;
	}

	// 更新配置
	async function updateConfig() {
		loading = true;
		try {
			await metaBaseAPI.put('/admin/trojan/config', { config: configForm });
			await loadStatus();
			showConfigForm = false;
			error = null;
		} catch (err) {
			error = 'Failed to update configuration: ' + err.message;
		}
		loading = false;
	}

	// 添加客户端
	async function addClient() {
		loading = true;
		try {
			// 生成密码
			if (!newClient.password) {
				const passwordResponse = await metaBaseAPI.post('/admin/trojan/generate-client-password');
				newClient.password = passwordResponse.password;
			}

			await metaBaseAPI.post('/admin/trojan/clients', { client: newClient });
			await loadClients();
			showClientForm = false;
			resetClientForm();
			error = null;
		} catch (err) {
			error = 'Failed to add client: ' + err.message;
		}
		loading = false;
	}

	// 删除客户端
	async function removeClient(clientId) {
		if (!confirm('确定要删除这个客户端吗？')) return;

		loading = true;
		try {
			await metaBaseAPI.delete(`/admin/trojan/clients/${clientId}`);
			await loadClients();
			error = null;
		} catch (err) {
			error = 'Failed to remove client: ' + err.message;
		}
		loading = false;
	}

	// 下载客户端配置
	async function downloadClientConfig(clientId) {
		loading = true;
		try {
			const config = await metaBaseAPI.get(`/admin/trojan/clients/${clientId}/config`);
			const blob = new Blob([JSON.stringify(config, null, 2)], { type: 'application/json' });
			const url = URL.createObjectURL(blob);
			const a = document.createElement('a');
			a.href = url;
			a.download = `trojan-client-${clientId}.json`;
			a.click();
			URL.revokeObjectURL(url);
		} catch (err) {
			error = 'Failed to download client config: ' + err.message;
		}
		loading = false;
	}

	// 复制配置到剪贴板
	async function copyToClipboard(text) {
		try {
			await navigator.clipboard.writeText(text);
		} catch (err) {
			console.error('Failed to copy to clipboard:', err);
		}
	}

	// 重置客户端表单
	function resetClientForm() {
		newClient = {
			id: '',
			name: '',
			password: '',
			status: 'active',
			expires_at: null,
			data_limit: 0,
			ip_whitelist: [],
			tags: []
		};
		editingClient = null;
	}

	// 格式化数据大小
	function formatBytes(bytes) {
		if (bytes === 0) return '0 B';
		const k = 1024;
		const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
		const i = Math.floor(Math.log(bytes) / Math.log(k));
		return parseFloat((bytes / Math.pow(k, i)).toFixed(2)) + ' ' + sizes[i];
	}

	// 格式化时间
	function formatTime(timestamp) {
		if (!timestamp) return 'N/A';
		return new Date(timestamp).toLocaleString();
	}

	// 切换密码显示
	function togglePasswordVisibility(clientId) {
		showPassword[clientId] = !showPassword[clientId];
	}

	// 定期刷新状态
	let refreshInterval;
	onMount(() => {
		refreshInterval = setInterval(() => {
			if (status.running) {
				loadStatus();
				loadConnections();
			}
		}, 5000);
	});

	// 清理定时器
	$: {
		if (typeof window !== 'undefined') {
			return () => {
				if (refreshInterval) {
					clearInterval(refreshInterval);
				}
			};
		}
	}
</script>

<svelte:head>
	<Title>Trojan VPN 管理 - MetaBase</Title>
</svelte:head>

<div class="vpn-management">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-content">
			<div class="header-left">
				<Lock size={32} class="header-icon" />
				<div>
					<h1>Trojan VPN 管理</h1>
					<p>管理和配置 Trojan VPN 代理服务</p>
				</div>
			</div>
			<div class="header-right">
				{#if loading}
					<RefreshCw class="animate-spin" size={20} />
				{:else}
					<RefreshCw class="refresh-btn" on:click={loadStatus} size={20} />
				{/if}
			</div>
		</div>
	</div>

	{#if error}
		<div class="error-banner">
			<span>{error}</span>
		</div>
	{/if}

	<!-- 服务状态卡片 -->
	<div class="status-cards">
		<!-- 服务状态 -->
		<div class="status-card">
			<div class="card-header">
				<Shield size={24} />
				<h3>服务状态</h3>
			</div>
			<div class="card-content">
				<div class="status-indicator">
					<span class="status-dot" class:active={status.running} class:inactive={!status.running}></span>
					<span class="status-text">{status.running ? '运行中' : '已停止'}</span>
				</div>
				<div class="service-controls">
					{#if !status.running}
						<button class="btn btn-primary" on:click={startService} disabled={loading}>
							<Play size={16} />
							启动服务
						</button>
					{:else}
						<button class="btn btn-warning" on:click={restartService} disabled={loading}>
							<RefreshCw size={16} />
							重启服务
						</button>
						<button class="btn btn-danger" on:click={stopService} disabled={loading}>
							<Square size={16} />
							停止服务
						</button>
					{/if}
				</div>
			</div>
		</div>

		<!-- 连接统计 -->
		<div class="status-card">
			<div class="card-header">
				<Activity size={24} />
				<h3>连接统计</h3>
			</div>
			<div class="card-content">
				<div class="stat-row">
					<span class="stat-label">活跃连接:</span>
					<span class="stat-value">{status.stats?.active_connections || 0}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">总连接数:</span>
					<span class="stat-value">{status.stats?.total_connections || 0}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">上传流量:</span>
					<span class="stat-value">{formatBytes(status.stats?.upload_bytes || 0)}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">下载流量:</span>
					<span class="stat-value">{formatBytes(status.stats?.download_bytes || 0)}</span>
				</div>
			</div>
		</div>

		<!-- 服务配置 -->
		<div class="status-card">
			<div class="card-header">
				<Settings size={24} />
				<h3>服务配置</h3>
			</div>
			<div class="card-content">
				<div class="stat-row">
					<span class="stat-label">监听地址:</span>
					<span class="stat-value">{status.config?.host || 'N/A'}:{status.config?.port || 'N/A'}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">最大客户端:</span>
					<span class="stat-value">{status.config?.max_clients || 'N/A'}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">TLS 启用:</span>
					<span class="stat-value">{status.config?.enable_tls ? '是' : '否'}</span>
				</div>
				<div class="stat-row">
					<span class="stat-label">自动证书:</span>
					<span class="stat-value">{status.config?.auto_cert ? '是' : '否'}</span>
				</div>
			</div>
		</div>
	</div>

	<!-- 操作按钮 -->
	<div class="action-bar">
		<button class="btn btn-primary" on:click={() => showConfigForm = true}>
			<Settings size={16} />
			配置管理
		</button>
		<button class="btn btn-primary" on:click={() => { showClientForm = true; resetClientForm(); }}>
			<Plus size={16} />
			添加客户端
		</button>
	</div>

	<!-- 客户端管理 -->
	<div class="clients-section">
		<div class="section-header">
			<Users size={20} />
			<h3>客户端管理</h3>
			<span class="client-count">({clients.length} 个客户端)</span>
		</div>

		<div class="clients-table">
			<table>
				<thead>
					<tr>
						<th>名称</th>
						<th>ID</th>
						<th>密码</th>
						<th>状态</th>
						<th>流量使用</th>
						<th>最后活动</th>
						<th>操作</th>
					</tr>
				</thead>
				<tbody>
					{#each clients as client}
						<tr>
							<td class="client-name">{client.name}</td>
							<td class="client-id">{client.id}</td>
							<td class="client-password">
								<div class="password-field">
									<span class:password-hidden={!showPassword[client.id]}>
										{showPassword[client.id] ? client.password : '●●●●●●●●'}
									</span>
									<button
										class="password-toggle"
										on:click={() => togglePasswordVisibility(client.id)}
									>
										{#if showPassword[client.id]}
											<EyeOff size={16} />
										{:else}
											<Eye size={16} />
										{/if}
									</button>
								</div>
							</td>
							<td>
								<span class="status-badge" class:status-active={client.status === 'active'}>
									{client.status}
								</span>
							</td>
							<td>{formatBytes(client.data_used || 0)} / {formatBytes(client.data_limit || 0)}</td>
							<td>{formatTime(client.last_seen)}</td>
							<td class="actions">
								<button
									class="btn-icon"
									title="下载配置"
									on:click={() => downloadClientConfig(client.id)}
								>
									<Download size={16} />
								</button>
								<button
									class="btn-icon"
									title="复制密码"
									on:click={() => copyToClipboard(client.password)}
								>
									<Copy size={16} />
								</button>
								<button
									class="btn-icon btn-danger"
									title="删除客户端"
									on:click={() => removeClient(client.id)}
								>
									<Trash2 size={16} />
								</button>
							</td>
						</tr>
					{:else}
						<tr>
							<td colspan="7" class="no-clients">暂无客户端</td>
						</tr>
					{/each}
				</tbody>
			</table>
		</div>
	</div>

	<!-- 活跃连接 -->
	{#if status.running && connections.length > 0}
		<div class="connections-section">
			<div class="section-header">
				<Activity size={20} />
				<h3>活跃连接</h3>
				<span class="connection-count">({connections.length} 个连接)</span>
			</div>

			<div class="connections-table">
				<table>
					<thead>
						<tr>
							<th>客户端ID</th>
							<th>远程地址</th>
							<th>目标地址</th>
							<th>连接时间</th>
							<th>上传流量</th>
							<th>下载流量</th>
							<th>状态</th>
						</tr>
					</thead>
					<tbody>
						{#each connections as conn}
							<tr>
								<td>{conn.client_id}</td>
								<td>{conn.remote_addr}</td>
								<td>{conn.target_addr}</td>
								<td>{formatTime(conn.created_at)}</td>
								<td>{formatBytes(conn.bytes_sent || 0)}</td>
								<td>{formatBytes(conn.bytes_recv || 0)}</td>
								<td>
									<span class="status-badge status-active">
										{conn.status}
									</span>
								</td>
							</tr>
						{/each}
					</tbody>
				</table>
			</div>
		</div>
	{/if}
</div>

<!-- 配置表单模态框 -->
{#if showConfigForm}
	<div class="modal-overlay" on:click={() => showConfigForm = false}>
		<div class="modal" on:click|stopPropagation>
			<div class="modal-header">
				<h3>服务配置</h3>
				<button class="modal-close" on:click={() => showConfigForm = false}>×</button>
			</div>
			<div class="modal-body">
				<form on:submit|preventDefault={updateConfig}>
					<div class="form-group">
						<label for="enabled">启用服务</label>
						<input type="checkbox" id="enabled" bind:checked={configForm.enabled} />
					</div>
					<div class="form-group">
						<label for="host">监听地址</label>
						<input type="text" id="host" bind:value={configForm.host} required />
					</div>
					<div class="form-group">
						<label for="port">监听端口</label>
						<input type="number" id="port" bind:value={configForm.port} min="1" max="65535" required />
					</div>
					<div class="form-group">
						<label for="password">服务密码</label>
						<input type="password" id="password" bind:value={configForm.password} />
					</div>
					<div class="form-group">
						<label for="server_name">服务器名称</label>
						<input type="text" id="server_name" bind:value={configForm.server_name} />
					</div>
					<div class="form-group">
						<label for="max_clients">最大客户端数</label>
						<input type="number" id="max_clients" bind:value={configForm.max_clients} min="1" />
					</div>
					<div class="form-group">
						<label for="log_level">日志级别</label>
						<select id="log_level" bind:value={configForm.log_level}>
							<option value="debug">Debug</option>
							<option value="info">Info</option>
							<option value="warn">Warning</option>
							<option value="error">Error</option>
						</select>
					</div>
					<div class="form-group">
						<label for="auto_cert">自动证书</label>
						<input type="checkbox" id="auto_cert" bind:checked={configForm.auto_cert} />
					</div>
				</form>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" on:click={() => showConfigForm = false}>取消</button>
				<button class="btn btn-primary" on:click={updateConfig} disabled={loading}>
					{loading ? '保存中...' : '保存'}
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- 客户端表单模态框 -->
{#if showClientForm}
	<div class="modal-overlay" on:click={() => showClientForm = false}>
		<div class="modal" on:click|stopPropagation>
			<div class="modal-header">
				<h3>{editingClient ? '编辑客户端' : '添加客户端'}</h3>
				<button class="modal-close" on:click={() => showClientForm = false}>×</button>
			</div>
			<div class="modal-body">
				<form on:submit|preventDefault={addClient}>
					<div class="form-group">
						<label for="clientName">客户端名称</label>
						<input type="text" id="clientName" bind:value={newClient.name} required />
					</div>
					<div class="form-group">
						<label for="clientPassword">密码</label>
						<div class="password-input-group">
							<input type="password" id="clientPassword" bind:value={newClient.password} />
							<button
								type="button"
								class="btn btn-secondary"
								on:click={async () => {
									const response = await metaBaseAPI.post('/admin/trojan/generate-client-password');
									newClient.password = response.password;
								}}
							>
								生成密码
							</button>
						</div>
					</div>
					<div class="form-group">
						<label for="clientStatus">状态</label>
						<select id="clientStatus" bind:value={newClient.status}>
							<option value="active">活跃</option>
							<option value="disabled">禁用</option>
						</select>
					</div>
					<div class="form-group">
						<label for="clientDataLimit">流量限制 (MB)</label>
						<input type="number" id="clientDataLimit" bind:value={newClient.data_limit} min="0" />
					</div>
				</form>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" on:click={() => showClientForm = false}>取消</button>
				<button class="btn btn-primary" on:click={addClient} disabled={loading}>
					{loading ? '添加中...' : '添加'}
				</button>
			</div>
		</div>
	</div>
{/if}

<style>
	.vpn-management {
		padding: 1rem;
		max-width: 1400px;
		margin: 0 auto;
	}

	.page-header {
		margin-bottom: 2rem;
	}

	.header-content {
		display: flex;
		justify-content: space-between;
		align-items: center;
		background: white;
		padding: 1.5rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		border: 1px solid #e5e7eb;
	}

	.header-left {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.header-icon {
		color: #3b82f6;
	}

	.header-left h1 {
		margin: 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.header-left p {
		margin: 0.25rem 0 0 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.refresh-btn {
		cursor: pointer;
		color: #6b7280;
		transition: color 0.2s;
	}

	.refresh-btn:hover {
		color: #374151;
	}

	.error-banner {
		background: #fef2f2;
		border: 1px solid #fecaca;
		color: #dc2626;
		padding: 1rem;
		border-radius: 8px;
		margin-bottom: 1.5rem;
	}

	.status-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 1.5rem;
		margin-bottom: 2rem;
	}

	.status-card {
		background: white;
		padding: 1.5rem;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		border: 1px solid #e5e7eb;
	}

	.card-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		margin-bottom: 1rem;
		color: #3b82f6;
	}

	.card-header h3 {
		margin: 0;
		font-size: 1.125rem;
		font-weight: 600;
	}

	.status-indicator {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		margin-bottom: 1rem;
	}

	.status-dot {
		width: 8px;
		height: 8px;
		border-radius: 50%;
		display: inline-block;
	}

	.status-dot.active {
		background: #10b981;
		box-shadow: 0 0 0 2px rgba(16, 185, 129, 0.2);
	}

	.status-dot.inactive {
		background: #ef4444;
		box-shadow: 0 0 0 2px rgba(239, 68, 68, 0.2);
	}

	.status-text {
		font-weight: 500;
		color: #374151;
	}

	.service-controls {
		display: flex;
		gap: 0.75rem;
		flex-wrap: wrap;
	}

	.stat-row {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.5rem 0;
		border-bottom: 1px solid #f3f4f6;
	}

	.stat-row:last-child {
		border-bottom: none;
	}

	.stat-label {
		color: #6b7280;
		font-size: 0.875rem;
	}

	.stat-value {
		font-weight: 600;
		color: #111827;
	}

	.action-bar {
		display: flex;
		gap: 1rem;
		margin-bottom: 2rem;
	}

	.clients-section,
	.connections-section {
		background: white;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		border: 1px solid #e5e7eb;
		margin-bottom: 2rem;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
		color: #374151;
	}

	.section-header h3 {
		margin: 0;
		font-size: 1.125rem;
		font-weight: 600;
	}

	.client-count,
	.connection-count {
		color: #6b7280;
		font-size: 0.875rem;
	}

	.clients-table,
	.connections-table {
		overflow-x: auto;
	}

	table {
		width: 100%;
		border-collapse: collapse;
	}

	th {
		text-align: left;
		padding: 1rem;
		background: #f9fafb;
		border-bottom: 1px solid #e5e7eb;
		font-weight: 600;
		color: #374151;
		font-size: 0.875rem;
	}

	td {
		padding: 1rem;
		border-bottom: 1px solid #f3f4f6;
	}

	tr:last-child td {
		border-bottom: none;
	}

	.client-name {
		font-weight: 600;
		color: #111827;
	}

	.client-id {
		font-family: monospace;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.password-field {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.password-hidden {
		font-family: monospace;
		letter-spacing: 2px;
	}

	.password-toggle {
		background: none;
		border: none;
		cursor: pointer;
		color: #6b7280;
		padding: 0.25rem;
		border-radius: 4px;
	}

	.password-toggle:hover {
		background: #f3f4f6;
		color: #374151;
	}

	.status-badge {
		padding: 0.25rem 0.75rem;
		border-radius: 9999px;
		font-size: 0.75rem;
		font-weight: 600;
		text-transform: uppercase;
	}

	.status-badge.status-active {
		background: #dcfce7;
		color: #166534;
	}

	.actions {
		display: flex;
		gap: 0.5rem;
	}

	.btn-icon {
		background: none;
		border: none;
		cursor: pointer;
		padding: 0.5rem;
		border-radius: 6px;
		color: #6b7280;
		transition: all 0.2s;
	}

	.btn-icon:hover {
		background: #f3f4f6;
		color: #374151;
	}

	.btn-icon.btn-danger {
		color: #ef4444;
	}

	.btn-icon.btn-danger:hover {
		background: #fef2f2;
		color: #dc2626;
	}

	.no-clients {
		text-align: center;
		color: #6b7280;
		padding: 2rem;
	}

	/* 按钮样式 */
	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.75rem 1.25rem;
		border: none;
		border-radius: 8px;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s;
	}

	.btn-primary {
		background: #3b82f6;
		color: white;
	}

	.btn-primary:hover {
		background: #2563eb;
	}

	.btn-secondary {
		background: #6b7280;
		color: white;
	}

	.btn-secondary:hover {
		background: #4b5563;
	}

	.btn-warning {
		background: #f59e0b;
		color: white;
	}

	.btn-warning:hover {
		background: #d97706;
	}

	.btn-danger {
		background: #ef4444;
		color: white;
	}

	.btn-danger:hover {
		background: #dc2626;
	}

	.btn:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	/* 模态框样式 */
	.modal-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		display: flex;
		align-items: center;
		justify-content: center;
		z-index: 1000;
	}

	.modal {
		background: white;
		border-radius: 12px;
		width: 90%;
		max-width: 500px;
		max-height: 90vh;
		overflow-y: auto;
		box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
	}

	.modal-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.modal-header h3 {
		margin: 0;
		font-size: 1.25rem;
		font-weight: 600;
		color: #111827;
	}

	.modal-close {
		background: none;
		border: none;
		font-size: 1.5rem;
		cursor: pointer;
		color: #6b7280;
		padding: 0.25rem;
		line-height: 1;
	}

	.modal-body {
		padding: 1.5rem;
	}

	.modal-footer {
		display: flex;
		justify-content: flex-end;
		gap: 0.75rem;
		padding: 1.5rem;
		border-top: 1px solid #e5e7eb;
	}

	.form-group {
		margin-bottom: 1.25rem;
	}

	.form-group label {
		display: block;
		margin-bottom: 0.5rem;
		font-weight: 500;
		color: #374151;
		font-size: 0.875rem;
	}

	.form-group input,
	.form-group select {
		width: 100%;
		padding: 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s;
	}

	.form-group input:focus,
	.form-group select:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.form-group input[type="checkbox"] {
		width: auto;
		margin-right: 0.5rem;
	}

	.password-input-group {
		display: flex;
		gap: 0.75rem;
	}

	.password-input-group input {
		flex: 1;
	}

	/* 动画 */
	.animate-spin {
		animation: spin 1s linear infinite;
	}

	@keyframes spin {
		from {
			transform: rotate(0deg);
		}
		to {
			transform: rotate(360deg);
		}
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.vpn-management {
			padding: 0.5rem;
		}

		.header-content {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.status-cards {
			grid-template-columns: 1fr;
		}

		.service-controls {
			flex-direction: column;
		}

		.action-bar {
			flex-direction: column;
		}

		.clients-table,
		.connections-table {
			font-size: 0.875rem;
		}

		th,
		td {
			padding: 0.5rem;
		}

		.actions {
			flex-direction: column;
		}

		.modal {
			width: 95%;
			margin: 1rem;
		}

		.password-input-group {
			flex-direction: column;
		}
	}
</style>