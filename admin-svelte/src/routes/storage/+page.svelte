<script lang="ts">
  import { onMount, onDestroy } from 'svelte';
  import { storageStats, storageLoading, storageError, refreshStorageStats, systemStore, systemMetrics } from '$lib/stores/system';
  import { Database, HardDrive, Activity, TrendingUp, TrendingDown, RefreshCw, AlertTriangle, Search, BarChart3, Clock } from 'lucide-svelte';
  import { formatBytes, formatDuration } from '$lib/api/client';

  let selectedTimeRange = '1h'; // 1h, 6h, 24h, 7d
  let autoRefresh = true;
  let refreshInterval;

  // Time range options
  const timeRanges = [
    { value: '1h', label: '1 Hour' },
    { value: '6h', label: '6 Hours' },
    { value: '24h', label: '24 Hours' },
    { value: '7d', label: '7 Days' },
  ];

  // Auto-refresh management
  function toggleAutoRefresh() {
    autoRefresh = !autoRefresh;
    if (autoRefresh) {
      startAutoRefresh();
    } else {
      stopAutoRefresh();
    }
  }

  function startAutoRefresh() {
    stopAutoRefresh();
    refreshInterval = setInterval(() => {
      refreshStorageStats();
    }, 30000); // Refresh every 30 seconds
  }

  function stopAutoRefresh() {
    if (refreshInterval) {
      clearInterval(refreshInterval);
      refreshInterval = null;
    }
  }

  onMount(() => {
    refreshStorageStats();
    if (autoRefresh) {
      startAutoRefresh();
    }
  });

  onDestroy(() => {
    stopAutoRefresh();
  });

  // Helper functions for calculations
  function getStorageUsageColor(percentage: number) {
    if (percentage >= 90) return 'text-red-600';
    if (percentage >= 75) return 'text-yellow-600';
    return 'text-green-600';
  }

  function getPerformanceColor(avgTime: number) {
    if (avgTime >= 1000) return 'text-red-600';
    if (avgTime >= 500) return 'text-yellow-600';
    return 'text-green-600';
  }

  function getCacheHitRateColor(rate: number) {
    if (rate >= 90) return 'text-green-600';
    if (rate >= 70) return 'text-yellow-600';
    return 'text-red-600';
  }

  // Simulate historical data for charts
  let historicalData = {
    queries: Array.from({ length: 24 }, (_, i) => ({
      time: new Date(Date.now() - (23 - i) * 3600000).toISOString(),
      value: Math.floor(Math.random() * 1000) + 500,
    })),
    responseTime: Array.from({ length: 24 }, (_, i) => ({
      time: new Date(Date.now() - (23 - i) * 3600000).toISOString(),
      value: Math.random() * 200 + 50,
    })),
    storage: Array.from({ length: 24 }, (_, i) => ({
      time: new Date(Date.now() - (23 - i) * 3600000).toISOString(),
      value: Math.floor(Math.random() * 10000000000) + 5000000000,
    })),
  };

  // Reactive calculations
  $: storageEfficiency = $storageStats?.cache_size && $storageStats?.storage_size
    ? ($storageStats.cache_size / $storageStats.storage_size) * 100
    : 0;

  $: avgRecordsPerTable = $storageStats?.tables && $storageStats?.total_records
    ? Math.round($storageStats.total_records / $storageStats.tables)
    : 0;

  $: healthScore = calculateHealthScore();

  function calculateHealthScore(): number {
    if (!$storageStats) return 0;

    const factors = [
      { weight: 0.3, value: Math.max(0, 100 - ($storageStats.query_performance.avg_query_time / 10)) }, // Response time
      { weight: 0.3, value: $storageStats.query_performance.cache_hit_rate }, // Cache hit rate
      { weight: 0.2, value: Math.max(0, 100 - ($storageStats.query_performance.slow_queries / 10)) }, // Slow queries
      { weight: 0.2, value: 100 - (Math.min($systemMetrics?.disk_usage.percentage || 0, 100)) }, // Disk usage
    ];

    return factors.reduce((score, factor) => score + (factor.value * factor.weight), 0);
  }
</script>

<svelte:head>
  <title>Storage Monitoring - MetaBase Admin</title>
</svelte:head>

<div class="storage-monitoring">
  <div class="page-header">
    <div class="header-content">
      <h1 class="page-title">
        <Database class="title-icon" size={28} />
        Storage Monitoring
      </h1>
      <p class="page-description">
        Monitor database performance, storage usage, and query analytics
      </p>
    </div>
    <div class="header-controls">
      <div class="time-range-selector">
        <select bind:value={selectedTimeRange} class="select">
          {#each timeRanges as range}
            <option value={range.value}>{range.label}</option>
          {/each}
        </select>
      </div>
      <button
        class="btn btn-secondary"
        class:active={autoRefresh}
        on:click={toggleAutoRefresh}
        title={autoRefresh ? 'Disable auto-refresh' : 'Enable auto-refresh'}
      >
        <RefreshCw class="icon" size={16} class:spin={autoRefresh} />
        Auto Refresh
      </button>
      <button
        class="btn btn-primary"
        on:click={refreshStorageStats}
        disabled={$storageLoading}
      >
        <RefreshCw class="icon" size={16} class:spin={$storageLoading} />
        Refresh
      </button>
    </div>
  </div>

  {#if $storageError}
    <div class="error-card">
      <AlertTriangle class="icon" size={20} />
      <div class="error-content">
        <h3>Failed to Load Storage Statistics</h3>
        <p>{$storageError}</p>
        <button class="btn btn-primary" on:click={refreshStorageStats}>
          Try Again
        </button>
      </div>
    </div>
  {/if}

  {#if $storageLoading && !$storageStats}
    <div class="loading-card">
      <div class="loading-spinner"></div>
      <p>Loading storage statistics...</p>
    </div>
  {/if}

  {#if $storageStats}
    <!-- Overview Cards -->
    <div class="overview-grid">
      <div class="overview-card health">
        <div class="card-header">
          <div class="card-title">
            <Activity class="icon" size={20} />
            Health Score
          </div>
          <div class="card-value {getPerformanceColor($storageStats.query_performance.avg_query_time)}">
            {healthScore.toFixed(0)}%
          </div>
        </div>
        <div class="health-indicator">
          <div class="health-bar">
            <div class="health-fill" style="width: {healthScore}%"></div>
          </div>
          <div class="health-label">
            {healthScore >= 80 ? 'Excellent' : healthScore >= 60 ? 'Good' : healthScore >= 40 ? 'Fair' : 'Poor'}
          </div>
        </div>
      </div>

      <div class="overview-card">
        <div class="card-header">
          <div class="card-title">
            <Database class="icon" size={20} />
            Databases
          </div>
          <div class="card-value">
            {$storageStats.databases}
          </div>
        </div>
        <div class="card-subtitle">
          {$storageStats.tables} tables • {avgRecordsPerTable.toLocaleString()} avg records/table
        </div>
      </div>

      <div class="overview-card">
        <div class="card-header">
          <div class="card-title">
            <HardDrive class="icon" size={20} />
            Storage
          </div>
          <div class="card-value {getStorageUsageColor($systemMetrics?.disk_usage.percentage || 0)}">
            {formatBytes($storageStats.storage_size)}
          </div>
        </div>
        <div class="card-subtitle">
          Cache: {formatBytes($storageStats.cache_size)} ({storageEfficiency.toFixed(1)}%)
        </div>
      </div>

      <div class="overview-card">
        <div class="card-header">
          <div class="card-title">
            <Search class="icon" size={20} />
            Total Records
          </div>
          <div class="card-value">
            {($storageStats.total_records / 1000000).toFixed(1)}M
          </div>
        </div>
        <div class="card-subtitle">
          Across {$storageStats.tables} tables
        </div>
      </div>
    </div>

    <!-- Performance Metrics -->
    <div class="metrics-section">
      <div class="section-header">
        <h2>
          <BarChart3 class="icon" size={20} />
          Performance Metrics
        </h2>
      </div>

      <div class="metrics-grid">
        <div class="metric-card">
          <div class="metric-header">
            <h3>Query Performance</h3>
            <Clock class="icon" size={20} />
          </div>
          <div class="metric-stats">
            <div class="stat-item">
              <div class="stat-label">Average Query Time</div>
              <div class="stat-value {getPerformanceColor($storageStats.query_performance.avg_query_time)}">
                {$storageStats.query_performance.avg_query_time.toFixed(2)}ms
              </div>
              <div class="stat-trend">
                {#if $storageStats.query_performance.avg_query_time < 100}
                  <TrendingDown class="trend-icon down" size={16} />
                  <span class="trend-text good">Good</span>
                {:else if $storageStats.query_performance.avg_query_time < 500}
                  <TrendingUp class="trend-icon" size={16} />
                  <span class="trend-text warning">Fair</span>
                {:else}
                  <TrendingUp class="trend-icon up" size={16} />
                  <span class="trend-text bad">Slow</span>
                {/if}
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Slow Queries</div>
              <div class="stat-value {$storageStats.query_performance.slow_queries > 10 ? 'text-red-600' : 'text-green-600'}">
                {$storageStats.query_performance.slow_queries}
              </div>
              <div class="stat-trend">
                <span class="trend-text">Last hour</span>
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Cache Hit Rate</div>
              <div class="stat-value {getCacheHitRateColor($storageStats.query_performance.cache_hit_rate)}">
                {$storageStats.query_performance.cache_hit_rate.toFixed(1)}%
              </div>
              <div class="stat-trend">
                {#if $storageStats.query_performance.cache_hit_rate >= 90}
                  <TrendingUp class="trend-icon up" size={16} />
                  <span class="trend-text good">Excellent</span>
                {:else if $storageStats.query_performance.cache_hit_rate >= 70}
                  <TrendingUp class="trend-icon" size={16} />
                  <span class="trend-text warning">Good</span>
                {:else}
                  <TrendingDown class="trend-icon down" size={16} />
                  <span class="trend-text bad">Poor</span>
                {/if}
              </div>
            </div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <h3>Storage Details</h3>
            <HardDrive class="icon" size={20} />
          </div>
          <div class="metric-stats">
            <div class="stat-item">
              <div class="stat-label">Total Storage</div>
              <div class="stat-value">
                {formatBytes($storageStats.storage_size)}
              </div>
              <div class="stat-trend">
                {formatBytes($storageStats.cache_size)} cache
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Cache Efficiency</div>
              <div class="stat-value {storageEfficiency >= 20 ? 'text-green-600' : 'text-yellow-600'}">
                {storageEfficiency.toFixed(1)}%
              </div>
              <div class="stat-trend">
                <span class="trend-text">Cache/Storage ratio</span>
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Disk Usage</div>
              <div class="stat-value {getStorageUsageColor($systemMetrics?.disk_usage.percentage || 0)}">
                {($systemMetrics?.disk_usage.percentage || 0).toFixed(1)}%
              </div>
              <div class="stat-trend">
                {formatBytes($systemMetrics?.disk_usage.used || 0)} / {formatBytes($systemMetrics?.disk_usage.total || 0)}
              </div>
            </div>
          </div>
        </div>

        <div class="metric-card">
          <div class="metric-header">
            <h3>Database Stats</h3>
            <Database class="icon" size={20} />
          </div>
          <div class="metric-stats">
            <div class="stat-item">
              <div class="stat-label">Databases</div>
              <div class="stat-value">
                {$storageStats.databases}
              </div>
              <div class="stat-trend">
                <span class="trend-text">Active databases</span>
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Tables</div>
              <div class="stat-value">
                {$storageStats.tables}
              </div>
              <div class="stat-trend">
                <span class="trend-text">Total tables</span>
              </div>
            </div>

            <div class="stat-item">
              <div class="stat-label">Records per Table</div>
              <div class="stat-value">
                {avgRecordsPerTable.toLocaleString()}
              </div>
              <div class="stat-trend">
                <span class="trend-text">Average</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Charts Section -->
    <div class="charts-section">
      <div class="section-header">
        <h2>
          <BarChart3 class="icon" size={20} />
          Performance Trends
        </h2>
      </div>

      <div class="charts-grid">
        <div class="chart-card">
          <div class="chart-header">
            <h3>Query Volume</h3>
            <div class="chart-legend">
              <span class="legend-item">
                <span class="legend-color blue"></span>
                Queries per hour
              </span>
            </div>
          </div>
          <div class="chart-container">
            <div class="chart-placeholder">
              <div class="mock-chart">
                {#each historicalData.queries.slice(-12) as point}
                  <div
                    class="chart-bar"
                    style="height: {(point.value / 1500) * 100}%; left: {((historicalData.queries.indexOf(point) % 12) / 12) * 100}%"
                    title="{new Date(point.time).toLocaleTimeString()}: {point.value} queries"
                  ></div>
                {/each}
              </div>
              <div class="chart-labels">
                <span>12h ago</span>
                <span>6h ago</span>
                <span>Now</span>
              </div>
            </div>
          </div>
        </div>

        <div class="chart-card">
          <div class="chart-header">
            <h3>Response Time</h3>
            <div class="chart-legend">
              <span class="legend-item">
                <span class="legend-color green"></span>
                Avg response time (ms)
              </span>
            </div>
          </div>
          <div class="chart-container">
            <div class="chart-placeholder">
              <div class="mock-chart line">
                {#each historicalData.responseTime.slice(-12) as point, i}
                  <div
                    class="chart-point"
                    style="bottom: {(point.value / 250) * 100}%; left: {(i / 11) * 100}%"
                    title="{new Date(point.time).toLocaleTimeString()}: {point.value.toFixed(1)}ms"
                  ></div>
                {/each}
                <svg class="chart-line" viewBox="0 0 100 100">
                  <polyline
                    points={historicalData.responseTime.slice(-12).map((point, i) => `${(i / 11) * 100},${100 - (point.value / 250) * 100}`).join(' ')}
                    fill="none"
                    stroke="#10b981"
                    stroke-width="2"
                  />
                </svg>
              </div>
              <div class="chart-labels">
                <span>12h ago</span>
                <span>6h ago</span>
                <span>Now</span>
              </div>
            </div>
          </div>
        </div>

        <div class="chart-card">
          <div class="chart-header">
            <h3>Storage Growth</h3>
            <div class="chart-legend">
              <span class="legend-item">
                <span class="legend-color purple"></span>
                Storage size (GB)
              </span>
            </div>
          </div>
          <div class="chart-container">
            <div class="chart-placeholder">
              <div class="mock-chart">
                {#each historicalData.storage.slice(-12) as point}
                  <div
                    class="chart-bar"
                    style="height: {(point.value / 20000000000) * 100}%; left: {((historicalData.storage.indexOf(point) % 12) / 12) * 100}%; background: #8b5cf6;"
                    title="{new Date(point.time).toLocaleTimeString()}: {formatBytes(point.value)}"
                  ></div>
                {/each}
              </div>
              <div class="chart-labels">
                <span>12h ago</span>
                <span>6h ago</span>
                <span>Now</span>
              </div>
            </div>
          </div>
        </div>
      </div>
    </div>

    <!-- Detailed Statistics -->
    <div class="details-section">
      <div class="section-header">
        <h2>
          <Database class="icon" size={20} />
          Detailed Statistics
        </h2>
      </div>

      <div class="details-grid">
        <div class="details-card">
          <h3>Performance Indicators</h3>
          <div class="details-list">
            <div class="detail-item">
              <span class="detail-label">Average Query Time</span>
              <span class="detail-value {getPerformanceColor($storageStats.query_performance.avg_query_time)}">
                {$storageStats.query_performance.avg_query_time.toFixed(2)}ms
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Slow Queries (Last Hour)</span>
              <span class="detail-value {$storageStats.query_performance.slow_queries > 10 ? 'text-red-600' : 'text-green-600'}">
                {$storageStats.query_performance.slow_queries}
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Cache Hit Rate</span>
              <span class="detail-value {getCacheHitRateColor($storageStats.query_performance.cache_hit_rate)}">
                {$storageStats.query_performance.cache_hit_rate.toFixed(1)}%
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Health Score</span>
              <span class="detail-value {healthScore >= 80 ? 'text-green-600' : healthScore >= 60 ? 'text-yellow-600' : 'text-red-600'}">
                {healthScore.toFixed(0)}%
              </span>
            </div>
          </div>
        </div>

        <div class="details-card">
          <h3>Storage Information</h3>
          <div class="details-list">
            <div class="detail-item">
              <span class="detail-label">Total Storage Size</span>
              <span class="detail-value">{formatBytes($storageStats.storage_size)}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Cache Size</span>
              <span class="detail-value">{formatBytes($storageStats.cache_size)}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Cache Efficiency</span>
              <span class="detail-value {storageEfficiency >= 20 ? 'text-green-600' : 'text-yellow-600'}">
                {storageEfficiency.toFixed(1)}%
              </span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Disk Usage</span>
              <span class="detail-value {getStorageUsageColor($systemMetrics?.disk_usage.percentage || 0)}">
                {($systemMetrics?.disk_usage.percentage || 0).toFixed(1)}%
              </span>
            </div>
          </div>
        </div>

        <div class="details-card">
          <h3>Database Statistics</h3>
          <div class="details-list">
            <div class="detail-item">
              <span class="detail-label">Total Databases</span>
              <span class="detail-value">{$storageStats.databases}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Total Tables</span>
              <span class="detail-value">{$storageStats.tables}</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Total Records</span>
              <span class="detail-value">{($storageStats.total_records / 1000000).toFixed(1)}M</span>
            </div>
            <div class="detail-item">
              <span class="detail-label">Avg Records per Table</span>
              <span class="detail-value">{avgRecordsPerTable.toLocaleString()}</span>
            </div>
          </div>
        </div>
      </div>
    </div>
  {/if}
</div>

<style>
  .storage-monitoring {
    display: flex;
    flex-direction: column;
    gap: 1.5rem;
  }

  .page-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    flex-wrap: wrap;
    gap: 1rem;
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

  .header-controls {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    flex-wrap: wrap;
  }

  .time-range-selector .select {
    padding: 0.5rem 0.75rem;
    border: 1px solid #d1d5db;
    border-radius: 6px;
    background: white;
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

  .btn-secondary.active {
    background-color: #3b82f6;
    color: white;
    border-color: #3b82f6;
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

  @keyframes spin {
    0% { transform: rotate(0deg); }
    100% { transform: rotate(360deg); }
  }

  .spin {
    animation: spin 1s linear infinite;
  }

  .overview-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(280px, 1fr));
    gap: 1rem;
  }

  .overview-card {
    background: white;
    border-radius: 8px;
    padding: 1.5rem;
    border: 1px solid #e5e7eb;
    box-shadow: 0 1px 3px rgba(0, 0, 0, 0.1);
  }

  .overview-card.health {
    background: linear-gradient(135deg, #3b82f6 0%, #2563eb 100%);
    color: white;
    border: none;
  }

  .card-header {
    display: flex;
    justify-content: space-between;
    align-items: flex-start;
    margin-bottom: 1rem;
  }

  .card-title {
    display: flex;
    align-items: center;
    gap: 0.5rem;
    font-size: 0.875rem;
    font-weight: 600;
    color: #6b7280;
  }

  .overview-card.health .card-title {
    color: rgba(255, 255, 255, 0.9);
  }

  .card-title .icon {
    color: #6b7280;
  }

  .overview-card.health .card-title .icon {
    color: rgba(255, 255, 255, 0.9);
  }

  .card-value {
    font-size: 2rem;
    font-weight: 700;
    line-height: 1;
    color: #1f2937;
  }

  .overview-card.health .card-value {
    color: white;
  }

  .card-subtitle {
    font-size: 0.875rem;
    color: #6b7280;
    margin-top: 0.5rem;
  }

  .overview-card.health .card-subtitle {
    color: rgba(255, 255, 255, 0.8);
  }

  .health-indicator {
    margin-top: 1rem;
  }

  .health-bar {
    width: 100%;
    height: 6px;
    background: rgba(255, 255, 255, 0.3);
    border-radius: 3px;
    overflow: hidden;
    margin-bottom: 0.5rem;
  }

  .health-fill {
    height: 100%;
    background: white;
    border-radius: 3px;
    transition: width 0.3s ease;
  }

  .health-label {
    font-size: 0.875rem;
    font-weight: 500;
    text-align: center;
    color: rgba(255, 255, 255, 0.9);
  }

  .metrics-section, .charts-section, .details-section {
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
    margin: 0;
    font-size: 1.25rem;
    font-weight: 600;
    color: #1f2937;
  }

  .section-header .icon {
    color: #6b7280;
  }

  .metrics-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(320px, 1fr));
    gap: 1rem;
  }

  .metric-card {
    padding: 1.5rem;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: #fafafa;
  }

  .metric-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1.5rem;
  }

  .metric-header h3 {
    margin: 0;
    font-size: 1rem;
    font-weight: 600;
    color: #1f2937;
  }

  .metric-header .icon {
    color: #6b7280;
  }

  .metric-stats {
    display: flex;
    flex-direction: column;
    gap: 1rem;
  }

  .stat-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.75rem 0;
    border-bottom: 1px solid #e5e7eb;
  }

  .stat-item:last-child {
    border-bottom: none;
  }

  .stat-label {
    font-size: 0.875rem;
    color: #6b7280;
    font-weight: 500;
  }

  .stat-value {
    font-size: 1rem;
    font-weight: 600;
    color: #1f2937;
  }

  .stat-trend {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
  }

  .trend-icon {
    color: #6b7280;
  }

  .trend-icon.up {
    color: #10b981;
  }

  .trend-icon.down {
    color: #dc2626;
  }

  .trend-text.good {
    color: #10b981;
  }

  .trend-text.warning {
    color: #f59e0b;
  }

  .trend-text.bad {
    color: #dc2626;
  }

  .charts-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(350px, 1fr));
    gap: 1rem;
  }

  .chart-card {
    padding: 1rem;
    border: 1px solid #e5e7eb;
    border-radius: 8px;
    background: #fafafa;
  }

  .chart-header {
    display: flex;
    justify-content: space-between;
    align-items: center;
    margin-bottom: 1rem;
  }

  .chart-header h3 {
    margin: 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: #1f2937;
  }

  .chart-legend {
    display: flex;
    gap: 1rem;
  }

  .legend-item {
    display: flex;
    align-items: center;
    gap: 0.25rem;
    font-size: 0.75rem;
    color: #6b7280;
  }

  .legend-color {
    width: 12px;
    height: 12px;
    border-radius: 2px;
  }

  .legend-color.blue {
    background: #3b82f6;
  }

  .legend-color.green {
    background: #10b981;
  }

  .legend-color.purple {
    background: #8b5cf6;
  }

  .chart-container {
    height: 120px;
    position: relative;
  }

  .chart-placeholder {
    width: 100%;
    height: 100%;
    position: relative;
  }

  .mock-chart {
    width: 100%;
    height: 80px;
    position: relative;
    display: flex;
    align-items: flex-end;
    gap: 2px;
  }

  .mock-chart.line {
    height: 80px;
    align-items: center;
    justify-content: center;
  }

  .chart-bar {
    width: calc(8.33% - 2px);
    background: #3b82f6;
    border-radius: 2px 2px 0 0;
    position: absolute;
    bottom: 40px;
    min-height: 4px;
  }

  .chart-point {
    width: 6px;
    height: 6px;
    background: #10b981;
    border-radius: 50%;
    position: absolute;
  }

  .chart-line {
    position: absolute;
    width: 100%;
    height: 80px;
    top: 0;
    left: 0;
    pointer-events: none;
  }

  .chart-labels {
    display: flex;
    justify-content: space-between;
    margin-top: 0.5rem;
    font-size: 0.75rem;
    color: #6b7280;
  }

  .details-grid {
    display: grid;
    grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
    gap: 1rem;
  }

  .details-card {
    padding: 1rem;
    border: 1px solid #e5e7eb;
    border-radius: 6px;
    background: #fafafa;
  }

  .details-card h3 {
    margin: 0 0 1rem 0;
    font-size: 0.875rem;
    font-weight: 600;
    color: #374151;
    text-transform: uppercase;
    letter-spacing: 0.05em;
  }

  .details-list {
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
  }

  .detail-item {
    display: flex;
    justify-content: space-between;
    align-items: center;
    padding: 0.5rem 0;
  }

  .detail-label {
    font-size: 0.875rem;
    color: #6b7280;
  }

  .detail-value {
    font-size: 0.875rem;
    font-weight: 500;
    color: #1f2937;
  }

  /* Color utilities */
  .text-green-600 { color: #059669; }
  .text-yellow-600 { color: #d97706; }
  .text-red-600 { color: #dc2626; }

  /* Responsive design */
  @media (max-width: 768px) {
    .page-header {
      flex-direction: column;
      align-items: stretch;
    }

    .header-controls {
      justify-content: flex-start;
    }

    .overview-grid {
      grid-template-columns: 1fr;
    }

    .metrics-grid {
      grid-template-columns: 1fr;
    }

    .charts-grid {
      grid-template-columns: 1fr;
    }

    .details-grid {
      grid-template-columns: 1fr;
    }
  }
</style>