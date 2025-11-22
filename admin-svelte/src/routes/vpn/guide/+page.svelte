<script>
	import { Lock, Globe, Shield, Download, Copy, Terminal, Smartphone, Monitor, ChevronDown, ChevronRight } from 'lucide-svelte';

	// 状态管理
	let activeSection = 'overview';
	let selectedClient = null;
	let showConfigSteps = false;
	let showAdvancedConfig = false;
	let clientConfigs = [
		{
			id: 'example1',
			name: 'Windows 客户端',
			platform: 'windows',
			downloadUrl: 'https://github.com/p4gefau1t/trojan-go/releases',
			configTemplate: {
				"run_type": "client",
				"local_addr": "127.0.0.1",
				"local_port": 1080,
				"remote_addr": "your-server.com",
				"remote_port": 8443,
				"password": ["your-password-here"],
				"log_level": "info",
				"ssl": {
					"verify": true,
					"verify_hostname": true,
					"cert": "",
					"key": "",
					"key_password": "",
					"cipher": "",
					"curves": "",
					"prefer_server_cipher": false,
					"alpn": ["http/1.1"],
					"alpn_port_override": 0,
					"reuse_session": true,
					"session_ticket": false,
					"plain_http_response": "",
					"curves": "",
					"cipher": "",
					"cipher_tls13": "",
					"fingerprint": "chrome"
				}
			}
		},
		{
			id: 'example2',
			name: 'macOS 客户端',
			platform: 'macos',
			downloadUrl: 'https://github.com/p4gefau1t/trojan-go/releases',
			configTemplate: {
				"run_type": "client",
				"local_addr": "127.0.0.1",
				"local_port": 1080,
				"remote_addr": "your-server.com",
				"remote_port": 8443,
				"password": ["your-password-here"],
				"log_level": "info",
				"ssl": {
					"verify": true,
					"verify_hostname": true,
					"fingerprint": "chrome",
					"alpn": ["http/1.1"]
				},
				"mux": {
					"enabled": true,
					"concurrency": -1,
					"idle_timeout": 60
				}
			}
		},
		{
			id: 'example3',
			name: 'Android 客户端',
			platform: 'android',
			downloadUrl: 'https://github.com/p4gefau1t/trojan-go/releases',
			configTemplate: {
				"run_type": "client",
				"local_addr": "127.0.0.1",
				"local_port": 1080,
				"remote_addr": "your-server.com",
				"remote_port": 8443,
				"password": ["your-password-here"],
				"log_level": "info",
				"ssl": {
					"verify": true,
					"fingerprint": "chrome"
				}
			}
		},
		{
			id: 'example4',
			name: 'iOS 客户端',
			platform: 'ios',
			downloadUrl: 'https://apps.apple.com/us/app/shadowrocket/id932747118',
			configTemplate: {
				"servers": [
					{
						"server": "your-server.com",
						"server_port": 8443,
						"password": "your-password-here",
						"method": "trojan",
						"remarks": "MetaBase Trojan",
						"ssr_protocol": "",
						"ssr_obfs": "",
						"obfs_param": "",
						"protocol_param": "",
						"speed_limit_per_con": 0,
						"speed_limit_per_user": 0
					}
				]
			}
		}
	];

	// 切换章节
	function toggleSection(section) {
		activeSection = activeSection === section ? '' : section;
	}

	// 复制配置到剪贴板
	async function copyToClipboard(text) {
		try {
			await navigator.clipboard.writeText(text);
			// 可以添加一个提示消息
		} catch (err) {
			console.error('Failed to copy:', err);
		}
	}

	// 生成配置文件
	function generateConfig(client) {
		const config = { ...client.configTemplate };
		// 这里可以替换为实际的配置值
		return JSON.stringify(config, null, 2);
	}

	// 格式化配置步骤
	function getConfigSteps(platform) {
		const steps = {
			windows: [
				'下载 Trojan-Go Windows 版本',
				'解压下载的 ZIP 文件',
				'创建配置文件 config.json',
				'编辑配置文件，填入服务器信息',
				'运行 trojan.exe -config config.json'
			],
			macos: [
				'下载 Trojan-Go macOS 版本',
				'解压下载的 ZIP 文件',
				'创建配置文件 config.json',
				'编辑配置文件，填入服务器信息',
				'打开终端，运行 ./trojan-go -config config.json'
			],
			android: [
				'下载 v2rayNG 或 Clash for Android',
				'在 Trojan 管理页面获取配置信息',
				'在应用中添加新的代理配置',
				'选择 Trojan 协议并填入服务器信息',
				'连接代理并测试'
			],
			ios: [
				'下载 Shadowrocket 或 Quantumult X',
				'在 Trojan 管理页面获取配置信息',
				'通过扫描二维码或手动添加配置',
				'启用代理',
				'连接并测试'
			]
		};
		return steps[platform] || [];
	}

	// 获取平台图标
	function getPlatformIcon(platform) {
		const icons = {
			windows: Monitor,
			macos: Monitor,
			android: Smartphone,
			ios: Smartphone
		};
		return icons[platform] || Globe;
	}
</script>

<svelte:head>
	<Title>Trojan VPN 使用指南 - MetaBase</Title>
</svelte:head>

<div class="vpn-guide">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-content">
			<div class="header-left">
				<Lock size={32} class="header-icon" />
				<div>
					<h1>Trojan VPN 使用指南</h1>
					<p>详细的客户端配置和使用教程</p>
				</div>
			</div>
		</div>
	</div>

	<!-- 快速开始 -->
	<div class="section-card">
		<div class="section-header" on:click={() => toggleSection('quickstart')}>
			<div class="section-title">
				<Shield size={24} />
				<h3>快速开始</h3>
			</div>
			{#if activeSection === 'quickstart'}
				<ChevronDown size={20} />
			{:else}
				<ChevronRight size={20} />
			{/if}
		</div>

		{#if activeSection === 'quickstart'}
			<div class="section-content">
				<div class="quick-steps">
					<div class="step">
						<div class="step-number">1</div>
						<div class="step-content">
							<h4>获取配置信息</h4>
							<p>在 Trojan VPN 管理页面添加客户端，获取服务器地址、端口和密码</p>
						</div>
					</div>
					<div class="step">
						<div class="step-number">2</div>
						<div class="step-content">
							<h4>下载客户端</h4>
							<p>根据你的设备选择合适的客户端软件</p>
						</div>
					</div>
					<div class="step">
						<div class="step-number">3</div>
						<div class="step-content">
							<h4>配置客户端</h4>
							<p>在客户端中填入服务器信息并连接</p>
						</div>
					</div>
					<div class="step">
						<div class="step-number">4</div>
						<div class="step-content">
							<h4>测试连接</h4>
							<p>验证连接是否成功，检查IP地址和速度</p>
						</div>
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- 客户端配置 -->
	<div class="section-card">
		<div class="section-header" on:click={() => toggleSection('clients')}>
			<div class="section-title">
				<Monitor size={24} />
				<h3>客户端配置</h3>
			</div>
			{#if activeSection === 'clients'}
				<ChevronDown size={20} />
			{:else}
				<ChevronRight size={20} />
			{/if}
		</div>

		{#if activeSection === 'clients'}
			<div class="section-content">
				<div class="clients-grid">
					{#each clientConfigs as client}
						<div class="client-card">
							<div class="client-header">
								<div class="client-icon">
									<svelte:component this={getPlatformIcon(client.platform)} size={32} />
								</div>
								<div class="client-info">
									<h4>{client.name}</h4>
									<div class="client-platform">{client.platform.toUpperCase()}</div>
								</div>
								<a href={client.downloadUrl} target="_blank" class="download-btn">
									<Download size={16} />
									下载
								</a>
							</div>

							<div class="client-config">
								<div class="config-header">
									<span>配置示例</span>
									<button
										class="copy-btn"
										on:click={() => copyToClipboard(generateConfig(client))}
										title="复制配置"
									>
										<Copy size={16} />
									</button>
								</div>
								<pre class="config-code">{generateConfig(client)}</pre>
							</div>

							<div class="client-steps">
								<h5>配置步骤</h5>
								<ol>
									{#each getConfigSteps(client.platform) as step}
										<li>{step}</li>
									{/each}
								</ol>
							</div>
						</div>
					{/each}
				</div>
			</div>
		{/if}
	</div>

	<!-- 配置模板 -->
	<div class="section-card">
		<div class="section-header" on:click={() => toggleSection('templates')}>
			<div class="section-title">
				<Terminal size={24} />
				<h3>配置模板</h3>
			</div>
			{#if activeSection === 'templates'}
				<ChevronDown size={20} />
			{:else}
				<ChevronRight size={20} />
			{/if}
		</div>

		{#if activeSection === 'templates'}
			<div class="section-content">
				<div class="templates-container">
					<!-- 基础配置模板 -->
					<div class="template-section">
						<h4>基础配置模板</h4>
						<div class="template-code">
							<div class="code-header">
								<span>config.json</span>
								<button
									class="copy-btn"
									on:click={() => copyToClipboard(`{
  "run_type": "client",
  "local_addr": "127.0.0.1",
  "local_port": 1080,
  "remote_addr": "your-server.com",
  "remote_port": 8443,
  "password": ["your-password-here"],
  "log_level": "info"
}`)}
								>
									<Copy size={16} />
								</button>
							</div>
							<pre>{
  "run_type": "client",
  "local_addr": "127.0.0.1",
  "local_port": 1080,
  "remote_addr": "your-server.com",
  "remote_port": 8443,
  "password": ["your-password-here"],
  "log_level": "info"
}</pre>
						</div>
					</div>

					<!-- 高级配置模板 -->
					<div class="template-section">
						<div class="template-toggle" on:click={() => showAdvancedConfig = !showAdvancedConfig}>
							<h4>高级配置模板</h4>
							{#if showAdvancedConfig}
								<ChevronDown size={16} />
							{:else}
								<ChevronRight size={16} />
							{/if}
						</div>

						{#if showAdvancedConfig}
							<div class="template-code">
								<div class="code-header">
									<span>advanced-config.json</span>
									<button
										class="copy-btn"
										on:click={() => copyToClipboard(`{
  "run_type": "client",
  "local_addr": "127.0.0.1",
  "local_port": 1080,
  "remote_addr": "your-server.com",
  "remote_port": 8443,
  "password": ["your-password-here"],
  "log_level": "info",
  "ssl": {
    "verify": true,
    "verify_hostname": true,
    "cert": "",
    "key": "",
    "key_password": "",
    "cipher": "",
    "curves": "",
    "prefer_server_cipher": false,
    "alpn": ["http/1.1"],
    "alpn_port_override": 0,
    "reuse_session": true,
    "session_ticket": false,
    "plain_http_response": "",
	"fingerprint": "chrome",
	"cipher_tls13": "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_256_GCM_SHA384"
  },
  "mux": {
    "enabled": true,
    "concurrency": -1,
    "idle_timeout": 60
  },
  "router": {
    "enabled": false
  },
  "websocket": {
    "enabled": false,
    "path": "/ws",
    "host": "your-server.com"
  }
}`)}
									>
										<Copy size={16} />
									</button>
								</div>
								<pre>{
  "run_type": "client",
  "local_addr": "127.0.0.1",
  "local_port": 1080,
  "remote_addr": "your-server.com",
  "remote_port": 8443,
  "password": ["your-password-here"],
  "log_level": "info",
  "ssl": {
    "verify": true,
    "verify_hostname": true,
    "cert": "",
    "key": "",
    "key_password": "",
    "cipher": "",
    "curves": "",
    "prefer_server_cipher": false,
    "alpn": ["http/1.1"],
    "alpn_port_override": 0,
    "reuse_session": true,
    "session_ticket": false,
    "plain_http_response": "",
	"fingerprint": "chrome",
	"cipher_tls13": "TLS_AES_128_GCM_SHA256:TLS_CHACHA20_POLY1305_SHA256:TLS_AES_256_GCM_SHA384"
  },
  "mux": {
    "enabled": true,
    "concurrency": -1,
    "idle_timeout": 60
  },
  "router": {
    "enabled": false
  },
  "websocket": {
    "enabled": false,
    "path": "/ws",
    "host": "your-server.com"
  }
}</pre>
							</div>
						{/if}
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- 常见问题 -->
	<div class="section-card">
		<div class="section-header" on:click={() => toggleSection('faq')}>
			<div class="section-title">
				<Globe size={24} />
				<h3>常见问题</h3>
			</div>
			{#if activeSection === 'faq'}
				<ChevronDown size={20} />
			{:else}
				<ChevronRight size={20} />
			{/if}
		</div>

		{#if activeSection === 'faq'}
			<div class="section-content">
				<div class="faq-container">
					<div class="faq-item">
						<h5>如何验证连接是否成功？</h5>
						<p>连接成功后，可以通过访问 https://www.whatismyip.com 检查你的IP地址是否显示为服务器IP。</p>
					</div>

					<div class="faq-item">
						<h5>连接速度慢怎么办？</h5>
						<p>尝试以下方法优化速度：1) 更换服务器节点；2) 检查本地网络；3) 使用CDN加速；4) 启用Mux多路复用。</p>
					</div>

					<div class="faq-item">
						<h5>证书验证失败怎么办？</h5>
						<p>检查服务器证书是否正确配置，或者临时禁用证书验证（仅用于测试环境）。</p>
					</div>

					<div class="faq-item">
						<h5>如何设置系统代理？</h5>
						<p>Windows：设置 → 网络和Internet → 代理；macOS：系统偏好设置 → 网络 → 高级 → 代理。</p>
					</div>

					<div class="faq-item">
						<h5>支持哪些平台？</h5>
						<p>Trojan支持Windows、macOS、Linux、Android、iOS等主流平台。</p>
					</div>
				</div>
			</div>
		{/if}
	</div>

	<!-- 安全提示 -->
	<div class="section-card security-card">
		<div class="section-header" on:click={() => toggleSection('security')}>
			<div class="section-title">
				<Shield size={24} />
				<h3>安全提示</h3>
			</div>
			{#if activeSection === 'security'}
				<ChevronDown size={20} />
			{:else}
				<ChevronRight size={20} />
			{/if}
		</div>

		{#if activeSection === 'security'}
			<div class="section-content">
				<div class="security-tips">
					<div class="tip-item warning">
						<h5>🔒 保护配置文件</h5>
						<p>配置文件包含敏感信息，请妥善保管，不要分享给他人。</p>
					</div>

					<div class="tip-item info">
						<h5>🌐 使用HTTPS网站</h5>
						<p>即使使用VPN，仍然建议访问HTTPS网站以确保端到端加密。</p>
					</div>

					<div class="tip-item warning">
						<h5>🔍 定期更新客户端</h5>
						<p>保持客户端软件为最新版本，以获得安全更新和性能改进。</p>
					</div>

					<div class="tip-item info">
						<h5>📊 监控流量使用</h5>
						<p>定期检查流量使用情况，避免超出限制。</p>
					</div>

					<div class="tip-item error">
						<h5>🚫 遵守当地法律</h5>
						<p>使用VPN时请遵守当地法律法规，不得用于非法用途。</p>
					</div>
				</div>
			</div>
		{/if}
	</div>
</div>

<style>
	.vpn-guide {
		padding: 1rem;
		max-width: 1200px;
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

	.section-card {
		background: white;
		border-radius: 12px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		border: 1px solid #e5e7eb;
		margin-bottom: 1.5rem;
		overflow: hidden;
	}

	.security-card {
		border-left: 4px solid #ef4444;
	}

	.section-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 1.5rem;
		cursor: pointer;
		transition: background-color 0.2s;
		border-bottom: 1px solid #f3f4f6;
	}

	.section-header:hover {
		background-color: #f9fafb;
	}

	.section-title {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		color: #374151;
	}

	.section-title h3 {
		margin: 0;
		font-size: 1.125rem;
		font-weight: 600;
	}

	.section-content {
		padding: 1.5rem;
		border-top: 1px solid #e5e7eb;
	}

	/* 快速步骤 */
	.quick-steps {
		display: grid;
		gap: 1.5rem;
	}

	.step {
		display: flex;
		align-items: flex-start;
		gap: 1rem;
	}

	.step-number {
		display: flex;
		align-items: center;
		justify-content: center;
		width: 32px;
		height: 32px;
		background: #3b82f6;
		color: white;
		border-radius: 50%;
		font-weight: 600;
		flex-shrink: 0;
	}

	.step-content h4 {
		margin: 0 0 0.25rem 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.step-content p {
		margin: 0;
		color: #6b7280;
		font-size: 0.875rem;
		line-height: 1.5;
	}

	/* 客户端网格 */
	.clients-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}

	.client-card {
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		overflow: hidden;
	}

	.client-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem;
		background: #f9fafb;
		border-bottom: 1px solid #e5e7eb;
	}

	.client-icon {
		color: #3b82f6;
	}

	.client-info h4 {
		margin: 0 0 0.25rem 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.client-platform {
		font-size: 0.75rem;
		color: #6b7280;
		background: #f3f4f6;
		padding: 0.125rem 0.5rem;
		border-radius: 12px;
		display: inline-block;
	}

	.download-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		padding: 0.5rem 1rem;
		background: #3b82f6;
		color: white;
		text-decoration: none;
		border-radius: 6px;
		font-size: 0.875rem;
		font-weight: 500;
		transition: background-color 0.2s;
	}

	.download-btn:hover {
		background: #2563eb;
	}

	.client-config {
		padding: 1rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.config-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		margin-bottom: 0.5rem;
		font-weight: 600;
		color: #374151;
		font-size: 0.875rem;
	}

	.copy-btn {
		background: none;
		border: none;
		cursor: pointer;
		color: #6b7280;
		padding: 0.25rem;
		border-radius: 4px;
		transition: all 0.2s;
	}

	.copy-btn:hover {
		background: #f3f4f6;
		color: #374151;
	}

	.config-code {
		background: #1f2937;
		color: #e5e7eb;
		padding: 1rem;
		border-radius: 6px;
		font-size: 0.75rem;
		line-height: 1.5;
		overflow-x: auto;
		margin: 0;
	}

	.client-steps {
		padding: 1rem;
	}

	.client-steps h5 {
		margin: 0 0 0.75rem 0;
		font-size: 0.875rem;
		font-weight: 600;
		color: #111827;
	}

	.client-steps ol {
		margin: 0;
		padding-left: 1.25rem;
		color: #6b7280;
		font-size: 0.875rem;
		line-height: 1.6;
	}

	.client-steps li {
		margin-bottom: 0.5rem;
	}

	/* 模板区域 */
	.templates-container {
		display: grid;
		gap: 2rem;
	}

	.template-section h4 {
		margin: 0 0 1rem 0;
		font-size: 1.125rem;
		font-weight: 600;
		color: #111827;
	}

	.template-toggle {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		cursor: pointer;
		margin-bottom: 1rem;
		padding: 0.75rem;
		background: #f9fafb;
		border-radius: 6px;
		transition: background-color 0.2s;
	}

	.template-toggle:hover {
		background: #f3f4f6;
	}

	.template-toggle h4 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.template-code {
		background: #1f2937;
		border-radius: 8px;
		overflow: hidden;
	}

	.code-header {
		display: flex;
		justify-content: space-between;
		align-items: center;
		padding: 0.75rem 1rem;
		background: #374151;
		color: #e5e7eb;
		font-size: 0.875rem;
		font-weight: 500;
	}

	.template-code pre {
		padding: 1rem;
		color: #e5e7eb;
		font-size: 0.75rem;
		line-height: 1.5;
		overflow-x: auto;
		margin: 0;
	}

	/* FAQ */
	.faq-container {
		display: grid;
		gap: 1.5rem;
	}

	.faq-item h5 {
		margin: 0 0 0.5rem 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.faq-item p {
		margin: 0;
		color: #6b7280;
		line-height: 1.6;
	}

	/* 安全提示 */
	.security-tips {
		display: grid;
		gap: 1rem;
	}

	.tip-item {
		padding: 1rem;
		border-radius: 8px;
		border-left: 4px solid;
	}

	.tip-item.warning {
		background: #fef3c7;
		border-color: #f59e0b;
	}

	.tip-item.info {
		background: #dbeafe;
		border-color: #3b82f6;
	}

	.tip-item.error {
		background: #fef2f2;
		border-color: #ef4444;
	}

	.tip-item h5 {
		margin: 0 0 0.5rem 0;
		font-size: 0.875rem;
		font-weight: 600;
		color: #111827;
	}

	.tip-item p {
		margin: 0;
		color: #6b7280;
		font-size: 0.875rem;
		line-height: 1.5;
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.vpn-guide {
			padding: 0.5rem;
		}

		.header-content {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.clients-grid {
			grid-template-columns: 1fr;
		}

		.client-header {
			flex-direction: column;
			gap: 1rem;
			align-items: flex-start;
		}

		.step {
			align-items: flex-start;
		}

		.step-number {
			margin-top: 0.125rem;
		}

		.config-code,
		.template-code pre {
			font-size: 0.625rem;
		}

		.faq-container,
		.security-tips {
			gap: 1rem;
		}
	}
</style>