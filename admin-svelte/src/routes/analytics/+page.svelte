<script>
	import { onMount } from 'svelte';
	import { TrendingUp, Users, Eye, MousePointer, Calendar, Download, BarChart3, PieChart, Activity } from 'lucide-svelte';

	// 时间范围选择
	let timeRange = '7d';
	let loading = false;

	// 分析数据
	let analytics = {
		overview: {
			totalVisits: 0,
			uniqueVisitors: 0,
			pageViews: 0,
			avgSessionDuration: 0,
			bounceRate: 0
		},
		trends: [],
		topPages: [],
		devices: [],
		browsers: [],
		referrers: []
	};

	// 生成模拟数据
	function generateMockData() {
		const now = new Date();

		// 生成趋势数据
		const trends = [];
		for (let i = 30; i >= 0; i--) {
			const date = new Date(now);
			date.setDate(date.getDate() - i);
			trends.push({
				date: date.toISOString().split('T')[0],
				visits: Math.floor(Math.random() * 1000) + 500,
				uniqueVisitors: Math.floor(Math.random() * 800) + 300,
				pageViews: Math.floor(Math.random() * 3000) + 1500
			});
		}

		// 生成热门页面
		const topPages = [
			{ path: '/', visits: 3420, percentage: 28.5 },
			{ path: '/admin', visits: 2156, percentage: 18.0 },
			{ path: '/docs', visits: 1876, percentage: 15.6 },
			{ path: '/api/users', visits: 1234, percentage: 10.3 },
			{ path: '/dashboard', visits: 987, percentage: 8.2 },
			{ path: '/api/products', visits: 876, percentage: 7.3 },
			{ path: '/help', visits: 654, percentage: 5.4 },
			{ path: '/about', visits: 543, percentage: 4.5 }
		];

		// 生成设备数据
		const devices = [
			{ type: 'Desktop', count: 7234, percentage: 60.3 },
			{ type: 'Mobile', count: 3567, percentage: 29.7 },
			{ type: 'Tablet', count: 1199, percentage: 10.0 }
		];

		// 生成浏览器数据
		const browsers = [
			{ name: 'Chrome', count: 6789, percentage: 56.6 },
			{ name: 'Safari', count: 2345, percentage: 19.5 },
			{ name: 'Firefox', count: 1567, percentage: 13.1 },
			{ name: 'Edge', count: 876, percentage: 7.3 },
			{ name: 'Other', count: 423, percentage: 3.5 }
		];

		// 生成来源数据
		const referrers = [
			{ source: 'Direct', count: 4567, percentage: 38.1 },
			{ source: 'Google', count: 3234, percentage: 26.9 },
			{ source: 'GitHub', count: 1876, percentage: 15.6 },
			{ source: 'Twitter', count: 1234, percentage: 10.3 },
			{ source: 'Other', count: 1089, percentage: 9.1 }
		];

		// 计算总览数据
		const totalVisits = trends.reduce((sum, day) => sum + day.visits, 0);
		const totalPageViews = trends.reduce((sum, day) => sum + day.pageViews, 0);
		const uniqueVisitors = Math.floor(totalVisits * 0.65);

		return {
			overview: {
				totalVisits,
				uniqueVisitors,
				pageViews: totalPageViews,
				avgSessionDuration: Math.floor(Math.random() * 300) + 120,
				bounceRate: (Math.random() * 30 + 20).toFixed(1)
			},
			trends,
			topPages,
			devices,
			browsers,
			referrers
		};
	}

	onMount(() => {
		loadAnalytics();
	});

	function loadAnalytics() {
		loading = true;
		setTimeout(() => {
			analytics = generateMockData();
			loading = false;
		}, 800);
	}

	function formatNumber(num) {
		if (num >= 1000000) return (num / 1000000).toFixed(1) + 'M';
		if (num >= 1000) return (num / 1000).toFixed(1) + 'K';
		return num.toString();
	}

	function formatDuration(seconds) {
		const mins = Math.floor(seconds / 60);
		const secs = seconds % 60;
		return `${mins}:${secs.toString().padStart(2, '0')}`;
	}

	function exportAnalytics() {
		const data = {
			overview: analytics.overview,
			topPages: analytics.topPages,
			devices: analytics.devices,
			browsers: analytics.browsers,
			referrers: analytics.referrers,
			exportDate: new Date().toISOString()
		};

		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `analytics_${timeRange}_${new Date().toISOString().split('T')[0]}.json`;
		a.click();
		URL.revokeObjectURL(url);
	}

	// 计算增长率
	function calculateGrowth(current, previous) {
		if (previous === 0) return 0;
		const growth = ((current - previous) / previous) * 100;
		return growth.toFixed(1);
	}
</script>

<div class="analytics-page">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-left">
			<h1>访问分析</h1>
			<p class="page-description">流量统计和用户行为分析</p>
		</div>
		<div class="header-right">
			<div class="time-range-selector">
				<select bind:value={timeRange}>
					<option value="24h">最近24小时</option>
					<option value="7d">最近7天</option>
					<option value="30d">最近30天</option>
					<option value="90d">最近90天</option>
				</select>
			</div>
			<button class="btn btn-primary" on:click={exportAnalytics}>
				<Download size={16} />
				导出报告
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="loading-spinner"></div>
			<p>加载分析数据中...</p>
		</div>
	{:else}
		<!-- 总览指标 -->
		<div class="overview-section">
			<div class="metric-card">
				<div class="metric-icon visits">
					<Eye size={24} />
				</div>
				<div class="metric-content">
					<div class="metric-value">{formatNumber(analytics.overview.totalVisits)}</div>
					<div class="metric-label">总访问次数</div>
					<div class="metric-change positive">
						<TrendingUp size={16} />
						<span>+12.5%</span>
						<span>较上期</span>
					</div>
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-icon users">
					<Users size={24} />
				</div>
				<div class="metric-content">
					<div class="metric-value">{formatNumber(analytics.overview.uniqueVisitors)}</div>
					<div class="metric-label">独立访客</div>
					<div class="metric-change positive">
						<TrendingUp size={16} />
						<span>+8.3%</span>
						<span>较上期</span>
					</div>
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-icon pages">
					<BarChart3 size={24} />
				</div>
				<div class="metric-content">
					<div class="metric-value">{formatNumber(analytics.overview.pageViews)}</div>
					<div class="metric-label">页面浏览量</div>
					<div class="metric-change positive">
						<TrendingUp size={16} />
						<span>+15.7%</span>
						<span>较上期</span>
					</div>
				</div>
			</div>

			<div class="metric-card">
				<div class="metric-icon duration">
					<Activity size={24} />
				</div>
				<div class="metric-content">
					<div class="metric-value">{formatDuration(analytics.overview.avgSessionDuration)}</div>
					<div class="metric-label">平均停留时间</div>
					<div class="metric-change negative">
						<TrendingUp size={16} />
						<span>-5.2%</span>
						<span>较上期</span>
					</div>
				</div>
			</div>
		</div>

		<!-- 图表和分析区域 -->
		<div class="charts-section">
			<!-- 访问趋势图 -->
			<div class="chart-card large">
				<div class="card-header">
					<h3>访问趋势</h3>
					<div class="chart-legend">
						<span class="legend-item">
							<span class="legend-color visits"></span>
							访问次数
						</span>
						<span class="legend-item">
							<span class="legend-color visitors"></span>
							独立访客
						</span>
					</div>
				</div>
				<div class="card-body">
					<div class="chart-placeholder trend-chart">
						<div class="chart-grid">
							{#each analytics.trends.slice(-7) as trend}
								<div class="chart-bar-container">
									<div class="chart-bars">
										<div class="chart-bar visits" style="height: {(trend.visits / 1500) * 100}%"></div>
										<div class="chart-bar visitors" style="height: {(trend.uniqueVisitors / 1200) * 100}%"></div>
									</div>
									<div class="chart-label">{new Date(trend.date).toLocaleDateString('zh-CN', { month: 'short', day: 'numeric' })}</div>
								</div>
							{/each}
						</div>
					</div>
				</div>
			</div>

			<!-- 热门页面 -->
			<div class="chart-card">
				<div class="card-header">
					<h3>热门页面</h3>
					<MousePointer size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="top-pages-list">
						{#each analytics.topPages.slice(0, 5) as page, index}
							<div class="page-item">
								<div class="page-rank">{index + 1}</div>
								<div class="page-info">
									<div class="page-path">{page.path}</div>
									<div class="page-stats">{formatNumber(page.visits)} 次访问</div>
								</div>
								<div class="page-percentage">{page.percentage}%</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- 设备分布 -->
			<div class="chart-card">
				<div class="card-header">
					<h3>设备分布</h3>
					<PieChart size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="device-list">
						{#each analytics.devices as device}
							<div class="device-item">
								<div class="device-info">
									<div class="device-name">{device.type}</div>
									<div class="device-stats">{formatNumber(device.count)} 用户</div>
								</div>
								<div class="device-percentage">{device.percentage}%</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- 浏览器分布 -->
			<div class="chart-card">
				<div class="card-header">
					<h3>浏览器分布</h3>
					<PieChart size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="browser-list">
						{#each analytics.browsers as browser}
							<div class="browser-item">
								<div class="browser-info">
									<div class="browser-name">{browser.name}</div>
									<div class="browser-stats">{formatNumber(browser.count)} 用户</div>
								</div>
								<div class="browser-percentage">{browser.percentage}%</div>
							</div>
						{/each}
					</div>
				</div>
			</div>

			<!-- 访问来源 -->
			<div class="chart-card">
				<div class="card-header">
					<h3>访问来源</h3>
					<TrendingUp size={16} class="card-icon" />
				</div>
				<div class="card-body">
					<div class="referrer-list">
						{#each analytics.referrers as referrer}
							<div class="referrer-item">
								<div class="referrer-info">
									<div class="referrer-name">{referrer.source}</div>
									<div class="referrer-stats">{formatNumber(referrer.count)} 访问</div>
								</div>
								<div class="referrer-percentage">{referrer.percentage}%</div>
							</div>
						{/each}
					</div>
				</div>
			</div>
		</div>
	{/if}
</div>

<style>
	.analytics-page {
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
		gap: 1rem;
	}

	.time-range-selector select {
		padding: 0.5rem 1rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		background: white;
	}

	.overview-section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(250px, 1fr));
		gap: 1rem;
	}

	.metric-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1.5rem;
		display: flex;
		align-items: center;
		gap: 1rem;
		transition: transform 0.2s ease, box-shadow 0.2s ease;
	}

	.metric-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	}

	.metric-icon {
		width: 48px;
		height: 48px;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.metric-icon.visits { background: #3b82f6; }
	.metric-icon.users { background: #10b981; }
	.metric-icon.pages { background: #f59e0b; }
	.metric-icon.duration { background: #8b5cf6; }

	.metric-value {
		font-size: 2rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
		margin-bottom: 0.25rem;
	}

	.metric-label {
		font-size: 0.875rem;
		color: #6b7280;
		margin-bottom: 0.5rem;
	}

	.metric-change {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.75rem;
		font-weight: 500;
	}

	.metric-change.positive {
		color: #10b981;
	}

	.metric-change.negative {
		color: #ef4444;
	}

	.charts-section {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(400px, 1fr));
		gap: 1.5rem;
	}

	.chart-card.large {
		grid-column: 1 / -1;
	}

	.chart-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		overflow: hidden;
	}

	.card-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.card-header h3 {
		margin: 0;
		font-size: 1rem;
		font-weight: 600;
		color: #111827;
	}

	.card-icon {
		color: #6b7280;
	}

	.chart-legend {
		display: flex;
		gap: 1rem;
	}

	.legend-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: #6b7280;
	}

	.legend-color {
		width: 12px;
		height: 12px;
		border-radius: 2px;
	}

	.legend-color.visits {
		background: #3b82f6;
	}

	.legend-color.visitors {
		background: #10b981;
	}

	.card-body {
		padding: 1.5rem;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		color: #6b7280;
	}

	.loading-spinner {
		width: 32px;
		height: 32px;
		border: 3px solid #e5e7eb;
		border-top: 3px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 1rem;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
		100% { transform: rotate(360deg); }
	}

	.chart-placeholder {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		min-height: 200px;
	}

	.trend-chart {
		min-height: 300px;
		justify-content: flex-end;
	}

	.chart-grid {
		display: flex;
		align-items: end;
		justify-content: space-between;
		width: 100%;
		height: 250px;
		gap: 0.5rem;
		padding: 0 1rem;
	}

	.chart-bar-container {
		display: flex;
		flex-direction: column;
		align-items: center;
		flex: 1;
		max-width: 80px;
	}

	.chart-bars {
		display: flex;
		align-items: end;
		gap: 2px;
		height: 200px;
		margin-bottom: 0.5rem;
	}

	.chart-bar {
		width: 20px;
		background: linear-gradient(to top, rgba(59, 130, 246, 0.8), rgba(59, 130, 246, 0.4));
		border-radius: 2px 2px 0 0;
		transition: height 0.3s ease;
	}

	.chart-bar.visitors {
		background: linear-gradient(to top, rgba(16, 185, 129, 0.8), rgba(16, 185, 129, 0.4));
	}

	.chart-label {
		font-size: 0.75rem;
		color: #6b7280;
		text-align: center;
	}

	.top-pages-list,
	.device-list,
	.browser-list,
	.referrer-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.page-item,
	.device-item,
	.browser-item,
	.referrer-item {
		display: flex;
		align-items: center;
		gap: 1rem;
		padding: 0.75rem;
		background: #f9fafb;
		border-radius: 6px;
		transition: background-color 0.2s ease;
	}

	.page-item:hover,
	.device-item:hover,
	.browser-item:hover,
	.referrer-item:hover {
		background: #f3f4f6;
	}

	.page-rank {
		width: 24px;
		height: 24px;
		background: #3b82f6;
		color: white;
		border-radius: 50%;
		display: flex;
		align-items: center;
		justify-content: center;
		font-size: 0.75rem;
		font-weight: 600;
	}

	.page-info,
	.device-info,
	.browser-info,
	.referrer-info {
		flex: 1;
	}

	.page-path,
	.device-name,
	.browser-name,
	.referrer-name {
		font-weight: 500;
		color: #111827;
		font-size: 0.875rem;
	}

	.page-stats,
	.device-stats,
	.browser-stats,
	.referrer-stats {
		font-size: 0.75rem;
		color: #6b7280;
		margin-top: 0.125rem;
	}

	.page-percentage,
	.device-percentage,
	.browser-percentage,
	.referrer-percentage {
		font-weight: 600;
		color: #374151;
		font-size: 0.875rem;
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

	.btn-primary:hover {
		background-color: #2563eb;
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
			justify-content: space-between;
		}

		.overview-section {
			grid-template-columns: repeat(2, 1fr);
		}

		.charts-section {
			grid-template-columns: 1fr;
		}

		.chart-card.large {
			grid-column: 1;
		}

		.chart-grid {
			padding: 0 0.5rem;
		}

		.chart-bar {
			width: 16px;
		}
	}

	@media (max-width: 480px) {
		.overview-section {
			grid-template-columns: 1fr;
		}

		.chart-legend {
			flex-direction: column;
			gap: 0.5rem;
		}
	}
</style>