<script>
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Menu, X, Activity, FileText, BarChart3, Settings, Database, Users, Folder, Search, Shield, HardDrive, Globe, Terminal } from 'lucide-svelte';

	let sidebarOpen = false;
	let currentPage = '';

	$: breadcrumb = getCurrentBreadcrumb();

	onMount(() => {
		currentPage = $page.url.pathname;
		// 确保页面加载时正确设置侧边栏状态
		const isMobile = window.innerWidth < 768;
		sidebarOpen = !isMobile;
	});

	function getCurrentBreadcrumb() {
		const path = $page.url.pathname;
		const navItem = navigation.find(item => item.path === path);
		if (navItem) {
			return [navItem.label];
		}
		return ['仪表盘'];
	}

	function toggleSidebar() {
		sidebarOpen = !sidebarOpen;
	}

	function navigate(path) {
		goto(path);
		// 在移动设备上导航后自动关闭侧边栏
		const isMobile = window.innerWidth < 768;
		if (isMobile) {
			sidebarOpen = false;
		}
	}

	const navigation = [
		{
			icon: Activity,
			label: '仪表盘',
			path: '/',
			description: '系统概览和关键指标'
		},
		{
			icon: Users,
			label: '用户管理',
			path: '/users',
			description: '用户账户和权限管理'
		},
		{
			icon: Globe,
			label: '租户管理',
			path: '/tenants',
			description: '多租户配置和管理'
		},
		{
			icon: BarChart3,
			label: '数据分析',
			path: '/analytics',
			description: '事件分析和实时统计'
		},
		{
			icon: Folder,
			label: '文件管理',
			path: '/files',
			description: '文件上传下载和存储管理'
		},
		{
			icon: Search,
			label: '搜索管理',
			path: '/search',
			description: '索引管理和搜索配置'
		},
		{
			icon: HardDrive,
			label: '存储监控',
			path: '/storage',
			description: '数据库性能和缓存统计'
		},
		{
			icon: Shield,
			label: '安全配置',
			path: '/security',
			description: '认证设置和访问控制'
		},
		{
			icon: Terminal,
			label: '系统日志',
			path: '/logs',
			description: '应用程序日志和错误追踪'
		},
		{
			icon: FileText,
			label: '请求日志',
			path: '/requests',
			description: 'HTTP请求记录和搜索'
		},
		{
			icon: Activity,
			label: '性能监控',
			path: '/performance',
			description: '系统性能和资源监控'
		},
		{
			icon: Settings,
			label: '系统设置',
			path: '/settings',
			description: '系统参数和环境配置'
		},
		{
			icon: Database,
			label: '数据表管理',
			path: '/tables',
			description: '数据表结构和管理'
		}
	];
</script>

<svelte:head>
	<style>
		:global(body) {
			margin: 0;
			font-family: 'Inter', system-ui, -apple-system, sans-serif;
			-webkit-font-smoothing: antialiased;
			-moz-osx-font-smoothing: grayscale;
		}
	</style>
</svelte:head>

<!-- 标准左右两栏布局 -->
<div class="app-container">
	<!-- 左侧边栏 -->
	<aside class="sidebar">
		<div class="sidebar-header">
			<div class="logo">
				<span class="logo-icon">●</span>
				<span class="logo-text">MetaBase</span>
			</div>
			<button class="mobile-sidebar-toggle-close" on:click={toggleSidebar} aria-label="关闭侧边栏">
				<X size={20} />
			</button>
		</div>

		<nav class="sidebar-nav">
			{#each navigation as item}
				<a
					href={item.path}
					class="nav-link"
					class:active={$page.url.pathname === item.path}
					on:click|preventDefault={() => navigate(item.path)}
					title={item.description}
				>
					<svelte:component this={item.icon} size={20} />
					<span>{item.label}</span>
				</a>
			{/each}
		</nav>

		<div class="sidebar-footer">
			<div class="user-info">
				<div class="user-avatar">A</div>
				<div class="user-details">
					<div class="user-name">管理员</div>
					<div class="user-role">超级管理员</div>
				</div>
			</div>
		</div>
	</aside>

	<!-- 右侧主内容区 -->
	<main class="main-content">
		<!-- 顶部导航栏 -->
		<header class="topbar">
			<div class="topbar-left">
				<button class="mobile-sidebar-toggle" on:click={toggleSidebar} aria-label="切换侧边栏">
					<Menu size={20} />
				</button>
				<nav class="breadcrumb">
					{#each breadcrumb as crumb, index}
						<span class="breadcrumb-item">{crumb}</span>
						{#if index < breadcrumb.length - 1}
							<span class="breadcrumb-separator">/</span>
						{/if}
					{/each}
				</nav>
			</div>

			<div class="topbar-right">
				<button class="action-btn" title="刷新">
					⟳
				</button>
				<div class="user-dropdown">
					<button class="user-btn">
						<div class="user-avatar small">A</div>
						<span>管理员</span>
						<span>▼</span>
					</button>
				</div>
			</div>
		</header>

		<!-- 主容器 - 占据剩余空间 -->
		<div class="main-container">
			<!-- 页面内容区域 - 居中显示并设置最大宽度 -->
			<div class="content-area">
				<slot />
			</div>
		</div>
	</main>

	<!-- 移动端侧边栏遮罩 -->
	{#if sidebarOpen && window.innerWidth < 768}
		<div class="mobile-overlay" on:click={toggleSidebar}></div>
	{/if}
</div>

<style>
	/* 全局重置和基础样式 */
	* {
		box-sizing: border-box;
	}

	/* 应用容器 - 作为整个页面的根容器 */
	.app-container {
		/* 不使用flex布局，让sidebar和main-content独立定位 */
		min-height: 100vh;
		background-color: #f9fafb;
	}

	/* 主布局包装器，确保正确的z-index层级 */
	.layout-wrapper {
		position: relative;
		min-height: 100vh;
	}

	/* 侧边栏样式 - 固定定位 */
	.sidebar {
		width: 260px;
		background: white;
		border-right: 1px solid #e5e7eb;
		display: flex;
		flex-direction: column;
		position: fixed; /* 固定定位 */
		top: 0;
		left: 0;
		height: 100vh;
		z-index: 20;
		transition: transform 0.3s ease;
		overflow-y: auto; /* 确保侧边栏内容过多时可滚动 */
	}

	.sidebar-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.logo {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		font-weight: 600;
		font-size: 1.125rem;
		color: #111827;
	}

	.logo-icon {
		width: 32px;
		height: 32px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-size: 18px;
	}

	.mobile-sidebar-toggle,
	.mobile-sidebar-toggle-close {
		background: transparent;
		border: none;
		padding: 0.5rem;
		border-radius: 6px;
		cursor: pointer;
		color: #6b7280;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s ease;
	}

	.mobile-sidebar-toggle:hover,
	.mobile-sidebar-toggle-close:hover {
		background-color: #f3f4f6;
		color: #374151;
	}

	.mobile-sidebar-toggle-close {
		display: none;
	}

	.sidebar-nav {
		flex: 1;
		padding: 1rem 0;
		overflow-y: auto;
	}

	.nav-link {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		padding: 0.75rem 1rem;
		color: #374151;
		text-decoration: none;
		transition: all 0.2s ease;
		border-left: 3px solid transparent;
		width: 100%;
	}

	.nav-link:hover {
		background-color: #f9fafb;
		color: #111827;
	}

	.nav-link.active {
		background-color: #eff6ff;
		color: #2563eb;
		border-left-color: #2563eb;
	}

	.sidebar-footer {
		padding: 1rem;
		border-top: 1px solid #e5e7eb;
		margin-top: auto;
	}

	.user-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.user-avatar {
		width: 40px;
		height: 40px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		font-weight: 600;
		font-size: 0.875rem;
	}

	.user-avatar.small {
		width: 32px;
		height: 32px;
		font-size: 0.75rem;
	}

	.user-details {
		flex: 1;
	}

	.user-name {
		font-weight: 600;
		color: #111827;
		font-size: 0.875rem;
	}

	.user-role {
		color: #6b7280;
		font-size: 0.75rem;
	}

	/* 主内容区样式 - 右侧区域 */
  	.main-content {
		/* 添加margin-left为侧边栏留出空间 */
		margin-left: 260px;
		min-height: 100vh;
		display: flex;
		flex-direction: column;
		overflow: visible;
		background-color: #f9fafb;
	}

	/* 主容器 - 占据剩余空间 */
	.main-container {
		flex: 1;
		display: flex;
		flex-direction: column;
		width: 100%;
		overflow: visible;
		height: auto;
		padding: 1.5rem;
	}

	/* 顶部导航栏 */
	.topbar {
		height: 64px;
		background: white;
		border-bottom: 1px solid #e5e7eb;
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0 1.5rem;
		position: sticky;
		top: 0;
		z-index: 10;
	}

	.topbar-left {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.mobile-sidebar-toggle {
		display: none;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.breadcrumb-item {
		white-space: nowrap;
	}

	.breadcrumb-separator {
		color: #9ca3af;
	}

	.topbar-right {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.action-btn {
		background: transparent;
		border: none;
		padding: 0.5rem;
		border-radius: 6px;
		cursor: pointer;
		color: #6b7280;
		display: flex;
		align-items: center;
		justify-content: center;
		transition: all 0.2s ease;
	}

	.action-btn:hover {
		background-color: #f3f4f6;
		color: #374151;
	}

	.user-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: transparent;
		border: 1px solid #e5e7eb;
		padding: 0.375rem 0.75rem;
		border-radius: 6px;
		cursor: pointer;
		transition: all 0.2s ease;
		color: #374151;
	}

	.user-btn:hover {
		background-color: #f9fafb;
		border-color: #d1d5db;
	}

	/* 内容区域 - 居中显示并设置最大宽度 */
  	.content-area {
		/* 基础样式 */
		width: 100%;
		max-width: 1600px;
		margin: 0 auto;
		background-color: white;
		border-radius: 8px;
		box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
		padding: 2rem;
		border: 1px solid #e5e7eb;
		box-sizing: border-box;
		
		/* 确保高度自动适应内容 */
		min-height: unset;
		height: auto;
		overflow: visible;
		position: static;
		float: none;
		clear: none;
	}

	/* 确保overflow-content元素不会导致父容器高度计算错误 */
	.overflow-content {
		min-height: auto;
		height: auto;
		box-sizing: border-box;
	}

	/* 移动端遮罩 */
	.mobile-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		z-index: 15;
	}

	/* 移动端设计 */
	@media (max-width: 767px) {
		/* 移动设备：侧边栏默认隐藏 */
		.sidebar {
			position: fixed;
			top: 0;
			left: 0;
			transform: translateX(-100%);
			z-index: 20;
		}

		/* 移动设备：侧边栏打开状态 */
		:global(.app-container) .sidebar {
			transform: translateX(0);
		}

		/* 移动设备：显示关闭按钮 */
		.mobile-sidebar-toggle-close {
			display: flex;
		}

		/* 移动设备：显示顶部栏切换按钮 */
		.mobile-sidebar-toggle {
			display: flex;
		}

		/* 移动设备：调整主容器内边距 */
			.main-container {
				padding: 1rem;
			}

			/* 移动设备：调整内容区域 */
			.content-area {
				padding: 1.5rem;
				max-width: 100%;
				box-shadow: none;
				border-radius: 0;
			}

		/* 移动设备：调整顶部栏 */
		.topbar {
			padding: 0 1rem;
		}
	}

	/* 大屏幕优化 */
	@media (min-width: 1200px) {
		.main-container {
			padding: 2rem;
		}
	}

	@media (min-width: 1600px) {
		.main-container {
			padding: 2rem 3rem;
		}

		.content-area {
			padding: 2rem 3rem;
		}
	}
</style>