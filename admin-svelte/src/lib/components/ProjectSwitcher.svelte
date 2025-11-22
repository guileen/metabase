<script lang="ts">
	import { createEventDispatcher } from 'svelte';
	import { currentProject, availableProjects, projectActions } from '$lib/stores/projectStore';
	import { Building, ChevronDown, ArrowRight, Users, Settings } from 'lucide-svelte';
	import { formatProjectRole, getRoleColor } from '$lib/api';

	const dispatch = createEventDispatcher();

	// State
	let isOpen = false;
	let currentProjectValue, availableProjectsValue;

	// Subscribe to stores
	currentProject.subscribe(value => currentProjectValue = value);
	availableProjects.subscribe(value => availableProjectsValue = value);

	// Functions
	function toggleDropdown() {
		isOpen = !isOpen;
	}

	function closeDropdown() {
		isOpen = false;
	}

	function selectProject(projectId) {
		if (projectId !== currentProjectValue?.project_id) {
			projectActions.switchProject(projectId);
			dispatch('projectChanged', { projectId });
		}
		closeDropdown();
	}

	function openProjectManagement() {
		// Navigate to projects page
		window.location.href = '/projects';
		closeDropdown();
	}

	// Close dropdown when clicking outside
	function handleClickOutside(event) {
		const target = event.target;
		if (!target.closest('.project-switcher')) {
			closeDropdown();
		}
	}

	// Handle escape key
	function handleKeydown(event) {
		if (event.key === 'Escape') {
			closeDropdown();
		}
	}
</script>

<svelte:window on:click={handleClickOutside} on:keydown={handleKeydown} />

<div class="project-switcher">
	<!-- Current Project Display -->
	<div class="current-project" class:active={$currentProject} on:click={toggleDropdown}>
		{#if $currentProject}
			<div class="project-info">
				<div class="project-icon">
					<Building size={20} />
				</div>
				<div class="project-details">
					<div class="project-name">{$currentProject.project_id}</div>
					<div class="project-role" class:manage={$currentProject.can_manage}>
						{formatProjectRole($currentProject.effective_role)}
					</div>
				</div>
			</div>
			<div class="dropdown-arrow" class:rotate={isOpen}>
				<ChevronDown size={16} />
			</div>
		{:else}
			<div class="no-project">
				<div class="no-project-icon">
					<Building size={20} />
				</div>
				<div class="no-project-text">
					<div class="main-text">选择项目</div>
					<div class="sub-text">点击选择要操作的项目</div>
				</div>
				<div class="dropdown-arrow" class:rotate={isOpen}>
					<ChevronDown size={16} />
				</div>
			</div>
		{/if}
	</div>

	<!-- Dropdown Menu -->
	{#if isOpen}
		<div class="dropdown-menu">
			<div class="dropdown-header">
				<span>可访问项目 ({$availableProjects.length})</span>
				<button
					class="manage-btn"
					on:click={openProjectManagement}
					title="项目管理"
				>
					<Settings size={16} />
				</button>
			</div>

			{#if $availableProjects.length === 0}
				<div class="empty-state">
					<Building size={24} />
					<span>暂无可访问的项目</span>
					<button class="create-project-btn" on:click={openProjectManagement}>
						创建项目
					</button>
				</div>
			{:else}
				<div class="project-list">
					{#each $availableProjects as project}
						<div
							class="project-item"
							class:selected={$currentProject?.project_id === project.project_id}
							class:manage={project.can_manage}
							on:click={() => selectProject(project.project_id)}
						>
							<div class="project-item-info">
								<div class="project-item-icon">
									<Building size={18} />
								</div>
								<div class="project-item-details">
									<div class="project-item-name">{project.project_id}</div>
									<div class="project-item-meta">
										<span class={`role-badge ${getRoleColor(project.effective_role)}`}>
											{formatProjectRole(project.effective_role)}
										</span>
										{#if project.can_manage}
											<span class="manage-indicator">可管理</span>
										{/if}
									</div>
								</div>
							</div>
							{#if $currentProject?.project_id === project.project_id}
								<div class="current-indicator">
									<ArrowRight size={16} />
								</div>
							{/if}
						</div>
					{/each}
				</div>

				<div class="dropdown-footer">
					<button class="manage-all-btn" on:click={openProjectManagement}>
						<Settings size={16} />
						<span>管理所有项目</span>
					</button>
				</div>
			{/if}
		</div>
	{/if}
</div>

<style>
	.project-switcher {
		position: relative;
		width: 100%;
		max-width: 280px;
	}

	.current-project {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.625rem 0.875rem;
		background: white;
		border: 1px solid #e5e7eb;
		border-radius: 10px;
		cursor: pointer;
		transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
		min-height: 44px;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.05);
		position: relative;
		overflow: hidden;
	}

	.current-project::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: linear-gradient(135deg, rgba(59, 130, 246, 0.02), rgba(37, 99, 235, 0.05));
		opacity: 0;
		transition: opacity 0.2s ease;
	}

	.current-project:hover {
		border-color: #94a3b8;
		box-shadow: 0 2px 8px rgba(0, 0, 0, 0.08);
		transform: translateY(-1px);
	}

	.current-project:hover::before {
		opacity: 1;
	}

	.current-project.active {
		border-color: #3b82f6;
		background: linear-gradient(135deg, #f0f9ff, #e0f2fe);
		box-shadow: 0 2px 8px rgba(59, 130, 246, 0.15);
	}

	.project-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex: 1;
		min-width: 0;
	}

	.project-icon {
		flex-shrink: 0;
		width: 36px;
		height: 36px;
		background: linear-gradient(135deg, #3b82f6, #2563eb);
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: white;
		box-shadow: 0 2px 4px rgba(59, 130, 246, 0.2);
		position: relative;
		overflow: hidden;
	}

	.project-icon::before {
		content: '';
		position: absolute;
		top: 0;
		left: 0;
		right: 0;
		bottom: 0;
		background: linear-gradient(135deg, rgba(255,255,255,0.1), transparent);
		border-radius: 8px;
	}

	.project-details {
		flex: 1;
		min-width: 0;
	}

	.project-name {
		font-weight: 500;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.125rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.project-role {
		color: #6b7280;
		font-size: 0.75rem;
	}

	.project-role.manage {
		color: #059669;
		font-weight: 500;
	}

	.dropdown-arrow {
		flex-shrink: 0;
		color: #6b7280;
		transition: transform 0.2s ease;
	}

	.dropdown-arrow.rotate {
		transform: rotate(180deg);
	}

	.no-project {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		width: 100%;
	}

	.no-project-icon {
		flex-shrink: 0;
		width: 36px;
		height: 36px;
		background: linear-gradient(135deg, #f8fafc, #f1f5f9);
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #64748b;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
	}

	.no-project-text {
		flex: 1;
		min-width: 0;
	}

	.no-project-text .main-text {
		font-weight: 500;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.125rem;
	}

	.no-project-text .sub-text {
		color: #6b7280;
		font-size: 0.75rem;
	}

	/* Dropdown Menu */
	.dropdown-menu {
		position: absolute;
		top: calc(100% + 0.5rem);
		left: 0;
		right: 0;
		background: white;
		border: 1px solid #e2e8f0;
		border-radius: 12px;
		box-shadow: 0 10px 40px rgba(0, 0, 0, 0.12), 0 2px 10px rgba(0, 0, 0, 0.08);
		z-index: 1000;
		max-height: 360px;
		overflow: hidden;
		backdrop-filter: blur(10px);
		animation: slideDown 0.2s cubic-bezier(0.4, 0, 0.2, 1);
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

	.dropdown-header {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.875rem 1rem;
		border-bottom: 1px solid #f1f5f9;
		background: linear-gradient(135deg, #f8fafc, #f1f5f9);
	}

	.dropdown-header span {
		font-weight: 600;
		color: #334155;
		font-size: 0.8125rem;
		letter-spacing: 0.025em;
		text-transform: uppercase;
	}

	.manage-btn {
		padding: 0.5rem;
		background: white;
		border: 1px solid #e2e8f0;
		border-radius: 8px;
		color: #64748b;
		cursor: pointer;
		transition: all 0.2s cubic-bezier(0.4, 0, 0.2, 1);
		display: flex;
		align-items: center;
		justify-content: center;
		box-shadow: 0 1px 2px rgba(0, 0, 0, 0.04);
	}

	.manage-btn:hover {
		background: #f8fafc;
		color: #334155;
		border-color: #cbd5e1;
		transform: translateY(-1px);
		box-shadow: 0 2px 4px rgba(0, 0, 0, 0.08);
	}

	/* Empty State */
	.empty-state {
		padding: 2rem 1rem;
		text-align: center;
		color: #6b7280;
		display: flex;
		flex-direction: column;
		align-items: center;
		gap: 1rem;
	}

	.create-project-btn {
		padding: 0.5rem 1rem;
		background: #3b82f6;
		color: white;
		border: none;
		border-radius: 6px;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.create-project-btn:hover {
		background: #2563eb;
	}

	/* Project List */
	.project-list {
		max-height: 200px;
		overflow-y: auto;
	}

	.project-item {
		display: flex;
		align-items: center;
		justify-content: space-between;
		padding: 0.75rem 1rem;
		cursor: pointer;
		transition: background-color 0.2s ease;
		border-bottom: 1px solid #f3f4f6;
	}

	.project-item:hover {
		background: #f9fafb;
	}

	.project-item.selected {
		background: #eff6ff;
		border-left: 3px solid #3b82f6;
	}

	.project-item.manage {
		border-left: 3px solid #10b981;
	}

	.project-item-info {
		display: flex;
		align-items: center;
		gap: 0.75rem;
		flex: 1;
		min-width: 0;
	}

	.project-item-icon {
		flex-shrink: 0;
		width: 28px;
		height: 28px;
		background: #f3f4f6;
		border-radius: 4px;
		display: flex;
		align-items: center;
		justify-content: center;
		color: #6b7280;
	}

	.project-item.selected .project-item-icon {
		background: #dbeafe;
		color: #3b82f6;
	}

	.project-item-details {
		flex: 1;
		min-width: 0;
	}

	.project-item-name {
		font-weight: 500;
		color: #111827;
		font-size: 0.875rem;
		margin-bottom: 0.125rem;
		white-space: nowrap;
		overflow: hidden;
		text-overflow: ellipsis;
	}

	.project-item-meta {
		display: flex;
		align-items: center;
		gap: 0.5rem;
	}

	.role-badge {
		padding: 0.125rem 0.5rem;
		border-radius: 8px;
		font-size: 0.625rem;
		font-weight: 500;
		text-transform: uppercase;
	}

	.manage-indicator {
		padding: 0.125rem 0.5rem;
		background: #d1fae5;
		color: #065f46;
		border-radius: 8px;
		font-size: 0.625rem;
		font-weight: 500;
	}

	.current-indicator {
		flex-shrink: 0;
		color: #3b82f6;
	}

	.dropdown-footer {
		padding: 0.75rem 1rem;
		border-top: 1px solid #e5e7eb;
		background: #f9fafb;
	}

	.manage-all-btn {
		width: 100%;
		display: flex;
		align-items: center;
		justify-content: center;
		gap: 0.5rem;
		padding: 0.625rem;
		background: white;
		border: 1px solid #d1d5db;
		border-radius: 6px;
		color: #374151;
		font-size: 0.875rem;
		font-weight: 500;
		cursor: pointer;
		transition: all 0.2s ease;
	}

	.manage-all-btn:hover {
		background: #f3f4f6;
		border-color: #9ca3af;
	}

	/* Scrollbar styling */
	.project-list::-webkit-scrollbar {
		width: 6px;
	}

	.project-list::-webkit-scrollbar-track {
		background: #f1f5f9;
	}

	.project-list::-webkit-scrollbar-thumb {
		background: #cbd5e1;
		border-radius: 3px;
	}

	.project-list::-webkit-scrollbar-thumb:hover {
		background: #94a3b8;
	}

	/* Responsive */
	@media (max-width: 768px) {
		.project-switcher {
			max-width: 100%;
		}

		.project-name,
		.project-item-name {
			font-size: 0.75rem;
		}

		.project-role,
		.role-badge,
		.manage-indicator {
			font-size: 0.625rem;
		}
	}
</style>