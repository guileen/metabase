<script>
	import { onMount } from 'svelte';
	import { Settings, Database, Shield, Bell, Globe, Key, Save, RefreshCw, Download, Upload, Eye, EyeOff } from 'lucide-svelte';

	// 配置分类
	let activeCategory = 'general';

	// 配置数据
	let config = {
		general: {
			siteName: 'MetaBase',
			siteDescription: 'Next Generation Backend Server',
			adminEmail: 'admin@metabase.com',
			timezone: 'Asia/Shanghai',
			language: 'zh-CN',
			debugMode: false,
			maintenanceMode: false
		},
		server: {
			port: 7609,
			host: '0.0.0.0',
			readTimeout: 30,
			writeTimeout: 30,
			maxConnections: 1000,
			enableHttps: false,
			sslCertPath: '',
			sslKeyPath: ''
		},
		database: {
			driver: 'sqlite',
			host: 'localhost',
			port: 5432,
			database: 'metabase',
			username: 'admin',
			password: '••••••••',
			maxConnections: 20,
			sslMode: 'disable'
		},
		security: {
			jwtSecret: '••••••••••••••••',
			sessionTimeout: 3600,
			maxLoginAttempts: 5,
			lockoutDuration: 900,
			enableTwoFactor: false,
			passwordPolicy: {
				minLength: 8,
				requireUppercase: true,
				requireLowercase: true,
				requireNumbers: true,
				requireSymbols: true
			}
		},
		logging: {
			level: 'info',
			filePath: '/var/log/metabase/app.log',
			maxSize: '100MB',
			maxBackups: 10,
			enableConsole: true,
			enableFile: true,
			format: 'json'
		},
		notifications: {
			enableEmail: false,
			smtpHost: '',
			smtpPort: 587,
			smtpUsername: '',
			smtpPassword: '••••••••',
			smtpUseTLS: true,
			adminNotifications: true,
			errorAlerts: true,
			performanceAlerts: false
		},
		cache: {
			provider: 'redis',
			host: 'localhost',
			port: 6379,
			password: '',
			db: 0,
			ttl: 3600,
			maxConnections: 10
		},
		api: {
			rateLimit: {
				enabled: true,
				requests: 100,
				window: 60
			},
			cors: {
				enabled: true,
				origins: ['*'],
				methods: ['GET', 'POST', 'PUT', 'DELETE'],
				headers: ['Content-Type', 'Authorization']
			},
			keyRequired: false,
			version: 'v1'
		}
	};

	// 原始配置用于重置
	let originalConfig = {};

	// 保存状态
	let saving = false;
	let savedMessage = false;

	// 密码显示状态
	let showPasswords = {};

	const categories = [
		{
			id: 'general',
			name: '基本设置',
			icon: Settings,
			description: '网站基本信息和通用配置'
		},
		{
			id: 'server',
			name: '服务器设置',
			icon: Globe,
			description: 'HTTP服务器和网络配置'
		},
		{
			id: 'database',
			name: '数据库配置',
			icon: Database,
			description: '数据库连接和配置参数'
		},
		{
			id: 'security',
			name: '安全设置',
			icon: Shield,
			description: '认证、授权和安全策略'
		},
		{
			id: 'logging',
			name: '日志配置',
			icon: Settings,
			description: '日志级别、格式和存储设置'
		},
		{
			id: 'notifications',
			name: '通知配置',
			icon: Bell,
			description: '邮件和系统通知设置'
		},
		{
			id: 'cache',
			name: '缓存配置',
			icon: Database,
			description: '缓存服务和性能优化'
		},
		{
			id: 'api',
			name: 'API配置',
			icon: Key,
			description: 'API访问控制和限制'
		}
	];

	onMount(() => {
		originalConfig = JSON.parse(JSON.stringify(config));
	});

	function togglePassword(field) {
		showPasswords[field] = !showPasswords[field];
	}

	function resetToDefault() {
		if (confirm('确定要重置为默认配置吗？此操作不可撤销。')) {
			config = JSON.parse(JSON.stringify(originalConfig));
		}
	}

	async function saveConfig() {
		saving = true;
		savedMessage = false;

		// 模拟保存配置
		await new Promise(resolve => setTimeout(resolve, 1500));

		// 更新原始配置
		originalConfig = JSON.parse(JSON.stringify(config));

		saving = false;
		savedMessage = true;

		// 3秒后隐藏成功消息
		setTimeout(() => {
			savedMessage = false;
		}, 3000);
	}

	function exportConfig() {
		const dataStr = JSON.stringify(config, null, 2);
		const blob = new Blob([dataStr], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `config_${new Date().toISOString().split('T')[0]}.json`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function importConfig() {
		const input = document.createElement('input');
		input.type = 'file';
		input.accept = '.json';
		input.onchange = async (e) => {
			const file = e.target.files[0];
			if (file) {
				try {
					const text = await file.text();
					const imported = JSON.parse(text);
					if (confirm('导入配置将覆盖当前设置，确定继续吗？')) {
						config = imported;
					}
				} catch (error) {
					alert('配置文件格式错误，请检查文件内容。');
				}
			}
		};
		input.click();
	}

	function getFieldName(field) {
		const fieldNames = {
			siteName: '网站名称',
			siteDescription: '网站描述',
			adminEmail: '管理员邮箱',
			port: '端口',
			host: '主机地址',
			database: '数据库名',
			username: '用户名',
			password: '密码',
			jwtSecret: 'JWT密钥',
			sessionTimeout: '会话超时'
		};
		return fieldNames[field] || field;
	}

	function isPasswordField(field) {
		return field.toLowerCase().includes('password') ||
			   field.toLowerCase().includes('secret') ||
			   field.toLowerCase().includes('key');
	}
</script>

<div class="config-page">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-left">
			<h1>配置中心</h1>
			<p class="page-description">系统参数和功能配置</p>
		</div>
		<div class="header-right">
			<button class="btn btn-secondary" on:click={importConfig}>
				<Upload size={16} />
				导入配置
			</button>
			<button class="btn btn-secondary" on:click={exportConfig}>
				<Download size={16} />
				导出配置
			</button>
			<button class="btn btn-secondary" on:click={resetToDefault}>
				<RefreshCw size={16} />
				重置默认
			</button>
			<button class="btn btn-primary" on:click={saveConfig} disabled={saving}>
				<Save size={16} />
				{saving ? '保存中...' : '保存配置'}
			</button>
		</div>
	</div>

	{#if savedMessage}
		<div class="success-message">
			配置保存成功！
		</div>
	{/if}

	<div class="config-content">
		<!-- 配置分类导航 -->
		<div class="category-sidebar">
			<div class="category-list">
				{#each categories as category}
					<button
						class="category-item"
						class:active={activeCategory === category.id}
						on:click={() => activeCategory = category.id}
					>
						<svelte:component this={category.icon} size={18} />
						<div class="category-info">
							<div class="category-name">{category.name}</div>
							<div class="category-desc">{category.description}</div>
						</div>
					</button>
				{/each}
			</div>
		</div>

		<!-- 配置表单 -->
		<div class="config-form">
			{#each categories as category}
				{#if category.id === activeCategory}
					<div class="form-section">
						<div class="section-header">
							<svelte:component this={category.icon} size={20} />
							<h2>{category.name}</h2>
							<p>{category.description}</p>
						</div>

						<div class="form-content">
							{#if category.id === 'general'}
								<div class="form-grid">
									<div class="form-group">
										<label for="siteName">网站名称</label>
										<input
											id="siteName"
											type="text"
											bind:value={config.general.siteName}
											placeholder="请输入网站名称"
										/>
									</div>

									<div class="form-group">
										<label for="adminEmail">管理员邮箱</label>
										<input
											id="adminEmail"
											type="email"
											bind:value={config.general.adminEmail}
											placeholder="admin@example.com"
										/>
									</div>

									<div class="form-group full-width">
										<label for="siteDescription">网站描述</label>
										<textarea
											id="siteDescription"
											bind:value={config.general.siteDescription}
											placeholder="请输入网站描述"
											rows="3"
										></textarea>
									</div>

									<div class="form-group">
										<label for="timezone">时区</label>
										<select bind:value={config.general.timezone}>
											<option value="Asia/Shanghai">Asia/Shanghai</option>
											<option value="UTC">UTC</option>
											<option value="America/New_York">America/New_York</option>
											<option value="Europe/London">Europe/London</option>
										</select>
									</div>

									<div class="form-group">
										<label for="language">语言</label>
										<select bind:value={config.general.language}>
											<option value="zh-CN">简体中文</option>
											<option value="en-US">English</option>
											<option value="ja-JP">日本語</option>
										</select>
									</div>

									<div class="form-group">
										<label class="checkbox-label">
											<input
												type="checkbox"
												bind:checked={config.general.debugMode}
											/>
											<span>调试模式</span>
										</label>
									</div>

									<div class="form-group">
										<label class="checkbox-label">
											<input
												type="checkbox"
												bind:checked={config.general.maintenanceMode}
											/>
											<span>维护模式</span>
										</label>
									</div>
								</div>
							{:else if category.id === 'server'}
								<div class="form-grid">
									<div class="form-group">
										<label for="host">监听地址</label>
										<input
											id="host"
											type="text"
											bind:value={config.server.host}
											placeholder="0.0.0.0"
										/>
									</div>

									<div class="form-group">
										<label for="port">端口</label>
										<input
											id="port"
											type="number"
											bind:value={config.server.port}
											min="1"
											max="65535"
										/>
									</div>

									<div class="form-group">
										<label for="readTimeout">读取超时（秒）</label>
										<input
											id="readTimeout"
											type="number"
											bind:value={config.server.readTimeout}
											min="1"
										/>
									</div>

									<div class="form-group">
										<label for="writeTimeout">写入超时（秒）</label>
										<input
											id="writeTimeout"
											type="number"
											bind:value={config.server.writeTimeout}
											min="1"
										/>
									</div>

									<div class="form-group">
										<label for="maxConnections">最大连接数</label>
										<input
											id="maxConnections"
											type="number"
											bind:value={config.server.maxConnections}
											min="1"
										/>
									</div>

									<div class="form-group">
										<label class="checkbox-label">
											<input
												type="checkbox"
												bind:checked={config.server.enableHttps}
											/>
											<span>启用HTTPS</span>
										</label>
									</div>

									<div class="form-group">
										<label for="sslCertPath">SSL证书路径</label>
										<input
											id="sslCertPath"
											type="text"
											bind:value={config.server.sslCertPath}
											placeholder="/path/to/cert.pem"
										/>
									</div>

									<div class="form-group">
										<label for="sslKeyPath">SSL私钥路径</label>
										<input
											id="sslKeyPath"
											type="text"
											bind:value={config.server.sslKeyPath}
											placeholder="/path/to/key.pem"
										/>
									</div>
								</div>
							{:else if category.id === 'database'}
								<div class="form-grid">
									<div class="form-group">
										<label for="dbDriver">数据库驱动</label>
										<select bind:value={config.database.driver}>
											<option value="sqlite">SQLite</option>
											<option value="postgres">PostgreSQL</option>
											<option value="mysql">MySQL</option>
										</select>
									</div>

									<div class="form-group">
										<label for="dbHost">主机地址</label>
										<input
											id="dbHost"
											type="text"
											bind:value={config.database.host}
											placeholder="localhost"
										/>
									</div>

									<div class="form-group">
										<label for="dbPort">端口</label>
										<input
											id="dbPort"
											type="number"
											bind:value={config.database.port}
											placeholder="5432"
										/>
									</div>

									<div class="form-group">
										<label for="dbName">数据库名</label>
										<input
											id="dbName"
											type="text"
											bind:value={config.database.database}
											placeholder="metabase"
										/>
									</div>

									<div class="form-group">
										<label for="dbUsername">用户名</label>
										<input
											id="dbUsername"
											type="text"
											bind:value={config.database.username}
											placeholder="admin"
										/>
									</div>

									<div class="form-group">
										<label for="dbPassword">密码</label>
										{#if showPasswords.dbPassword}
											<div class="password-input">
												<input
													id="dbPassword"
													type="text"
													bind:value={config.database.password}
													placeholder="请输入密码"
												/>
												<button
													type="button"
													class="password-toggle"
													on:click={() => togglePassword('dbPassword')}
												>
													<EyeOff size={16} />
												</button>
											</div>
										{:else}
											<div class="password-input">
												<input
													id="dbPassword"
													type="password"
													bind:value={config.database.password}
													placeholder="请输入密码"
												/>
												<button
													type="button"
													class="password-toggle"
													on:click={() => togglePassword('dbPassword')}
												>
													<Eye size={16} />
												</button>
											</div>
										{/if}
									</div>

									<div class="form-group">
										<label for="maxConnections">最大连接数</label>
										<input
											id="maxConnections"
											type="number"
											bind:value={config.database.maxConnections}
											min="1"
										/>
									</div>

									<div class="form-group">
										<label for="sslMode">SSL模式</label>
										<select bind:value={config.database.sslMode}>
											<option value="disable">禁用</option>
											<option value="require">必需</option>
											<option value="verify-ca">验证CA</option>
											<option value="verify-full">完全验证</option>
										</select>
									</div>
								</div>
							{:else if category.id === 'security'}
								<div class="form-grid">
									<div class="form-group full-width">
										<label for="jwtSecret">JWT密钥</label>
										{#if showPasswords.jwtSecret}
											<div class="password-input">
												<input
													id="jwtSecret"
													type="text"
													bind:value={config.security.jwtSecret}
													placeholder="请输入JWT密钥"
												/>
												<button
													type="button"
													class="password-toggle"
													on:click={() => togglePassword('jwtSecret')}
												>
													<EyeOff size={16} />
												</button>
											</div>
										{:else}
											<div class="password-input">
												<input
													id="jwtSecret"
													type="password"
													bind:value={config.security.jwtSecret}
													placeholder="请输入JWT密钥"
												/>
												<button
													type="button"
													class="password-toggle"
													on:click={() => togglePassword('jwtSecret')}
												>
													<Eye size={16} />
												</button>
											</div>
										{/if}
									</div>

									<div class="form-group">
										<label for="sessionTimeout">会话超时（秒）</label>
										<input
											id="sessionTimeout"
											type="number"
											bind:value={config.security.sessionTimeout}
											min="60"
										/>
									</div>

									<div class="form-group">
										<label for="maxLoginAttempts">最大登录尝试次数</label>
										<input
											id="maxLoginAttempts"
											type="number"
											bind:value={config.security.maxLoginAttempts}
											min="1"
										/>
									</div>

									<div class="form-group">
										<label for="lockoutDuration">锁定时长（秒）</label>
										<input
											id="lockoutDuration"
											type="number"
											bind:value={config.security.lockoutDuration}
											min="60"
										/>
									</div>

									<div class="form-group">
										<label class="checkbox-label">
											<input
												type="checkbox"
												bind:checked={config.security.enableTwoFactor}
											/>
											<span>启用双因子认证</span>
										</label>
									</div>

									<div class="form-group">
										<label>密码策略</label>
										<div class="password-policy">
											<label class="checkbox-label">
												<input
													type="checkbox"
													bind:checked={config.security.passwordPolicy.requireUppercase}
												/>
												<span>要求大写字母</span>
											</label>
											<label class="checkbox-label">
												<input
													type="checkbox"
													bind:checked={config.security.passwordPolicy.requireLowercase}
												/>
												<span>要求小写字母</span>
											</label>
											<label class="checkbox-label">
												<input
													type="checkbox"
													bind:checked={config.security.passwordPolicy.requireNumbers}
												/>
												<span>要求数字</span>
											</label>
											<label class="checkbox-label">
												<input
													type="checkbox"
													bind:checked={config.security.passwordPolicy.requireSymbols}
												/>
												<span>要求特殊字符</span>
											</label>
										</div>
									</div>
								</div>
							{:else}
								<div class="form-grid">
									<div class="form-group">
										<p class="config-notice">
											{category.name} 配置项正在开发中...
										</p>
									</div>
								</div>
							{/if}
						</div>
					</div>
				{/if}
			{/each}
		</div>
	</div>
</div>

<style>
	.config-page {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
	}

	.header-left h1 {
		margin: 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.page-description {
		margin: 0.5rem 0 0 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.header-right {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.success-message {
		background: #10b981;
		color: white;
		padding: 1rem 1.5rem;
		border-radius: 6px;
		text-align: center;
		font-weight: 500;
		animation: slideDown 0.3s ease;
	}

	@keyframes slideDown {
		from {
			opacity: 0;
			transform: translateY(-10px);
		}
		to {
			opacity: 1;
			transform: translateY(0);
		}
	}

	.config-content {
		display: grid;
		grid-template-columns: 280px 1fr;
		gap: 1.5rem;
		flex: 1;
	}

	.category-sidebar {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1rem;
		height: fit-content;
		position: sticky;
		top: 1rem;
	}

	.category-list {
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.category-item {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem;
		border: none;
		background: none;
		border-radius: 6px;
		text-align: left;
		width: 100%;
		cursor: pointer;
		transition: all 0.2s ease;
		color: #374151;
	}

	.category-item:hover {
		background: #f3f4f6;
	}

	.category-item.active {
		background: #eff6ff;
		color: #2563eb;
	}

	.category-info {
		flex: 1;
	}

	.category-name {
		font-weight: 600;
		font-size: 0.875rem;
		line-height: 1.25;
	}

	.category-desc {
		font-size: 0.75rem;
		color: #6b7280;
		line-height: 1.4;
		margin-top: 0.125rem;
	}

	.category-item.active .category-desc {
		color: #3b82f6;
	}

	.config-form {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		overflow: hidden;
	}

	.form-section {
		padding: 0;
	}

	.section-header {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
		background: #f9fafb;
	}

	.section-header h2 {
		margin: 0;
		font-size: 1.25rem;
		font-weight: 600;
		color: #111827;
	}

	.section-header p {
		margin: 0.25rem 0 0 0;
		font-size: 0.875rem;
		color: #6b7280;
	}

	.form-content {
		padding: 1.5rem;
	}

	.form-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
		gap: 1.5rem;
	}

	.form-group {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.form-group.full-width {
		grid-column: 1 / -1;
	}

	.form-group label {
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
	}

	.form-group input,
	.form-group select,
	.form-group textarea {
		padding: 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s ease, box-shadow 0.2s ease;
		background: white;
	}

	.form-group input:focus,
	.form-group select:focus,
	.form-group textarea:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.form-group textarea {
		resize: vertical;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: #374151;
		cursor: pointer;
	}

	.checkbox-label input[type="checkbox"] {
		width: 16px;
		height: 16px;
	}

	.password-input {
		position: relative;
		display: flex;
		align-items: center;
	}

	.password-input input {
		flex: 1;
		padding-right: 2.5rem;
	}

	.password-toggle {
		position: absolute;
		right: 0.75rem;
		background: none;
		border: none;
		color: #6b7280;
		cursor: pointer;
		padding: 0.25rem;
		border-radius: 4px;
		transition: color 0.2s ease;
	}

	.password-toggle:hover {
		color: #374151;
	}

	.password-policy {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		padding: 1rem;
		background: #f9fafb;
		border-radius: 6px;
		border: 1px solid #e5e7eb;
	}

	.config-notice {
		padding: 2rem;
		text-align: center;
		color: #6b7280;
		background: #f9fafb;
		border-radius: 6px;
		border: 1px dashed #d1d5db;
		margin: 0;
	}

	.btn {
		display: inline-flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		border: 1px solid transparent;
		border-radius: 6px;
		font-size: 0.875rem;
		font-weight: 500;
		text-decoration: none;
		cursor: pointer;
		transition: all 0.2s ease;
		background: none;
	}

	.btn-primary {
		background-color: #3b82f6;
		color: white;
	}

	.btn-primary:hover:not(:disabled) {
		background-color: #2563eb;
	}

	.btn-primary:disabled {
		opacity: 0.5;
		cursor: not-allowed;
	}

	.btn-secondary {
		background-color: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-secondary:hover {
		background-color: #f9fafb;
		border-color: #9ca3af;
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.header-right {
			width: 100%;
			justify-content: flex-start;
			flex-wrap: wrap;
		}

		.config-content {
			grid-template-columns: 1fr;
		}

		.category-sidebar {
			position: static;
		}

		.category-list {
			flex-direction: row;
			overflow-x: auto;
			padding-bottom: 0.5rem;
		}

		.category-item {
			min-width: 150px;
		}

		.category-info {
			display: none;
		}

		.form-grid {
			grid-template-columns: 1fr;
		}

		.section-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 0.5rem;
		}
	}
</style>