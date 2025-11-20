<script>
	import { onMount } from 'svelte';
	import { Database, Plus, Search, Eye, Edit, Trash2, Download, Upload, RefreshCw, Table, Key, Calendar, FileText } from 'lucide-svelte';

	// 状态管理
	let loading = false;
	let searchQuery = '';
	let selectedTable = null;
	let showCreateModal = false;
	let showViewModal = false;
	let showEditModal = false;

	// 表数据
	let tables = [];
	let currentTableData = [];
	let currentSchema = [];

	// 分页
	let currentPage = 1;
	let pageSize = 20;
	let totalPages = 1;

	// 新建表单
	let createForm = {
		name: '',
		description: '',
		fields: [
			{ name: 'id', type: 'INTEGER', nullable: false, primary: true, autoIncrement: true }
		]
	};

	// 生成模拟表数据
	function generateMockTables() {
		return [
			{
				id: 1,
				name: 'users',
				description: '用户信息表',
				rows: 1250,
				size: '2.4 MB',
				created: '2024-01-15',
				updated: '2024-11-20',
				engine: 'SQLite',
				collation: 'UTF-8'
			},
			{
				id: 2,
				name: 'orders',
				description: '订单信息表',
				rows: 3420,
				size: '8.7 MB',
				created: '2024-02-01',
				updated: '2024-11-19',
				engine: 'SQLite',
				collation: 'UTF-8'
			},
			{
				id: 3,
				name: 'products',
				description: '产品信息表',
				rows: 567,
				size: '1.2 MB',
				created: '2024-01-20',
				updated: '2024-11-18',
				engine: 'SQLite',
				collation: 'UTF-8'
			},
			{
				id: 4,
				name: 'categories',
				description: '产品分类表',
				rows: 25,
				size: '0.1 MB',
				created: '2024-01-10',
				updated: '2024-11-15',
				engine: 'SQLite',
				collation: 'UTF-8'
			},
			{
				id: 5,
				name: 'logs',
				description: '系统日志表',
				rows: 15420,
				size: '45.3 MB',
				created: '2024-01-01',
				updated: '2024-11-20',
				engine: 'SQLite',
				collation: 'UTF-8'
			}
		];
	}

	function generateMockSchema(tableName) {
		const schemas = {
			users: [
				{ name: 'id', type: 'INTEGER', nullable: false, primary: true, autoIncrement: true },
				{ name: 'username', type: 'VARCHAR(50)', nullable: false, primary: false },
				{ name: 'email', type: 'VARCHAR(100)', nullable: false, primary: false },
				{ name: 'password_hash', type: 'VARCHAR(255)', nullable: false, primary: false },
				{ name: 'created_at', type: 'DATETIME', nullable: false, primary: false },
				{ name: 'updated_at', type: 'DATETIME', nullable: false, primary: false }
			],
			orders: [
				{ name: 'id', type: 'INTEGER', nullable: false, primary: true, autoIncrement: true },
				{ name: 'user_id', type: 'INTEGER', nullable: false, primary: false },
				{ name: 'total_amount', type: 'DECIMAL(10,2)', nullable: false, primary: false },
				{ name: 'status', type: 'VARCHAR(20)', nullable: false, primary: false },
				{ name: 'created_at', type: 'DATETIME', nullable: false, primary: false }
			],
			products: [
				{ name: 'id', type: 'INTEGER', nullable: false, primary: true, autoIncrement: true },
				{ name: 'name', type: 'VARCHAR(100)', nullable: false, primary: false },
				{ name: 'price', type: 'DECIMAL(10,2)', nullable: false, primary: false },
				{ name: 'category_id', type: 'INTEGER', nullable: true, primary: false },
				{ name: 'description', type: 'TEXT', nullable: true, primary: false }
			]
		};

		return schemas[tableName] || [];
	}

	function generateMockData(tableName) {
		if (tableName === 'users') {
			return [
				{ id: 1, username: 'admin', email: 'admin@example.com', created_at: '2024-01-15 10:00:00' },
				{ id: 2, username: 'user1', email: 'user1@example.com', created_at: '2024-01-16 14:30:00' },
				{ id: 3, username: 'user2', email: 'user2@example.com', created_at: '2024-01-17 09:15:00' }
			];
		}
		if (tableName === 'orders') {
			return [
				{ id: 1, user_id: 1, total_amount: 299.99, status: 'completed' },
				{ id: 2, user_id: 2, total_amount: 199.50, status: 'pending' },
				{ id: 3, user_id: 3, total_amount: 449.00, status: 'shipped' }
			];
		}
		return [];
	}

	onMount(() => {
		loadTables();
	});

	// 过滤表
	$: filteredTables = tables.filter(table =>
		table.name.toLowerCase().includes(searchQuery.toLowerCase()) ||
		table.description.toLowerCase().includes(searchQuery.toLowerCase())
	);

	// 分页
	$: paginatedTables = filteredTables.slice(
		(currentPage - 1) * pageSize,
		currentPage * pageSize
	);

	$: totalPages = Math.ceil(filteredTables.length / pageSize);

	function loadTables() {
		loading = true;
		setTimeout(() => {
			tables = generateMockTables();
			loading = false;
		}, 800);
	}

	function viewTable(table) {
		selectedTable = table;
		currentSchema = generateMockSchema(table.name);
		currentTableData = generateMockData(table.name);
		showViewModal = true;
	}

	function editTable(table) {
		selectedTable = table;
		showEditModal = true;
	}

	function deleteTable(table) {
		if (confirm(`确定要删除表 "${table.name}" 吗？此操作不可撤销。`)) {
			tables = tables.filter(t => t.id !== table.id);
		}
	}

	function addField() {
		createForm.fields.push({
			name: '',
			type: 'VARCHAR(255)',
			nullable: true,
			primary: false,
			autoIncrement: false
		});
	}

	function removeField(index) {
		if (createForm.fields.length > 1) {
			createForm.fields.splice(index, 1);
		}
	}

	function createTable() {
		if (!createForm.name) {
			alert('请输入表名');
			return;
		}

		// 检查表名是否已存在
		if (tables.some(t => t.name === createForm.name)) {
			alert('表名已存在');
			return;
		}

		const newTable = {
			id: Math.max(...tables.map(t => t.id)) + 1,
			name: createForm.name,
			description: createForm.description,
			rows: 0,
			size: '0 KB',
			created: new Date().toISOString().split('T')[0],
			updated: new Date().toISOString().split('T')[0],
			engine: 'SQLite',
			collation: 'UTF-8'
		};

		tables.unshift(newTable);
		showCreateModal = false;
		resetCreateForm();
	}

	function resetCreateForm() {
		createForm = {
			name: '',
			description: '',
			fields: [
				{ name: 'id', type: 'INTEGER', nullable: false, primary: true, autoIncrement: true }
			]
		};
	}

	function exportTable(table) {
		const data = {
			table: table.name,
			schema: generateMockSchema(table.name),
			data: generateMockData(table.name),
			exportDate: new Date().toISOString()
		};

		const blob = new Blob([JSON.stringify(data, null, 2)], { type: 'application/json' });
		const url = URL.createObjectURL(blob);
		const a = document.createElement('a');
		a.href = url;
		a.download = `${table.name}_${new Date().toISOString().split('T')[0]}.json`;
		a.click();
		URL.revokeObjectURL(url);
	}

	function formatSize(size) {
		return size;
	}

	function formatDate(dateStr) {
		return new Date(dateStr).toLocaleDateString('zh-CN');
	}

	function getFieldTypeColor(type) {
		if (type.includes('INT')) return '#3b82f6';
		if (type.includes('VARCHAR') || type.includes('TEXT')) return '#10b981';
		if (type.includes('DECIMAL') || type.includes('FLOAT')) return '#f59e0b';
		if (type.includes('DATETIME') || type.includes('DATE')) return '#8b5cf6';
		return '#6b7280';
	}
</script>

<div class="tables-page">
	<!-- 页面标题 -->
	<div class="page-header">
		<div class="header-left">
			<h1>表管理</h1>
			<p class="page-description">数据表结构和管理</p>
		</div>
		<div class="header-right">
			<button class="btn btn-primary" on:click={() => showCreateModal = true}>
				<Plus size={16} />
				新建表
			</button>
		</div>
	</div>

	<!-- 搜索和过滤器 -->
	<div class="filters-section">
		<div class="search-bar">
			<div class="search-input">
				<Search size={18} class="search-icon" />
				<input
					type="text"
					placeholder="搜索表名或描述..."
					bind:value={searchQuery}
				/>
			</div>
		</div>

		<div class="stats-cards">
			<div class="stat-card">
				<div class="stat-icon">
					<Database size={20} />
				</div>
				<div class="stat-content">
					<div class="stat-value">{tables.length}</div>
					<div class="stat-label">总表数</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon">
					<FileText size={20} />
				</div>
				<div class="stat-content">
					<div class="stat-value">
						{tables.reduce((sum, table) => sum + table.rows, 0).toLocaleString()}
					</div>
					<div class="stat-label">总记录数</div>
				</div>
			</div>
			<div class="stat-card">
				<div class="stat-icon">
					<Database size={20} />
				</div>
				<div class="stat-content">
					<div class="stat-value">
						{tables.reduce((sum, table) => {
							const size = parseFloat(table.size);
							return sum + (isNaN(size) ? 0 : size);
						}, 0).toFixed(1)} MB
					</div>
					<div class="stat-label">总大小</div>
				</div>
			</div>
		</div>
	</div>

	<!-- 表格列表 -->
	<div class="tables-section">
		{#if loading}
			<div class="loading-state">
				<div class="loading-spinner"></div>
				<p>加载数据表中...</p>
			</div>
		{:else if paginatedTables.length === 0}
			<div class="empty-state">
				<Database size={48} />
				<h3>暂无数据表</h3>
				<p>点击"新建表"开始创建您的第一个数据表</p>
				<button class="btn btn-primary" on:click={() => showCreateModal = true}>
					<Plus size={16} />
					新建表
				</button>
			</div>
		{:else}
			<div class="tables-grid">
				{#each paginatedTables as table}
					<div class="table-card">
						<div class="table-header">
							<div class="table-info">
								<h3 class="table-name">{table.name}</h3>
								<p class="table-description">{table.description}</p>
							</div>
							<div class="table-actions">
								<button
									class="action-btn"
									title="查看表"
									on:click={() => viewTable(table)}
								>
									<Eye size={16} />
								</button>
								<button
									class="action-btn"
									title="编辑表"
									on:click={() => editTable(table)}
								>
									<Edit size={16} />
								</button>
								<button
									class="action-btn"
									title="导出表"
									on:click={() => exportTable(table)}
								>
									<Download size={16} />
								</button>
								<button
									class="action-btn danger"
									title="删除表"
									on:click={() => deleteTable(table)}
								>
									<Trash2 size={16} />
								</button>
							</div>
						</div>

						<div class="table-stats">
							<div class="stat-item">
								<Table size={14} />
								<span>{table.rows.toLocaleString()} 行</span>
							</div>
							<div class="stat-item">
								<Database size={14} />
								<span>{formatSize(table.size)}</span>
							</div>
						</div>

						<div class="table-meta">
							<div class="meta-item">
								<Calendar size={14} />
								<span>创建于 {formatDate(table.created)}</span>
							</div>
							<div class="meta-item">
								<RefreshCw size={14} />
								<span>更新于 {formatDate(table.updated)}</span>
							</div>
						</div>
					</div>
				{/each}
			</div>

			<!-- 分页 -->
			{#if totalPages > 1}
				<div class="pagination">
					<div class="pagination-info">
						显示 {((currentPage - 1) * pageSize) + 1}-{Math.min(currentPage * pageSize, filteredTables.length)}
						共 {filteredTables.length} 个表
					</div>
					<div class="pagination-controls">
						<button
							class="btn btn-secondary btn-sm"
							disabled={currentPage === 1}
							on:click={() => currentPage = 1}
						>
							首页
						</button>
						<button
							class="btn btn-secondary btn-sm"
							disabled={currentPage === 1}
							on:click={() => currentPage -= 1}
						>
							上一页
						</button>
						<span class="page-info">
							第 {currentPage} / {totalPages} 页
						</span>
						<button
							class="btn btn-secondary btn-sm"
							disabled={currentPage === totalPages}
							on:click={() => currentPage += 1}
						>
							下一页
						</button>
						<button
							class="btn btn-secondary btn-sm"
							disabled={currentPage === totalPages}
							on:click={() => currentPage = totalPages}
						>
							末页
						</button>
					</div>
				</div>
			{/if}
		{/if}
	</div>
</div>

<!-- 新建表模态框 -->
{#if showCreateModal}
	<div class="modal-overlay" on:click={() => showCreateModal = false}>
		<div class="modal-content" on:click|stopPropagation>
			<div class="modal-header">
				<h2>新建数据表</h2>
				<button class="close-btn" on:click={() => showCreateModal = false}>×</button>
			</div>
			<div class="modal-body">
				<div class="form-group">
					<label for="tableName">表名 *</label>
					<input
						id="tableName"
						type="text"
						bind:value={createForm.name}
						placeholder="请输入表名"
						required
					/>
				</div>
				<div class="form-group">
					<label for="tableDescription">描述</label>
					<textarea
						id="tableDescription"
						bind:value={createForm.description}
						placeholder="请输入表描述"
						rows="3"
					></textarea>
				</div>

				<div class="form-group">
					<label>字段定义</label>
					<div class="fields-list">
						{#each createForm.fields as field, index}
							<div class="field-item">
								<input
									type="text"
									bind:value={field.name}
									placeholder="字段名"
									class="field-name"
								/>
								<select bind:value={field.type} class="field-type">
									<option value="INTEGER">INTEGER</option>
									<option value="VARCHAR(255)">VARCHAR(255)</option>
									<option value="TEXT">TEXT</option>
									<option value="DECIMAL(10,2)">DECIMAL(10,2)</option>
									<option value="DATETIME">DATETIME</option>
									<option value="BOOLEAN">BOOLEAN</option>
								</select>
								<label class="checkbox-label">
									<input
										type="checkbox"
										bind:checked={field.nullable}
									/>
									<span>可空</span>
								</label>
								<label class="checkbox-label">
									<input
										type="checkbox"
										bind:checked={field.primary}
									/>
									<span>主键</span>
								</label>
								<button
									class="btn btn-secondary btn-sm"
									on:click={() => removeField(index)}
									disabled={createForm.fields.length === 1}
								>
									<Trash2 size={14} />
								</button>
							</div>
						{/each}
						<button class="btn btn-secondary btn-sm" on:click={addField}>
							<Plus size={14} />
							添加字段
						</button>
					</div>
				</div>
			</div>
			<div class="modal-footer">
				<button class="btn btn-secondary" on:click={() => showCreateModal = false}>
					取消
				</button>
				<button class="btn btn-primary" on:click={createTable}>
					创建表
				</button>
			</div>
		</div>
	</div>
{/if}

<!-- 查看表模态框 -->
{#if showViewModal && selectedTable}
	<div class="modal-overlay" on:click={() => showViewModal = false}>
		<div class="modal-content large" on:click|stopPropagation>
			<div class="modal-header">
				<h2>查看表 - {selectedTable.name}</h2>
				<button class="close-btn" on:click={() => showViewModal = false}>×</button>
			</div>
			<div class="modal-body">
				<div class="tabs">
					<button class="tab active">表结构</button>
					<button class="tab">数据预览</button>
				</div>

				<div class="tab-content">
					<div class="schema-section">
						<h3>表结构</h3>
						<div class="schema-table">
							<table>
								<thead>
									<tr>
										<th>字段名</th>
										<th>类型</th>
										<th>可空</th>
										<th>主键</th>
										<th>自增</th>
									</tr>
								</thead>
								<tbody>
									{#each currentSchema as field}
										<tr>
											<td class="field-name">
												{#if field.primary}
													<Key size={14} class="primary-key" />
												{/if}
												{field.name}
											</td>
											<td>
												<span class="field-type" style="color: {getFieldTypeColor(field.type)}">
													{field.type}
												</span>
											</td>
											<td>{field.nullable ? '是' : '否'}</td>
											<td>{field.primary ? '是' : '否'}</td>
											<td>{field.autoIncrement ? '是' : '否'}</td>
										</tr>
									{/each}
								</tbody>
							</table>
						</div>
					</div>
				</div>
			</div>
		</div>
	</div>
{/if}

<style>
	.tables-page {
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

	.filters-section {
		display: flex;
		flex-direction: column;
		gap: 1rem;
	}

	.search-bar {
		width: 100%;
		max-width: 400px;
	}

	.search-input {
		position: relative;
	}

	.search-icon {
		position: absolute;
		left: 0.75rem;
		top: 50%;
		transform: translateY(-50%);
		color: #9ca3af;
	}

	.search-input input {
		width: 100%;
		padding: 0.75rem 1rem 0.75rem 2.5rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s ease;
	}

	.search-input input:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.stats-cards {
		display: grid;
		grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
		gap: 1rem;
	}

	.stat-card {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1rem;
		display: flex;
		align-items: center;
		gap: 0.75rem;
	}

	.stat-icon {
		width: 40px;
		height: 40px;
		background: #eff6ff;
		color: #3b82f6;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.stat-value {
		font-size: 1.5rem;
		font-weight: 700;
		color: #111827;
		line-height: 1;
	}

	.stat-label {
		font-size: 0.875rem;
		color: #6b7280;
	}

	.tables-section {
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 8px;
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

	.empty-state {
		display: flex;
		flex-direction: column;
		align-items: center;
		justify-content: center;
		padding: 4rem 2rem;
		text-align: center;
		color: #6b7280;
	}

	.empty-state h3 {
		margin: 1rem 0 0.5rem 0;
		font-size: 1.25rem;
		color: #374151;
	}

	.empty-state p {
		margin: 0 0 1.5rem 0;
	}

	.tables-grid {
		display: grid;
		grid-template-columns: repeat(auto-fill, minmax(350px, 1fr));
		gap: 1rem;
	}

	.table-card {
		border: 1px solid #e5e7eb;
		border-radius: 8px;
		padding: 1.25rem;
		transition: box-shadow 0.2s ease, transform 0.2s ease;
	}

	.table-card:hover {
		box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
		transform: translateY(-2px);
	}

	.table-header {
		display: flex;
		align-items: flex-start;
		justify-content: space-between;
		margin-bottom: 1rem;
	}

	.table-name {
		margin: 0 0 0.5rem 0;
		font-size: 1.125rem;
		font-weight: 600;
		color: #111827;
	}

	.table-description {
		margin: 0;
		font-size: 0.875rem;
		color: #6b7280;
		line-height: 1.4;
	}

	.table-actions {
		display: flex;
		gap: 0.5rem;
	}

	.action-btn {
		padding: 0.5rem;
		background: white;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		color: #6b7280;
		cursor: pointer;
		transition: all 0.2s ease;
		display: flex;
		align-items: center;
		justify-content: center;
	}

	.action-btn:hover {
		background: #f3f4f6;
		color: #374151;
		border-color: #9ca3af;
	}

	.action-btn.danger:hover {
		background: #fef2f2;
		color: #991b1b;
		border-color: #fecaca;
	}

	.table-stats {
		display: flex;
		gap: 1rem;
		margin-bottom: 1rem;
	}

	.stat-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.875rem;
		color: #6b7280;
	}

	.table-meta {
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
	}

	.meta-item {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-size: 0.75rem;
		color: #9ca3af;
	}

	.pagination {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1rem 0 0 0;
		border-top: 1px solid #e5e7eb;
		margin-top: 1rem;
	}

	.pagination-info {
		font-size: 0.875rem;
		color: #6b7280;
	}

	.pagination-controls {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.page-info {
		padding: 0 1rem;
		font-size: 0.875rem;
		color: #6b7280;
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
		padding: 1rem;
	}

	.modal-content {
		background: white;
		border-radius: 8px;
		max-width: 600px;
		width: 100%;
		max-height: 90vh;
		overflow-y: auto;
	}

	.modal-content.large {
		max-width: 900px;
	}

	.modal-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 1.5rem;
		border-bottom: 1px solid #e5e7eb;
	}

	.modal-header h2 {
		margin: 0;
		font-size: 1.25rem;
		font-weight: 600;
		color: #111827;
	}

	.close-btn {
		background: none;
		border: none;
		font-size: 1.5rem;
		color: #6b7280;
		cursor: pointer;
		padding: 0;
		width: 32px;
		height: 32px;
		display: flex;
		align-items: center;
		justify-content: center;
		border-radius: 6px;
		transition: background 0.2s ease;
	}

	.close-btn:hover {
		background: #f3f4f6;
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
		display: flex;
		flex-direction: column;
		gap: 0.5rem;
		margin-bottom: 1.5rem;
	}

	.form-group label {
		font-size: 0.875rem;
		font-weight: 500;
		color: #374151;
	}

	.form-group input,
	.form-group textarea {
		padding: 0.75rem;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		font-size: 0.875rem;
		transition: border-color 0.2s ease, box-shadow 0.2s ease;
	}

	.form-group input:focus,
	.form-group textarea:focus {
		outline: none;
		border-color: #3b82f6;
		box-shadow: 0 0 0 3px rgba(59, 130, 246, 0.1);
	}

	.fields-list {
		display: flex;
		flex-direction: column;
		gap: 0.75rem;
	}

	.field-item {
		display: grid;
		grid-template-columns: 1fr 150px 60px 60px 40px;
		gap: 0.5rem;
		align-items: center;
		padding: 0.75rem;
		border: 1px solid #e5e7eb;
		border-radius: 6px;
		background: #f9fafb;
	}

	.field-name,
	.field-type {
		padding: 0.5rem;
		border: 1px solid #d1d5db;
		border-radius: 4px;
		font-size: 0.875rem;
	}

	.checkbox-label {
		display: flex;
		align-items: center;
		gap: 0.25rem;
		font-size: 0.75rem;
		color: #374151;
	}

	.checkbox-label input[type="checkbox"] {
		width: auto;
		margin: 0;
	}

	.tabs {
		display: flex;
		border-bottom: 1px solid #e5e7eb;
		margin-bottom: 1.5rem;
	}

	.tab {
		padding: 0.75rem 1.5rem;
		background: none;
		border: none;
		border-bottom: 2px solid transparent;
		color: #6b7280;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.tab.active {
		color: #3b82f6;
		border-bottom-color: #3b82f6;
	}

	.schema-table table {
		width: 100%;
		border-collapse: collapse;
	}

	.schema-table th {
		background: #f9fafb;
		padding: 0.75rem;
		text-align: left;
		font-size: 0.75rem;
		font-weight: 600;
		color: #6b7280;
		text-transform: uppercase;
		letter-spacing: 0.05em;
		border-bottom: 1px solid #e5e7eb;
	}

	.schema-table td {
		padding: 0.75rem;
		border-bottom: 1px solid #f3f4f6;
		font-size: 0.875rem;
	}

	.field-name {
		display: flex;
		align-items: center;
		gap: 0.5rem;
		font-weight: 500;
		color: #374151;
	}

	.primary-key {
		color: #f59e0b;
	}

	.field-type {
		font-family: 'Monaco', 'Menlo', monospace;
		font-weight: 500;
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

	.btn-secondary {
		background-color: white;
		color: #374151;
		border-color: #d1d5db;
	}

	.btn-secondary:hover {
		background-color: #f9fafb;
		border-color: #9ca3af;
	}

	.btn-sm {
		padding: 0.375rem 0.75rem;
		font-size: 0.75rem;
	}

	/* 响应式设计 */
	@media (max-width: 768px) {
		.page-header {
			flex-direction: column;
			align-items: flex-start;
			gap: 1rem;
		}

		.stats-cards {
			grid-template-columns: 1fr;
		}

		.tables-grid {
			grid-template-columns: 1fr;
		}

		.field-item {
			grid-template-columns: 1fr;
			gap: 0.5rem;
		}

		.pagination {
			flex-direction: column;
			gap: 1rem;
			align-items: stretch;
		}

		.pagination-controls {
			justify-content: center;
		}

		.modal-content {
			margin: 1rem;
			max-width: none;
		}
	}
</style>