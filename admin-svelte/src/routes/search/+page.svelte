<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { searchStatus, searchLoading, searchError, refreshSearchStatus, systemStore } from '$lib/stores/system';
  import { apiClient } from '$lib/api/client';
  import { Search, RefreshCw, Database, TrendingUp, Clock, AlertTriangle, CheckCircle, Play, Pause, RotateCcw, Settings } from 'lucide-svelte';

  let showRebuildConfirm = false;
  let showOptimizeConfirm = false;
  let isRebuilding = false;
  let isOptimizing = false;
  let selectedOperation = '';

  // Search operations
  async function rebuildIndex() {
    isRebuilding = true;
    try {
      const response = await apiClient.rebuildIndex();
      if (response.success) {
        systemStore.addNotification({
          type: 'success',
          title: 'Index Rebuild Started',
          message: 'Search index rebuild has been initiated'
        });
      }
    } catch (error) {
      systemStore.addNotification({
        type: 'error',
        title: 'Rebuild Failed',
        message: error.message
      });
    } finally {
      isRebuilding = false;
      showRebuildConfirm = false;
    }
  }

  async function optimizeIndex() {
    isOptimizing = true;
    try {
      const response = await apiClient.optimizeIndex();
      if (response.success) {
        systemStore.addNotification({
          type: 'success',
          title: 'Index Optimization Started',
          message: 'Search index optimization has been initiated'
        });
      }
    } catch (error) {
      systemStore.addNotification({
        type: 'error',
        title: 'Optimization Failed',
        message: error.message
      });
    } finally {
      isOptimizing = false;
      showOptimizeConfirm = false;
    }
  }

  // Auto-refresh
  let refreshInterval;

  onMount(() => {
    refreshSearchStatus();

    // Set up auto-refresh every 10 seconds
    refreshInterval = setInterval(() => {
      refreshSearchStatus();
    }, 10000);
  });

  onDestroy(() => {
    if (refreshInterval) {
      clearInterval(refreshInterval);
    }
  });

  // Reactive declarations
  $: indexingProgress = $searchStatus?.indexed_documents && $searchStatus?.total_documents
    ? ($searchStatus.indexed_documents / $searchStatus.total_documents) * 100
    : 0;

  $: healthStatus = $searchStatus?.indexing ? 'indexing' :
    $searchStatus?.search_performance?.avg_query_time < 500 ? 'healthy' : 'degraded';

  function getStatusColor(status) {
    switch (status) {
      case 'healthy': return 'text-green-600';
      case 'degraded': return 'text-yellow-600';
      case 'indexing': return 'text-blue-600';
      case 'error': return 'text-red-600';
      default: return 'text-gray-600';
    }
  }

  function getStatusBg(status) {
    switch (status) {
      case 'healthy': return 'bg-green-100 text-green-800';
      case 'degraded': return 'bg-yellow-100 text-yellow-800';
      case 'indexing': return 'bg-blue-100 text-blue-800';
      case 'error': return 'bg-red-100 text-red-800';
      default: return 'bg-gray-100 text-gray-800';
    }
  }
</script>

<svelte:head>
  <title>Search Management - MetaBase Admin</title>
</svelte:head>

<div class="search-management">
  <div class="page-header">
    <div class="header-content">
      <h1 class="page-title">
        <Search class="title-icon" size={28} />
        Search Management
      </h1>
      <p class="page-description">
        Monitor and manage the search engine index performance
      </p>
    </div>
    <div class="header-actions">
      <button
        class="btn btn-secondary"
        on:click={refreshSearchStatus}
        disabled={$searchLoading}
      >
        <RefreshCw class="icon {$searchLoading ? 'spin' : ''}" size={16} />
        Refresh
      </button>
    </div>
  </div>

  {#if $searchError}
    <div class="error-card">
      <AlertTriangle class="icon" size={20} />
      <div class="error-content">
        <h3>Failed to Load Search Status</h3>
        <p>{$searchError}</p>
        <button class="btn btn-primary" on:click={refreshSearchStatus}>
          Try Again
        </button>
      </div>
    </div>
  {/if}

  {#if $searchLoading && !$searchStatus}
    <div class="loading-card">
      <div class="loading-spinner"></div>
      <p>Loading search status...</p>
    </div>
  {/if}

  {#if $searchStatus}
    <!-- Status Overview -->
    <div class="status-overview">
      <div class="status-card">
        <div class="status-header">
          <h3>Index Status</h3>
          <span class={`status-badge ${getStatusBg(healthStatus)}`}>
            {healthStatus.toUpperCase()}
          </span>
        </div>
        <div class="status-metrics">
          <div class="metric">
            <div class="metric-value">{$searchStatus.total_documents?.toLocaleString() || 0}</div>
            <div class="metric-label">Total Documents</div>
          </div>
          <div class="metric">
            <div class="metric-value">{$searchStatus.indexed_documents?.toLocaleString() || 0}</div>
            <div class="metric-label">Indexed Documents</div>
          </div>
          <div class="metric">
            <div class="metric-value">{$searchStatus.pending_documents?.toLocaleString() || 0}</div>
            <div class="metric-label">Pending Documents</div>
          </div>
        </div>
        {#if $searchStatus.indexing}
          <div class="progress-section">
            <div class="progress-header">
              <span>Indexing Progress</span>
              <span>{indexingProgress.toFixed(1)}%</span>
            </div>
            <div class="progress-bar">
              <div class="progress-fill" style="width: {indexingProgress}%"></div>
            </div>
          </div>
        {/if}
      </div>

      <div class="status-card">
        <div class="status-header">
          <h3>Performance</h3>
          <TrendingUp class="icon" size={20} />
        </div>
        <div class="status-metrics">
          <div class="metric">
            <div class="metric-value">{($searchStatus.search_performance?.avg_query_time || 0).toFixed(2)}ms</div>
            <div class="metric-label">Avg Query Time</div>
          </div>
          <div class="metric">
            <div class="metric-value">{($searchStatus.search_performance?.queries_per_second || 0).toFixed(1)}</div>
            <div class="metric-label">Queries/Second</div>
          </div>
          <div class="metric">
            <div class="metric-value">{($searchStatus.search_performance?.cache_hit_rate || 0).toFixed(1)}%</div>
            <div class="metric-label">Cache Hit Rate</div>
          </div>
        </div>
      </div>

      <div class="status-card">
        <div class="status-header">
          <h3>Storage</h3>
          <Database class="icon" size={20} />
        </div>
        <div class="status-metrics">
          <div class="metric">
            <div class="metric-value">{formatBytes($searchStatus.index_size || 0)}</div>
            <div class="metric-label">Index Size</div>
          </div>
          <div class="metric">
            <div class="metric-value">{Math.round(($searchStatus.indexed_documents / 1000000))}M</div>
            <div class="metric-label">Document Count</div>
          </div>
          <div class="metric">
            <div class="metric-value">{($searchStatus.index_size && $searchStatus.indexed_documents) ?
              Math.round($searchStatus.index_size / $searchStatus.indexed_documents) : 0}B</div>
            <div class="metric-label">Avg. Doc Size</div>
          </div>
        </div>
      </div>
    </div>

    <!-- Index Operations -->
    <div class="operations-section">
      <div class="section-header">
        <h2>
          <Settings class="icon" size={20} />
          Index Operations
        </h2>
        <p class="section-description">
          Manage the search index and perform maintenance operations
        </p>
      </div>

      <div class="operations-grid">
        <div class="operation-card">
          <div class="operation-header">
            <h3>Rebuild Index</h3>
            <RotateCcw class="icon" size={20} />
          </div>
          <div class="operation-description">
            <p>Completely rebuild the search index from scratch. This operation may take some time.</p>
          </div>
          <div class="operation-actions">
            <button
              class="btn btn-primary"
              on:click={() => showRebuildConfirm = true}
              disabled={isRebuilding || $searchStatus?.indexing}
            >
              {#if isRebuilding}
                <div class="loading-spinner small"></div>
                Rebuilding...
              {:else if $searchStatus?.indexing}
                <Pause class="icon" size={16} />
                Indexing in Progress
              {:else}
                <RotateCcw class="icon" size={16} />
                Rebuild Index
              {/if}
            </button>
          </div>
        </div>

        <div class="operation-card">
          <div class="operation-header">
            <h3>Optimize Index</h3>
            <TrendingUp class="icon" size={20} />
          </div>
          <div class="operation-description">
            <p>Optimize the search index for better performance and reduced storage usage.</p>
          </div>
          <div class="operation-actions">
            <button
              class="btn btn-secondary"
              on:click={() => showOptimizeConfirm = true}
              disabled={isOptimizing || $searchStatus?.indexing}
            >
              {#if isOptimizing}
                <div class="loading-spinner small"></div>
                Optimizing...
              {:else if $searchStatus?.indexing}
                <Pause class="icon" size={16} />
                Indexing in Progress
              {:else}
                <TrendingUp class="icon" size={16} />
                Optimize Index
              {/if}
            </button>
          </div>
        </div>
      </div>
    </div>

    <!-- Advanced Metrics -->
    <div class="metrics-section">
      <div class="section-header">
        <h2>
          <Clock class="icon" size={20} />
          Detailed Metrics
        </h2>
      </div>

      <div class="metrics-grid">
        <div class="metrics-card">
          <h3>Index Statistics</h3>
          <div class="metrics-list">
            <div class="metrics-item">
              <span class="label">Total Documents</span>
              <span class="value">{$searchStatus.total_documents?.toLocaleString() || 0}</span>
            </div>
            <div class="metrics-item">
              <span class="label">Indexed Documents</span>
              <span class="value">{$searchStatus.indexed_documents?.toLocaleString() || 0}</span>
            </div>
            <div class="metrics-item">
              <span class="label">Pending Documents</span>
              <span class="value">{$searchStatus.pending_documents?.toLocaleString() || 0}</span>
            </div>
            <div class="metrics-item">
              <span class="label">Index Size</span>
              <span class="value">{formatBytes($searchStatus.index_size || 0)}</span>
            </div>
          </div>
        </div>

        <div class="metrics-card">
          <h3>Performance Metrics</h3>
          <div class="metrics-list">
            <div class="metrics-item">
              <span class="label">Average Query Time</span>
              <span class="value">{($searchStatus.search_performance?.avg_query_time || 0).toFixed(2)}ms</span>
            </div>
            <div class="metrics-item">
              <span class="label">Queries per Second</span>
              <span class="value">{($searchStatus.search_performance?.queries_per_second || 0).toFixed(1)}</span>
            </div>
            <div class="metrics-item">
              <span class="label">Cache Hit Rate</span>
              <span class="value">{($searchStatus.search_performance?.cache_hit_rate || 0).toFixed(1)}%</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<!-- Rebuild Confirmation Modal -->
{#if showRebuildConfirm}
  <div class="modal-overlay" on:click={() => showRebuildConfirm = false}>
    <div class="modal-card" on:click|stopPropagation>
      <div class="modal-header">
        <h3>Confirm Index Rebuild</h3>
      </div>
      <div class="modal-body">
        <p>Are you sure you want to rebuild the search index?</p>
        <p class="warning">This operation will:</p>
        <ul>
          <li>Delete the current index</li>
          <li>Re-index all documents from scratch</li>
          <li>Temporarily disable search functionality</li>
          <li>Take significant time to complete</li>
        </ul>
      </div>
      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={() => showRebuildConfirm = false}>
          Cancel
        </button>
        <button
          class="btn btn-danger"
          on:click={rebuildIndex}
          disabled={isRebuilding}
        >
          {#if isRebuilding}
            <div class="loading-spinner small"></div>
            Rebuilding...
          {:else}
            <RotateCcw class="icon" size={16} />
            Rebuild Index
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<!-- Optimize Confirmation Modal -->
{#if showOptimizeConfirm}
  <div class="modal-overlay" on:click={() => showOptimizeConfirm = false}>
    <div class="modal-card" on:click|stopPropagation>
      <div class="modal-header">
        <h3>Confirm Index Optimization</h3>
      </div>
      <div class="modal-body">
        <p>Are you sure you want to optimize the search index?</p>
        <p>This operation will improve search performance and reduce storage size.</p>
      </div>
      <div class="modal-actions">
        <button class="btn btn-secondary" on:click={() => showOptimizeConfirm = false}>
          Cancel
        </button>
        <button
          class="btn btn-primary"
          on:click={optimizeIndex}
          disabled={isOptimizing}
        >
          {#if isOptimizing}
            <div class="loading-spinner small"></div>
            Optimizing...
          {:else}
            <TrendingUp class="icon" size={16} />
            Optimize Index
          {/if}
        </button>
      </div>
    </div>
  </div>
{/if}

<style>
  .search-management {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .header-content {
    flex: 1;
  }

  .page-title {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    font-size: 2rem;
    font-weight: 700;
    color: #1f2937;
    margin: 0;
  }

  .title-icon {
    color: #3b82f6;
  }

  .page-description {
    color: #6b7280;
    margin: 0.5rem 0 0 0;
    font-size: 1rem;
  }

  .header-actions {
    display: flex;
    gap: 0.5rem;
  }

  .error-card {
    display: flex;
    align-items: flex-start;
    gap: 1rem;
    padding: 1.5rem;
    background: #fef2f2;
    border: 1px solid #fecaca;
    border-radius: 8px;
    color: #991b1b;
  }

  .error-card .icon {
    color: #dc2626;
    margin-top: 0.125rem;
  }

  .error-content h3 {
    margin: 0 0 0.5rem 0;
    font-size: 1.125rem;
    font-weight: 600;
  }

  .error-content p {
    margin: 0 0 1rem 0;
  }

  .loading-card {
    display: flex;
    flex-direction: column;
    align-items: center;
    gap: 1rem;
    padding: 3rem 1.5rem;
    text-align: center;
    color: #6b7280;
  }

  .loading-spinner {
    width: 2rem;
    height: 2rem;
    border: 2px solid #e5e7eb;
    border-top: 2px solid #3b82f6;
    border-radius: 50%;
    animation: spin 1s linear infinite;
  }

  .loading-spinner.small {
    width: 1rem;
    height: 1rem;
    border-width: 1.5px;
  }

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  .status-overview {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1rem;
  }

  .status-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    border: 1px solid #e5e7eb;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .status-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .status-header h3 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: #1f2937;
  }

  .status-header .icon {
    color: #6b7280;
  }

  .status-badge {
    padding: 0.25rem 0.5rem;
    border-radius: 9999px;
    font-size: 0.75rem;
    font-weight: 500;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .status-metrics {
    display: grid;
    grid-template-columns: repeat(3, 1fr);
    gap: 1rem;
  }

  .metric {
    text-align: center;
  }

  .metric-value {
    font-size: 1.5rem;
    font-weight: 700;
    color: #1f2937;
    line-height: 1;
  }

  .metric-label {
    font-size: 0.875rem;
    color: #6b7280;
    margin-top: 0.25rem;
  }

  .progress-section {
    margin-top: 1rem;
    padding-top: 1rem;
    border-top: 1px solid #e5e7eb;
  }

  .progress-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 0.5rem;
    font-size: 0.875rem;
    font-weight: 500;
    color: #374151;
  }

  .progress-bar {
    width: 100%;
    height: 6px;
    background: #e5e7eb;
    border-radius: 3px;
    overflow: hidden;
  }

  .progress-fill {
    height: 100%;
    background: linear-gradient(90deg, #3b82f6, #2563eb);
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .operations-section, .metrics-section {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    border: 1px solid #e5e7eb;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .section-header {
    margin-bottom: 1.5rem;
  }

  .section-header h2 {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    margin: 0 0 0.5rem 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #1f2937;
  }

  .section-header .icon {
    color: #6b7280;
  }

  .section-description {
    margin: 0;
    color: #6b7280;
    font-size: 0.875rem;
  }

  .operations-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1rem;
  }

  .operation-card {
    padding: 1.5rem;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: #fafafa;
  }

  .operation-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .operation-header h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: #1f2937;
  }

  .operation-header .icon {
    color: #6b7280;
  }

  .operation-description {
    margin-bottom: 1rem;
  }

  .operation-description p {
    margin: 0;
    color: #6b7280;
    font-size: 0.875rem;
    line-height: 1.5;
  }

  .operation-actions {
    display: flex;
    gap: 0.5rem;
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1rem;
  }

  .metrics-card {
    padding: 1rem;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    background: #fafafa;
  }

  .metrics-card h3 {
    margin: 0 0 1rem 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: #374151;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .metrics-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .metrics-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
  }

  .metrics-item .label {
    font-size: 0.875rem;
    color: #6b7280;
  }

  .metrics-item .value {
    font-size: 0.875rem;
    font-weight: 500;
    color: #1f2937;
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

  .btn:disabled {
    opacity: 0.5;
    cursor: not-allowed;
  }

  .btn-primary {
    background-color: #3b82f6;
    color: white;
    border-color: #3b82f6;
  }

  .btn-primary:hover:not(:disabled) {
    background-color: #2563eb;
    border-color: #2563eb;
  }

  .btn-secondary {
    background-color: white;
    color: #374151;
    border-color: #d1d5db;
  }

  .btn-secondary:hover:not(:disabled) {
    background-color: #f9fafb;
    border-color: #9ca3af;
  }

  .btn-danger {
    background-color: #dc2626;
    color: white;
    border-color: #dc2626;
  }

  .btn-danger:hover:not(:disabled) {
    background-color: #b91c1c;
    border-color: #b91c1c;
  }

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

  .modal-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    max-width: 500px;
    width: 90%;
    max-height: 90vh;
    overflow-y: auto;
    box-shadow: 0 20px 25px -5px rgba(0, 0, 0, 0.1);
  }

  .modal-header {
    margin-bottom: 1rem;
  }

  .modal-header h3 {
    margin: 0;
    font-size: 1.125rem;
    font-weight: 600;
    color: #1f2937;
  }

  .modal-body {
    margin-bottom: 1.5rem;
  }

  .modal-body p {
    margin: 0 0 1rem 0;
    color: #374151;
  }

  .warning {
    color: #dc2626;
    font-weight: 500;
  }

  .modal-body ul {
    margin: 0;
    padding-left: 1.5rem;
    color: #6b7280;
  }

  .modal-body li {
    margin-bottom: 0.25rem;
  }

  .modal-actions {
    display: flex;
    gap: 0.5rem;
    justify-content: flex-end;
  }

  /* Responsive design */
  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      gap: 1rem;
      align-items: stretch;
    }

    .status-overview {
      grid-template-columns: 1fr;
    }

    .status-metrics {
      grid-template-columns: 1fr;
      gap: 0.75rem;
    }

    .operations-grid {
      grid-template-columns: 1fr;
    }

    .metrics-grid {
      grid-template-columns: 1fr;
    }
  }
</style>

<script context="module">
  function formatBytes(bytes) {
    if (bytes === 0) return '0 B';

    const k = 1024;
    const sizes = ['B', 'KB', 'MB', 'GB', 'TB'];
    const i = Math.floor(Math.log(bytes) / Math.log(k));

    return parseFloat((bytes / Math.pow(k, i)).toFixed(1)) + ' ' + sizes[i];
  }
</script>