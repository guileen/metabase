<script lang="ts">
  import { onMount } from 'svelte';
  import CMSNavigation from '../components/CMSNavigation.svelte';
  import BlogList from '../components/BlogList.svelte';
  import { CMSApiClient } from '../utils/api';
  import type { Content, Category } from '../types/cms';

  let featuredPosts: Content[] = [];
  let latestPosts: Content[] = [];
  let categories: Category[] = [];
  let loading = true;
  let error: string | null = null;

  // Site settings
  let siteTitle = 'MetaBase CMS';
  let siteDescription = 'A powerful content management system';
  let welcomeMessage = '';

  const api = new CMSApiClient({
    tenant_id: '00000000-0000-0000-0000-000000000001',
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
    try {
      // Load site settings
      const settings = await api.getSiteSettings();
      siteTitle = settings.site_title;
      siteDescription = settings.site_description;
      welcomeMessage = settings.welcome_message || '';

      // Load homepage content
      await Promise.all([
        loadFeaturedPosts(),
        loadLatestPosts(),
        loadCategories(),
      ]);

      // Update page title
      document.title = siteTitle;
      document.querySelector('meta[name="description"]')?.setAttribute('content', siteDescription);

    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load homepage';
    } finally {
      loading = false;
    }
  });

  async function loadFeaturedPosts() {
    try {
      const response = await api.getContent({
        content_type_id: 'blog-posts',
        featured: true,
        limit: 3,
        sort_by: 'published_at',
        sort_order: 'desc',
      });
      featuredPosts = response.items;
    } catch (err) {
      console.error('Failed to load featured posts:', err);
    }
  }

  async function loadLatestPosts() {
    try {
      const response = await api.getContent({
        content_type_id: 'blog-posts',
        status: 'published',
        limit: 6,
        sort_by: 'published_at',
        sort_order: 'desc',
      });
      latestPosts = response.items;
    } catch (err) {
      console.error('Failed to load latest posts:', err);
    }
  }

  async function loadCategories() {
    try {
      const contentType = await api.getContentType('blog-posts');
      categories = await api.getCategories(contentType.id);
    } catch (err) {
      console.error('Failed to load categories:', err);
    }
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'short',
      day: 'numeric',
    });
  }

  function getExcerpt(content: string, maxLength = 150): string {
    const plainText = content.replace(/<[^>]*>/g, '');
    if (plainText.length <= maxLength) return plainText;
    return plainText.substring(0, maxLength).trim() + '...';
  }

  function getReadingTime(content: string): string {
    const wordsPerMinute = 200;
    const words = content.replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    const minutes = Math.ceil(words / wordsPerMinute);
    return `${minutes} min read`;
  }
</script>

<svelte:head>
  <title>{siteTitle}</title>
  <meta name="description" content={siteDescription} />
</svelte:head>

<div class="home-page">
  <!-- Navigation -->
  <CMSNavigation currentPath="/" />

  <!-- Hero Section -->
  <section class="hero-section">
    <div class="hero-container">
      <div class="hero-content">
        <h1 class="hero-title">{siteTitle}</h1>
        <p class="hero-description">{siteDescription}</p>

        {#if welcomeMessage}
          <div class="welcome-message">
            <p>{welcomeMessage}</p>
          </div>
        {/if}

        <div class="hero-actions">
          <a href="/blog" class="btn btn-primary">
            Read Our Blog
          </a>
          <a href="/contact" class="btn btn-secondary">
            Get in Touch
          </a>
        </div>
      </div>

      <!-- Hero Image/Pattern -->
      <div class="hero-visual">
        <div class="hero-pattern"></div>
      </div>
    </div>
  </section>

  <!-- Featured Posts Section -->
  {#if featuredPosts.length > 0}
    <section class="featured-section">
      <div class="container">
        <div class="section-header">
          <h2 class="section-title">Featured Posts</h2>
          <p class="section-subtitle">Hand-picked content you shouldn't miss</p>
        </div>

        <div class="featured-grid">
          {#each featuredPosts as post, index}
            <article class="featured-card featured-{index + 1}">
              <!-- Featured Image -->
              {#if post.custom_fields?.featured_image}
                <div class="featured-image">
                  <img
                    src={post.custom_fields.featured_image}
                    alt={post.custom_fields.featured_image_alt || post.title}
                    loading="lazy"
                  />
                </div>
              {/if}

              <div class="featured-content">
                <!-- Categories -->
                {#if post.categories && post.categories.length > 0}
                  <div class="featured-categories">
                    {#each post.categories.slice(0, 2) as category}
                      <span
                        class="featured-category-badge"
                        style="background-color: {category.color || '#3b82f6'}"
                      >
                        {category.name}
                      </span>
                    {/each}
                  </div>
                {/if}

                <h3 class="featured-title">
                  <a href="/blog/{post.slug}" class="featured-title-link">
                    {post.title}
                  </a>
                </h3>

                <p class="featured-excerpt">
                  {getExcerpt(post.excerpt || post.content || '', 200)}
                </p>

                <div class="featured-meta">
                  <time datetime={post.published_at} class="featured-date">
                    {formatDate(post.published_at!)}
                  </time>

                  {#if post.content}
                    <span class="featured-reading-time">
                      📖 {getReadingTime(post.content)}
                    </span>
                  {/if}

                  {#if post.view_count > 0}
                    <span class="featured-views">
                      👁 {post.view_count.toLocaleString()}
                    </span>
                  {/if}
                </div>
              </div>
            </article>
          {/each}
        </div>
      </div>
    </section>
  {/if}

  <!-- Categories Section -->
  {#if categories.length > 0}
    <section class="categories-section">
      <div class="container">
        <div class="section-header">
          <h2 class="section-title">Categories</h2>
          <p class="section-subtitle">Explore content by topic</p>
        </div>

        <div class="categories-grid">
          {#each categories.slice(0, 6) as category}
            <a href="/blog/category/{category.slug}" class="category-card">
              <div class="category-icon" style="background-color: {category.color || '#3b82f6'}">
                {#if category.image_id}
                  <img src="/api/v1/cms/media/{category.image_id}" alt={category.name} />
                {:else}
                  📁
                {/if}
              </div>

              <div class="category-content">
                <h3 class="category-title">{category.name}</h3>
                {#if category.description}
                  <p class="category-description">{category.description}</p>
                {/if}
                <span class="category-count">
                  {category.post_count || 0} posts
                </span>
              </div>
            </a>
          {/each}
        </div>
      </div>
    </section>
  {/if}

  <!-- Latest Posts Section -->
  {#if latestPosts.length > 0}
    <section class="latest-section">
      <div class="container">
        <div class="section-header">
          <h2 class="section-title">Latest Posts</h2>
          <p class="section-subtitle">Fresh content from our blog</p>
        </div>

        <div class="latest-grid">
          {#each latestPosts as post}
            <article class="latest-card">
              <!-- Post Image -->
              {#if post.custom_fields?.featured_image}
                <div class="latest-image">
                  <img
                    src={post.custom_fields.featured_image}
                    alt={post.custom_fields.featured_image_alt || post.title}
                    loading="lazy"
                  />
                </div>
              {/if}

              <div class="latest-content">
                <!-- Categories -->
                {#if post.categories && post.categories.length > 0}
                  <div class="latest-categories">
                    {#each post.categories.slice(0, 1) as category}
                      <span
                        class="latest-category-badge"
                        style="background-color: {category.color || '#3b82f6'}"
                      >
                        {category.name}
                      </span>
                    {/each}
                  </div>
                {/if}

                <h3 class="latest-title">
                  <a href="/blog/{post.slug}" class="latest-title-link">
                    {post.title}
                  </a>
                </h3>

                <div class="latest-meta">
                  <time datetime={post.published_at} class="latest-date">
                    {formatDate(post.published_at!)}
                  </time>

                  {#if post.comment_count > 0}
                    <span class="latest-comments">
                      💬 {post.comment_count}
                    </span>
                  {/if}
                </div>
              </div>
            </article>
          {/each}
        </div>

        <!-- View All Button -->
        <div class="section-footer">
          <a href="/blog" class="btn btn-outline">
            View All Posts →
          </a>
        </div>
      </div>
    </section>
  {/if}

  <!-- Call to Action Section -->
  <section class="cta-section">
    <div class="container">
      <div class="cta-content">
        <h2 class="cta-title">Want to stay updated?</h2>
        <p class="cta-description">
          Subscribe to our newsletter to get the latest posts and updates delivered to your inbox.
        </p>

        <form class="cta-form" on:submit|preventDefault>
          <div class="cta-form-row">
            <input
              type="email"
              placeholder="Enter your email address"
              class="cta-input"
              required
            />
            <button type="submit" class="btn btn-primary">
              Subscribe
            </button>
          </div>
        </form>
      </div>
    </div>
  </section>

  <!-- Loading State -->
  {#if loading}
    <div class="loading-state">
      <div class="spinner"></div>
      <p>Loading homepage...</p>
    </div>
  {/if}

  <!-- Error State -->
  {#if error}
    <div class="error-state">
      <h2>Something went wrong</h2>
      <p>{error}</p>
      <button type="button" class="btn btn-primary" on:click={() => window.location.reload()}>
        Try Again
      </button>
    </div>
  {/if}
</div>

<style>
  .home-page {
    @apply min-h-screen;
  }

  .container {
    @apply max-w-7xl mx-auto px-4;
  }

  /* Hero Section */
  .hero-section {
    @apply bg-gradient-to-br from-blue-50 via-white to-purple-50 py-16 lg:py-24;
  }

  .hero-container {
    @apply container mx-auto px-4;
    @apply flex flex-col lg:flex-row items-center gap-12 lg:gap-16;
  }

  .hero-content {
    @apply flex-1 text-center lg:text-left;
  }

  .hero-title {
    @apply text-4xl lg:text-6xl font-bold text-gray-900 mb-6 leading-tight;
    background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
    -webkit-background-clip: text;
    -webkit-text-fill-color: transparent;
    background-clip: text;
  }

  .hero-description {
    @apply text-xl text-gray-600 mb-8 leading-relaxed;
  }

  .welcome-message {
    @apply p-6 bg-white/80 backdrop-blur-sm rounded-lg border border-blue-100 mb-8 shadow-sm;
  }

  .welcome-message p {
    @apply text-gray-700 m-0 leading-relaxed;
  }

  .hero-actions {
    @apply flex flex-col sm:flex-row gap-4 justify-center lg:justify-start;
  }

  .btn {
    @apply inline-flex items-center px-6 py-3 font-semibold rounded-lg transition-all duration-200 no-underline text-center;
  }

  .btn-primary {
    @apply bg-blue-600 text-white hover:bg-blue-700 shadow-lg hover:shadow-xl transform hover:-translate-y-0.5;
  }

  .btn-secondary {
    @apply bg-white text-blue-600 border-2 border-blue-600 hover:bg-blue-50;
  }

  .btn-outline {
    @apply bg-transparent text-blue-600 border-2 border-blue-600 hover:bg-blue-600 hover:text-white;
  }

  .hero-visual {
    @apply flex-1 flex items-center justify-center;
  }

  .hero-pattern {
    @apply w-full h-64 lg:h-96 bg-gradient-to-br from-blue-400/20 to-purple-600/20 rounded-2xl;
    position: relative;
    overflow: hidden;
  }

  .hero-pattern::before {
    content: '';
    position: absolute;
    top: -50%;
    left: -50%;
    width: 200%;
    height: 200%;
    background: radial-gradient(circle, rgba(59, 130, 246, 0.1) 1px, transparent 1px);
    background-size: 20px 20px;
    animation: float 20s linear infinite;
  }

  @keyframes float {
    0% { transform: translate(0, 0); }
    100% { transform: translate(20px, 20px); }
  }

  /* Section Headers */
  .section-header {
    @apply text-center mb-12;
  }

  .section-title {
    @apply text-3xl lg:text-4xl font-bold text-gray-900 mb-4;
  }

  .section-subtitle {
    @apply text-xl text-gray-600 max-w-2xl mx-auto;
  }

  /* Featured Section */
  .featured-section {
    @apply py-16 bg-white;
  }

  .featured-grid {
    @apply grid grid-cols-1 lg:grid-cols-3 gap-8;
  }

  .featured-card {
    @apply bg-white rounded-xl shadow-lg hover:shadow-xl transition-all duration-300 overflow-hidden border border-gray-100;
  }

  .featured-card.featured-1 {
    @apply lg:col-span-2 lg:row-span-2;
  }

  .featured-image {
    @apply aspect-video lg:aspect-[2/1] overflow-hidden bg-gray-100;
  }

  .featured-card.featured-1 .featured-image {
    @apply aspect-[16/9];
  }

  .featured-image img {
    @apply w-full h-full object-cover transition-transform duration-300 hover:scale-105;
  }

  .featured-content {
    @apply p-6 lg:p-8;
  }

  .featured-categories {
    @apply flex flex-wrap gap-2 mb-4;
  }

  .featured-category-badge {
    @apply inline-block px-3 py-1 text-xs font-medium text-white rounded-full;
  }

  .featured-title {
    @apply text-xl lg:text-2xl font-bold text-gray-900 mb-4 line-clamp-2;
  }

  .featured-title-link {
    @apply text-gray-900 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .featured-excerpt {
    @apply text-gray-600 mb-6 line-clamp-3 leading-relaxed;
  }

  .featured-meta {
    @apply flex flex-wrap items-center gap-4 text-sm text-gray-500;
  }

  /* Categories Section */
  .categories-section {
    @apply py-16 bg-gray-50;
  }

  .categories-grid {
    @apply grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-6;
  }

  .category-card {
    @apply bg-white rounded-lg p-6 shadow-md hover:shadow-lg transition-all duration-300 flex items-start gap-4 no-underline group;
  }

  .category-icon {
    @apply w-12 h-12 rounded-lg flex items-center justify-center text-white text-xl flex-shrink-0;
  }

  .category-icon img {
    @apply w-8 h-8 object-cover rounded;
  }

  .category-content {
    @apply flex-1;
  }

  .category-title {
    @apply text-lg font-semibold text-gray-900 mb-2 group-hover:text-blue-600 transition-colors duration-200;
  }

  .category-description {
    @apply text-gray-600 text-sm mb-2 line-clamp-2;
  }

  .category-count {
    @apply text-xs text-gray-500;
  }

  /* Latest Section */
  .latest-section {
    @apply py-16 bg-white;
  }

  .latest-grid {
    @apply grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-8;
  }

  .latest-card {
    @apply bg-white rounded-lg shadow-md hover:shadow-lg transition-all duration-300 overflow-hidden border border-gray-100 group;
  }

  .latest-image {
    @apply aspect-video overflow-hidden bg-gray-100;
  }

  .latest-image img {
    @apply w-full h-full object-cover transition-transform duration-300 group-hover:scale-105;
  }

  .latest-content {
    @apply p-6;
  }

  .latest-categories {
    @apply flex flex-wrap gap-2 mb-3;
  }

  .latest-category-badge {
    @apply inline-block px-2 py-1 text-xs font-medium text-white rounded-full;
  }

  .latest-title {
    @apply text-lg font-semibold text-gray-900 mb-3 line-clamp-2;
  }

  .latest-title-link {
    @apply text-gray-900 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .latest-meta {
    @apply flex items-center gap-4 text-sm text-gray-500;
  }

  /* CTA Section */
  .cta-section {
    @apply py-16 bg-gradient-to-r from-blue-600 to-purple-600 text-white;
  }

  .cta-content {
    @apply text-center max-w-2xl mx-auto;
  }

  .cta-title {
    @apply text-3xl lg:text-4xl font-bold mb-4;
  }

  .cta-description {
    @apply text-xl mb-8 opacity-90;
  }

  .cta-form {
    @apply max-w-md mx-auto;
  }

  .cta-form-row {
    @apply flex flex-col sm:flex-row gap-4;
  }

  .cta-input {
    @apply flex-1 px-4 py-3 rounded-lg border-2 border-transparent focus:border-white/50 focus:outline-none bg-white/20 backdrop-blur-sm text-white placeholder-white/70;
  }

  /* Loading and Error States */
  .loading-state,
  .error-state {
    @apply flex flex-col items-center justify-center py-32 text-center;
  }

  .spinner {
    @apply w-12 h-12 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin mb-4;
  }

  .error-state h2 {
    @apply text-2xl font-bold text-gray-900 mb-4;
  }

  /* Section Footer */
  .section-footer {
    @apply text-center mt-12;
  }

  /* Responsive adjustments */
  @media (max-width: 768px) {
    .hero-title {
      @apply text-3xl lg:text-4xl;
    }

    .featured-grid {
      @apply grid-cols-1 gap-6;
    }

    .featured-card.featured-1 {
      @apply col-span-1 row-span-1;
    }

    .categories-grid {
      @apply grid-cols-1;
    }

    .latest-grid {
      @apply grid-cols-1;
    }
  }
</style>