<script>
	import { onMount } from 'svelte';
	import { Folder, Upload, Search, Filter, Download, Trash2, Eye, HardDrive, FileText, Image, Film, Music, Archive } from 'lucide-svelte';

	let files = [];
	let loading = false;
	let searchQuery = '';
	let selectedType = 'all';
	let sortBy = 'name';
	let currentPage = 1;
	let pageSize = 12;

	onMount(async () => {
		await loadFiles();
	});

	async function loadFiles() {
		loading = true;
		try {
			// 模拟API调用
			await new Promise(resolve => setTimeout(resolve, 500));
			files = [
				{
					id: 1,
					name: 'user_avatar.png',
					type: 'image',
					size: 245760,
					mimeType: 'image/png',
					url: '/files/user_avatar.png',
					createdAt: '2024-01-20T10:30:00Z',
					accessCount: 156,
					storage: 'local'
				},
				{
					id: 2,
					name: 'project_report.pdf',
					type: 'document',
					size: 1048576,
					mimeType: 'application/pdf',
					url: '/files/project_report.pdf',
					createdAt: '2024-01-19T14:15:00Z',
					accessCount: 89,
					storage: 's3'
				},
				{
					id: 3,
					name: 'demo_video.mp4',
					type: 'video',
					size: 10485760,
					mimeType: 'video/mp4',
					url: '/files/demo_video.mp4',
					createdAt: '2024-01-18T16:45:00Z',
					accessCount: 234,
					storage: 'minio'
				},
				{
					id: 4,
					name: 'backup_data.zip',
					type: 'archive',
					size: 52428800,
					mimeType: 'application/zip',
					url: '/files/backup_data.zip',
					createdAt: '2024-01-17T09:20:00Z',
					accessCount: 12,
					storage: 'local'
				}
			];
		} catch (error) {
			console.error('Failed to load files:', error);
		} finally {
			loading = false;
		}
	}

	$: filteredFiles = files.filter(file => {
		const matchesSearch = file.name.toLowerCase().includes(searchQuery.toLowerCase());
		const matchesType = selectedType === 'all' || file.type === selectedType;
		return matchesSearch && matchesType;
	});

	$: sortedFiles = [...filteredFiles].sort((a, b) => {
		switch (sortBy) {
			case 'name': return a.name.localeCompare(b.name);
			case 'size': return b.size - a.size;
			case 'created': return new Date(b.createdAt) - new Date(a.createdAt);
			case 'access': return b.accessCount - a.accessCount;
			default: return 0;
		}
	});

	$: paginatedFiles = sortedFiles.slice(
		(currentPage - 1) * pageSize,
		currentPage * pageSize
	);

	$: totalPages = Math.ceil(sortedFiles.length / pageSize);

	function formatFileSize(bytes) {
		const sizes = ['B', 'KB', 'MB', 'GB'];
		if (bytes === 0) return '0 B';
		const i = Math.floor(Math.log(bytes) / Math.log(1024));
		return Math.round(bytes / Math.pow(1024, i) * 100) / 100 + ' ' + sizes[i];
	}

	function getFileIcon(type) {
		switch (type) {
			case 'image': return Image;
			case 'document': return FileText;
			case 'video': return Film;
			case 'audio': return Music;
			case 'archive': return Archive;
			default: return FileText;
		}
	}

	function getStorageBadge(storage) {
		switch (storage) {
			case 's3': return 'warning';
			case 'minio': return 'info';
			default: return 'success';
		}
	}

	const storageStats = {
		totalSpace: 107374182400, // 100GB
		usedSpace: 34603008000,  // ~32.2GB
		fileCount: 1523,
		typeDistribution: {
			image: 456,
			document: 623,
			video: 89,
			audio: 156,
			archive: 199
		}
	};

	$: usagePercentage = Math.round((storageStats.usedSpace / storageStats.totalSpace) * 100);
</script>

<div class="files-page">
	<div class="page-header">
		<div class="header-content">
			<h1>文件管理</h1>
			<p>管理文件存储、上传下载和权限控制</p>
		</div>
		<div class="header-actions">
			<button class="btn btn-primary">
				<Upload size={20} />
				上传文件
			</button>
		</div>
	</div>

	<!-- 存储统计卡片 -->
	<div class="stats-grid">
		<div class="stat-card">
			<div class="stat-icon">
				<HardDrive size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">{formatFileSize(storageStats.usedSpace)}</div>
				<div class="stat-label">已用存储</div>
				<div class="stat-progress">
					<div class="progress-bar">
						<div class="progress-fill" style="width: {usagePercentage}%"></div>
					</div>
					<span class="progress-text">{usagePercentage}%</span>
				</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon">
				<Folder size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">{storageStats.fileCount.toLocaleString()}</div>
				<div class="stat-label">文件总数</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon">
				<Image size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">{storageStats.typeDistribution.image}</div>
				<div class="stat-label">图片文件</div>
			</div>
		</div>

		<div class="stat-card">
			<div class="stat-icon">
				<FileText size={24} />
			</div>
			<div class="stat-content">
				<div class="stat-value">{storageStats.typeDistribution.document}</div>
				<div class="stat-label">文档文件</div>
			</div>
		</div>
	</div>

	<!-- 筛选和搜索 -->
	<div class="filters-section">
		<div class="filter-group">
			<div class="search-box">
				<Search size={20} />
				<input
					type="text"
					placeholder="搜索文件名..."
					bind:value={searchQuery}
				/>
			</div>
			<div class="filter-select">
				<select bind:value={selectedType}>
					<option value="all">所有类型</option>
					<option value="image">图片</option>
					<option value="document">文档</option>
					<option value="video">视频</option>
					<option value="audio">音频</option>
					<option value="archive">压缩包</option>
				</select>
			</div>
			<div class="filter-select">
				<select bind:value={sortBy}>
					<option value="name">按名称</option>
					<option value="size">按大小</option>
					<option value="created">按创建时间</option>
					<option value="access">按访问次数</option>
				</select>
			</div>
		</div>
		<div class="filter-actions">
			<button class="btn btn-secondary">
				<Filter size={16} />
				高级筛选
			</button>
		</div>
	</div>

	{#if loading}
		<div class="loading-state">
			<div class="spinner"></div>
			<p>加载文件数据中...</p>
		</div>
	{:else}
		<div class="files-grid">
			{#each paginatedFiles as file}
				<div class="file-card">
					<div class="file-preview">
						<svelte:component this={getFileIcon(file.type)} size={48} />
					</div>
					<div class="file-info">
						<div class="file-name" title={file.name}>{file.name}</div>
						<div class="file-meta">
							<span class="file-size">{formatFileSize(file.size)}</span>
							<span class="file-storage">
								<span class="badge badge-{getStorageBadge(file.storage)}">
									{file.storage.toUpperCase()}
								</span>
							</span>
						</div>
					</div>
					<div class="file-stats">
						<div class="stat-item">
							<span class="stat-label">访问次数</span>
							<span class="stat-value">{file.accessCount}</span>
						</div>
						<div class="stat-item">
							<span class="stat-label">创建时间</span>
							<span class="stat-value">{new Date(file.createdAt).toLocaleDateString('zh-CN')}</span>
						</div>
					</div>
					<div class="file-actions">
						<button class="btn btn-sm btn-secondary" title="预览">
							<Eye size={16} />
						</button>
						<button class="btn btn-sm btn-secondary" title="下载">
							<Download size={16} />
						</button>
						<button class="btn btn-sm btn-danger" title="删除">
							<Trash2 size={16} />
						</button>
					</div>
				</div>
			{/each}
		</div>

		{#if totalPages > 1}
			<div class="pagination">
				<button class="btn btn-secondary" disabled={currentPage === 1}>
					上一页
				</button>
				<span class="page-info">
					第 {currentPage} 页，共 {totalPages} 页
				</span>
				<button class="btn btn-secondary" disabled={currentPage === totalPages}>
					下一页
				</button>
			</div>
		{/if}
	{/if}
</div>

<style>
	.files-page {
		display: flex;
		flex-direction: column;
		gap: 1.5rem;
	}

	.page-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
	}

	.header-content h1 {
		margin: 0 0 0.5rem 0;
		font-size: 1.875rem;
		font-weight: 700;
		color: #111827;
	}

	.header-content p {
		margin: 0;
		color: #6b7280;
		font-size: 0.875rem;
	}

	.header-actions {
		display: flex;
		gap: 0.5rem;
	}

	.stats-grid {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(240px, 1fr));
		gap: 1rem;
	}

	.stat-card {
		background: white;
		border-radius: 8px;
		padding: 1.5rem;
		border: 1px solid #e5e7eb;
		display: flex;
		align-items: center;
		gap: 1rem;
	}

	.stat-icon {
		width: 48px;
		height: 48px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
	}

	.stat-content {
		flex: 1;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
		margin-bottom: 0.25rem;
	}

	.stat-label {
		color: #6b7280;
		font-size: 0.875rem;
		margin-bottom: 0.5rem;
	}

	.stat-progress {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.progress-bar {
		flex: 1;
		height: 4px;
		background: #e5e7eb;
		border-radius: 2px;
		overflow: hidden;
	}

	.progress-fill {
		height: 100%;
		background: linear-gradient(90deg, #3b82f6, #2563eb);
		border-radius: 2px;
		transition: width 0.3s ease;
	}

	.progress-text {
		font-size: 0.75rem;
		color: #6b7280;
		font-weight: 500;
	}

	.filters-section {
		display: flex;
		align-items: center;
		justify-content: space-between;
		gap: 1rem;
		padding: 1rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.filter-group {
		display: flex;
		align-items: center;
		gap: 1rem;
		flex: 1;
	}

	.search-box {
		display: flex;
		align-items: center;
		background: #f9fafb;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		padding: 0.5rem 0.75rem;
		gap: 0.5rem;
		flex: 1;
		max-width: 400px;
	}

	.search-box input {
		border: none;
		background: none;
		outline: none;
		flex: 1;
		font-size: 0.875rem;
	}

	.filter-select select {
		padding: 0.5rem 0.75rem;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		background: white;
		font-size: 0.875rem;
	}

	.filter-actions {
		display: flex;
		gap: 0.5rem;
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
		border-color: #3b82f6;
	}

	.btn-primary:hover {
		background-color: #2563eb;
		border-color: #2563eb;
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

	.btn-danger {
		background-color: #ef4444;
		color: white;
		border-color: #ef4444;
	}

	.btn-danger:hover {
		background-color: #dc2626;
		border-color: #dc2626;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	.loading-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 3rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.spinner {
		width: 40px;
		height: 40px;
		border: 4px solid #e5e7eb;
		border-top: 4px solid #3b82f6;
		border-radius: 50%;
		animation: spin 1s linear infinite;
		margin-bottom: 1rem;
	}

	@keyframes spin {
		0% { transform: rotate(0deg); }
		100% { transform: rotate(360deg); }
	}

	.files-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
		gap: 1rem;
	}

	.file-card {
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
		overflow: hidden;
		transition: transform 0.2s ease, box-shadow 0.2s ease;
	}

	.file-card:hover {
		transform: translateY(-2px);
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
	}

	.file-preview {
		display: flex;
		align-items: center;
		justify-content: center;
		height: 120px;
		background: #f9fafb;
		color: #6b7280;
	}

	.file-info {
		padding: 1rem;
		border-bottom: 1px solid #f3f4f6;
	}

	.file-name {
		font-weight: 600;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.5rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.file-meta {
		display: flex;
		align-items: center;
		justify-content: space-between;
		font-size: 0.75rem;
		color: #6b7280;
	}

	.file-size {
		font-weight: 500;
	}

	.badge {
		display: inline-flex;
		align-items: center;
		padding: 0.125rem 0.375rem;
		border-radius: 9999px;
		font-size: 0.625rem;
		font-weight: 500;
	}

	.badge-success {
		background: #10b98120;
		color: #10b981;
	}

	.badge-warning {
		background: #f59e0b20;
		color: #f59e0b;
	}

	.badge-info {
		background: #3b82f620;
		color: #3b82f6;
	}

	.file-stats {
		padding: 0.75rem 1rem;
		display: flex;
		flex-direction: column;
		gap: 0.25rem;
	}

	.stat-item {
		display: flex;
		justify-content: space-between;
		align-items: center;
		font-size: 0.75rem;
	}

	.stat-label {
		color: #6b7280;
	}

	.stat-value {
		color: #111827;
		font-weight: 500;
	}

	.file-actions {
		padding: 0.75rem 1rem;
		display: flex;
		justify-content: flex-end;
		gap: 0.25rem;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 1rem;
		padding: 1rem;
		background: white;
		border-radius: 8px;
		border: 1px solid #e5e7eb;
	}

	.page-info {
		color: #6b7280;
		font-size: 0.875rem;
	}

	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
		}

		.filters-section {
			flex-direction: column;
			align-items: stretch;
		}

		.filter-group {
			flex-direction: column;
		}

		.search-box {
			max-width: none;
		}

		.files-grid {
			grid-template-columns: 1fr;
		}
	}
</style>