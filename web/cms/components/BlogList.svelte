<script lang="ts">
  import { onMount } from 'svelte';
  import BlogPostCard from './BlogPostCard.svelte';
  import { CMSApiClient } from '../utils/api';
  import type { Content, Category, Tag } from '../types/cms';

  export let contentTypeSlug = 'blog-posts';
  export let limit = 12;
  export let featured = false;
  export let sticky = false;
  export let categorySlug: string | undefined = undefined;
  export let tagSlug: string | undefined = undefined;
  export let authorId: string | undefined = undefined;

  let posts: Content[] = [];
  let loading = true;
  let error: string | null = null;
  let total = 0;
  let offset = 0;
  let hasMore = true;

  // Filters
  let selectedCategory: Category | undefined;
  let selectedTag: Tag | undefined;
  let categories: Category[] = [];
  let tags: Tag[] = [];
  let searchQuery = '';

  // Sorting
  let sortBy = 'published_at';
  let sortOrder = 'desc';

  // View mode
  let viewMode: 'grid' | 'list' = 'grid';

  const api = new CMSApiClient({
    tenant_id: '00000000-0000-0000-0000-000000000001', // Will be replaced with actual config
    api_base_url: 'http://localhost:7609',
    api_key: 'demo-key',
    routing: {
      base_path: '',
      content_patterns: {},
    },
    features: {
      comments: true,
      ratings: true,
      search: true,
      categories: true,
      tags: true,
      media_library: true,
      seo: true,
    },
    theme: {
      primary_color: '#3b82f6',
      secondary_color: '#64748b',
      font_family: 'system-ui',
      header_style: 'default',
    },
  });

  onMount(async () => {
    await loadInitialData();
  });

  async function loadInitialData() {
    try {
      loading = true;

      // Load categories and tags
      await Promise.all([
        loadCategories(),
        loadTags(),
      ]);

      // Load initial posts
      await loadPosts();

    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load content';
    } finally {
      loading = false;
    }
  }

  async function loadCategories() {
    try {
      const contentType = await api.getContentType(contentTypeSlug);
      categories = await api.getCategories(contentType.id);
    } catch (err) {
      console.error('Failed to load categories:', err);
    }
  }

  async function loadTags() {
    try {
      tags = await api.getTags(true); // Get popular tags
    } catch (err) {
      console.error('Failed to load tags:', err);
    }
  }

  async function loadPosts(reset = false) {
    try {
      if (reset) {
        offset = 0;
        posts = [];
      }

      const filters = {
        content_type_id: contentTypeSlug,
        limit,
        offset,
        sort_by: sortBy,
        sort_order: sortOrder as 'asc' | 'desc',
        featured,
        sticky,
        category_id: selectedCategory?.id || categorySlug,
        tag_id: selectedTag?.id || tagSlug,
        author_id: authorId,
        query: searchQuery || undefined,
      };

      const response = await api.getContent(filters);

      if (reset) {
        posts = response.items;
      } else {
        posts = [...posts, ...response.items];
      }

      total = response.total;
      hasMore = response.has_next;
      offset += limit;

    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load posts';
      throw err;
    }
  }

  async function loadMore() {
    if (!hasMore || loading) return;

    try {
      loading = true;
      await loadPosts(false);
    } catch (err) {
      // Error already set in loadPosts
    } finally {
      loading = false;
    }
  }

  function handleCategorySelect(category: Category | undefined) {
    selectedCategory = category;
    loadPosts(true);
  }

  function handleTagSelect(tag: Tag | undefined) {
    selectedTag = tag;
    loadPosts(true);
  }

  function handleSearch() {
    loadPosts(true);
  }

  function handleSortChange(event: Event) {
    const select = event.target as HTMLSelectElement;
    [sortBy, sortOrder] = select.value.split('-');
    loadPosts(true);
  }

  function toggleViewMode() {
    viewMode = viewMode === 'grid' ? 'list' : 'grid';
  }
</script>

<div class="blog-list-container">
  <!-- Header -->
  <header class="blog-header">
    <h1 class="blog-title">
      {featured ? 'Featured Posts' : sticky ? 'Pinned Posts' : 'Blog'}
    </h1>

    <div class="header-actions">
      <button
        type="button"
        class="view-toggle"
        on:click={toggleViewMode}
        title="Toggle view mode"
      >
        {#if viewMode === 'grid'}
          📋 List View
        {:else}
          ⊞ Grid View
        {/if}
      </button>
    </div>
  </header>

  <!-- Filters and Search -->
  <section class="filters-section">
    <div class="filters-grid">
      <!-- Search -->
      <div class="search-box">
        <input
          type="text"
          bind:value={searchQuery}
          placeholder="Search posts..."
          class="search-input"
          on:keyup={(e) => {
            if (e.key === 'Enter') {
              handleSearch();
            }
          }}
        />
        <button type="button" class="search-btn" on:click={handleSearch}>
          🔍
        </button>
      </div>

      <!-- Category Filter -->
      <div class="filter-group">
        <label for="category-select" class="filter-label">Category</label>
        <select
          id="category-select"
          class="filter-select"
          on:change={(e) => {
            const categoryId = (e.target as HTMLSelectElement).value;
            const category = categoryId ? categories.find(c => c.id === categoryId) : undefined;
            handleCategorySelect(category);
          }}
        >
          <option value="">All Categories</option>
          {#each categories as category}
            <option value={category.id}>{category.name}</option>
          {/each}
        </select>
      </div>

      <!-- Tag Filter -->
      <div class="filter-group">
        <label for="tag-select" class="filter-label">Tag</label>
        <select
          id="tag-select"
          class="filter-select"
          on:change={(e) => {
            const tagId = (e.target as HTMLSelectElement).value;
            const tag = tagId ? tags.find(t => t.id === tagId) : undefined;
            handleTagSelect(tag);
          }}
        >
          <option value="">All Tags</option>
          {#each tags as tag}
            <option value={tag.id}>#{tag.name}</option>
          {/each}
        </select>
      </div>

      <!-- Sort -->
      <div class="filter-group">
        <label for="sort-select" class="filter-label">Sort By</label>
        <select
          id="sort-select"
          class="filter-select"
          on:change={handleSortChange}
        >
          <option value="published_at-desc">Newest First</option>
          <option value="published_at-asc">Oldest First</option>
          <option value="title-asc">Title A-Z</option>
          <option value="title-desc">Title Z-A</option>
          <option value="view_count-desc">Most Viewed</option>
          <option value="like_count-desc">Most Liked</option>
        </select>
      </div>
    </div>

    <!-- Active Filters -->
    {#if selectedCategory || selectedTag || searchQuery}
      <div class="active-filters">
        <span class="filter-label">Active filters:</span>

        {#if selectedCategory}
          <button
            type="button"
            class="filter-tag"
            on:click={() => handleCategorySelect(undefined)}
          >
            {selectedCategory.name} ✕
          </button>
        {/if}

        {#if selectedTag}
          <button
            type="button"
            class="filter-tag"
            on:click={() => handleTagSelect(undefined)}
          >
            #{selectedTag.name} ✕
          </button>
        {/if}

        {#if searchQuery}
          <button
            type="button"
            class="filter-tag"
            on:click={() => {
              searchQuery = '';
              handleSearch();
            }}
          >
            "{searchQuery}" ✕
          </button>
        {/if}
      </div>
    {/if}
  </section>

  <!-- Loading State -->
  {#if loading && posts.length === 0}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading posts...</p>
    </div>
  {:else if error}
    <div class="error-state">
      <p class="error-message">{error}</p>
      <button type="button" class="retry-btn" on:click={loadInitialData}>
        Try Again
      </button>
    </div>
  {:else if posts.length === 0}
    <div class="empty-state">
      <p>No posts found.</p>
      {#if searchQuery || selectedCategory || selectedTag}
        <button
          type="button"
          class="clear-filters-btn"
          on:click={() => {
            searchQuery = '';
            selectedCategory = undefined;
            selectedTag = undefined;
            loadPosts(true);
          }}
        >
          Clear all filters
        </button>
      {/if}
    </div>
  {:else}
    <!-- Posts Grid/List -->
    <section class="posts-section">
      <div class="posts-container {viewMode}">
        {#each posts as post (post.id)}
          <div class="post-item">
            <BlogPostCard {post} />
          </div>
        {/each}
      </div>

      <!-- Load More Button -->
      {#if hasMore}
        <div class="load-more-section">
          <button
            type="button"
            class="load-more-btn"
            on:click={loadMore}
            disabled={loading}
          >
            {#if loading}
              <div class="spinner-small"></div>
              Loading...
            {:else}
              Load More Posts ({total - posts.length} remaining)
            {/if}
          </button>
        </div>
      {/if}
    </section>
  {/if}

  <!-- Results Summary -->
  {#if posts.length > 0}
    <footer class="results-footer">
      <p class="results-count">
        Showing {posts.length} of {total} posts
      </p>
    </footer>
  {/if}
</div>

<style>
  .blog-list-container {
    @apply max-w-7xl mx-auto px-4 py-8;
  }

  .blog-header {
    @apply flex justify-between items-center mb-8;
  }

  .blog-title {
    @apply text-3xl font-bold text-gray-900;
  }

  .header-actions {
    @apply flex gap-2;
  }

  .view-toggle {
    @apply p-2 text-gray-600 hover:text-gray-900 bg-white border border-gray-300 rounded-lg hover:bg-gray-50 transition-colors duration-200;
  }

  .filters-section {
    @apply mb-8 p-6 bg-white rounded-lg shadow-sm border border-gray-200;
  }

  .filters-grid {
    @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4 mb-4;
  }

  .search-box {
    @apply relative;
  }

  .search-input {
    @apply w-full px-4 py-2 pr-10 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent;
  }

  .search-btn {
    @apply absolute right-2 top-1/2 transform -translate-y-1/2 p-2 text-gray-500 hover:text-gray-700;
  }

  .filter-group {
    @apply flex flex-col;
  }

  .filter-label {
    @apply text-sm font-medium text-gray-700 mb-1;
  }

  .filter-select {
    @apply px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent;
  }

  .active-filters {
    @apply flex flex-wrap gap-2 items-center;
  }

  .filter-tag {
    @apply inline-flex items-center px-3 py-1 text-sm bg-blue-100 text-blue-800 rounded-full hover:bg-blue-200 transition-colors duration-200;
  }

  .posts-section {
    @apply mb-8;
  }

  .posts-container {
    @apply grid gap-6;
  }

  .posts-container.grid {
    @apply grid-cols-1 md:grid-cols-2 lg:grid-cols-3;
  }

  .posts-container.list {
    @apply grid-cols-1;
  }

  .post-item {
    @apply w-full;
  }

  .loading-state,
  .error-state,
  .empty-state {
    @apply flex flex-col items-center justify-center py-16 text-center;
  }

  .spinner {
    @apply w-8 h-8 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin mb-4;
  }

  .spinner-small {
    @apply w-4 h-4 border-2 border-gray-200 border-t-blue-500 rounded-full animate-spin inline-block;
  }

  .error-message {
    @apply text-red-600 mb-4;
  }

  .retry-btn,
  .clear-filters-btn,
  .load-more-btn {
    @apply px-6 py-2 bg-blue-600 text-white rounded-lg hover:bg-blue-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed;
  }

  .load-more-section {
    @apply flex justify-center mt-8;
  }

  .results-footer {
    @apply text-center text-gray-500 text-sm;
  }

  .results-count {
    @apply mb-0;
  }

  /* Responsive adjustments */
  @media (max-width: 768px) {
    .blog-header {
      @apply flex-col gap-4 items-start;
    }

    .filters-grid {
      @apply grid-cols-1;
    }

    .posts-container.grid {
      @apply grid-cols-1;
    }
  }
</style>