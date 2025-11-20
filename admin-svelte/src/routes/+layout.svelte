<script>
	import { page } from '$app/stores';
	import { onMount } from 'svelte';
	import { goto } from '$app/navigation';
	import { Menu, X, Activity, FileText, BarChart3, Settings, Database, Users, Folder, Search, Shield, HardDrive, Globe } from 'lucide-svelte';

	let sidebarOpen = false;
	let currentPage = '';

	$: breadcrumb = getCurrentBreadcrumb();

	onMount(() => {
		currentPage = $page.url.pathname;
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
		sidebarOpen = false;
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

<div class="admin-container">
	<!-- 侧边栏 -->
	<aside class="sidebar" class:sidebar-open={sidebarOpen}>
		<div class="sidebar-header">
			<div class="logo">
				<span class="logo-icon">●</span>
				<span class="logo-text">MetaBase</span>
			</div>
			<button class="sidebar-toggle" on:click={toggleSidebar}>
				<X />
			</button>
		</div>

		<nav class="sidebar-nav">
			{#each navigation as item}
				<a
					href={item.path}
					class="nav-link"
					class:active={$page.url.pathname === item.path}
					on:click|preventDefault={() => navigate(item.path)}
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

	<!-- 主内容区 -->
	<main class="main-content">
		<!-- 顶部栏 -->
		<header class="topbar">
			<div class="topbar-left">
				<button class="mobile-sidebar-toggle" on:click={toggleSidebar}>
					<Menu />
				</button>
				<nav class="breadcrumb">
					{#each breadcrumb as crumb}
						<span class="breadcrumb-item">{crumb}</span>
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

		<!-- 页面内容 -->
		<div class="content-area">
			<slot />
		</div>
	</main>

	<!-- 移动端遮罩 -->
	{#if sidebarOpen}
		<div class="mobile-overlay" on:click={toggleSidebar} />
	{/if}
</div>

<style>
	.admin-container {
		display: flex;
		height: 100vh;
		overflow: hidden;
	}

	/* 侧边栏样式 */
	.sidebar {
		width: 260px;
		background: white;
		border-right: 1px solid #e5e7eb;
		display: flex;
		flex-direction: column;
		position: fixed;
		top: 0;
		left: 0;
		bottom: 0;
		z-index: 100;
		transform: translateX(-100%);
		transition: transform 0.3s ease;
	}

	.sidebar.sidebar-open {
		transform: translateX(0);
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

	.sidebar-toggle,
	.mobile-sidebar-toggle {
		background: none;
		border: none;
		padding: 0.5rem;
		border-radius: 6px;
		cursor: pointer;
		color: #6b7280;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.sidebar-toggle:hover,
	.mobile-sidebar-toggle:hover {
		background-color: #f3f4f6;
		color: #374151;
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

	/* 主内容区 */
	.main-content {
		flex: 1;
		display: flex;
		flex-direction: column;
		margin-left: 0;
		transition: margin-left 0.3s ease;
	}

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
		z-index: 50;
	}

	.topbar-left {
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.mobile-sidebar-toggle {
		display: flex;
	}

	.breadcrumb {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.breadcrumb-item:not(:last-child)::after {
		content: '/';
		margin-left: 0.5rem;
		color: #9ca3af;
	}

	.topbar-right {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.action-btn {
		background: none;
		border: none;
		padding: 0.5rem;
		border-radius: 6px;
		cursor: pointer;
		color: #6b7280;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.action-btn:hover {
		background-color: #f3f4f6;
		color: #374151;
	}

	.user-btn {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		background: none;
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

	.content-area {
		flex: 1;
		padding: 1.5rem;
		overflow-y: auto;
		background: #f9fafb;
	}

	.mobile-overlay {
		position: fixed;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: rgba(0, 0, 0, 0.5);
		z-index: 99;
	}

	/* 响应式设计 */
	@media (min-width: 768px) {
		.sidebar {
			position: relative;
			transform: translateX(0);
		}

		.mobile-sidebar-toggle {
			display: none;
		}

		.mobile-overlay {
			display: none;
		}

		.main-content {
			margin-left: 260px;
		}

		.sidebar-toggle {
			display: none;
		}
	}

	@media (max-width: 767px) {
		.content-area {
			padding: 1rem;
		}
	}
</style>