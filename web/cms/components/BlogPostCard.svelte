<script lang="ts">
  import { formatDistance } from 'date-fns';
  import type { Content } from '../types/cms';

  export let post: Content;
  export let showExcerpt = true;
  export let showAuthor = true;
  export let showDate = true;
  export let showCategory = true;
  export let showTags = true;
  export let showReadingTime = true;

  // Format functions
  function formatDate(dateString: string) {
    return formatDistance(new Date(dateString), new Date(), { addSuffix: true });
  }

  function getReadingTime(content: string): string {
    const wordsPerMinute = 200;
    const words = content.replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    const minutes = Math.ceil(words / wordsPerMinute);
    return `${minutes} min read`;
  }

  function getExcerpt(content: string, maxLength = 160): string {
    const plainText = content.replace(/<[^>]*>/g, '');
    if (plainText.length <= maxLength) return plainText;
    return plainText.substring(0, maxLength).trim() + '...';
  }
</script>

<div class="blog-post-card">
  <article class="card">
    <!-- Featured Image -->
    {#if post.custom_fields?.featured_image}
      <div class="card-image">
        <img
          src={post.custom_fields.featured_image}
          alt={post.custom_fields.featured_image_alt || post.title}
          loading="lazy"
        />
      </div>
    {/if}

    <div class="card-content">
      <!-- Categories -->
      {#if showCategory && post.categories && post.categories.length > 0}
        <div class="categories">
          {#each post.categories as category}
            <span
              class="category-badge"
              style="background-color: {category.color || '#3b82f6'}"
            >
              {category.name}
            </span>
          {/each}
        </div>
      {/if}

      <!-- Title -->
      <h2 class="card-title">
        <a href="/blog/{post.slug}" class="title-link">
          {post.title}
        </a>
      </h2>

      <!-- Excerpt -->
      {#if showExcerpt}
        <p class="card-excerpt">
          {getExcerpt(post.excerpt || post.content || '')}
        </p>
      {/if}

      <!-- Tags -->
      {#if showTags && post.tags && post.tags.length > 0}
        <div class="tags">
          {#each post.tags.slice(0, 3) as tag}
            <a href="/tag/{tag.slug}" class="tag-link">
              #{tag.name}
            </a>
          {/each}
        </div>
      {/if}

      <!-- Meta information -->
      <div class="card-meta">
        {#if showAuthor && post.author}
          <div class="author">
            <img
              src={post.author.avatar || '/default-avatar.png'}
              alt={post.author.name}
              class="author-avatar"
            />
            <span class="author-name">{post.author.name}</span>
          </div>
        {/if}

        <div class="meta-info">
          {#if showDate && post.published_at}
            <time datetime={post.published_at} class="publish-date">
              {formatDate(post.published_at)}
            </time>
          {/if}

          {#if showReadingTime && post.content}
            <span class="reading-time">
              {getReadingTime(post.content)}
            </span>
          {/if}
        </div>

        <!-- Stats -->
        <div class="stats">
          {#if post.view_count > 0}
            <span class="stat">
              👁 {post.view_count.toLocaleString()}
            </span>
          {/if}

          {#if post.like_count > 0}
            <span class="stat">
              ❤ {post.like_count.toLocaleString()}
            </span>
          {/if}

          {#if post.comment_count > 0}
            <span class="stat">
              💬 {post.comment_count.toLocaleString()}
            </span>
          {/if}
        </div>
      </div>

      <!-- Call to action -->
      <div class="card-actions">
        <a href="/blog/{post.slug}" class="read-more-btn">
          Read More →
        </a>
      </div>
    </div>

    <!-- Featured/Sticky indicators -->
    {#if post.featured || post.sticky}
      <div class="indicators">
        {#if post.featured}
          <span class="indicator featured">⭐ Featured</span>
        {/if}
        {#if post.sticky}
          <span class="indicator sticky">📌 Pinned</span>
        {/if}
      </div>
    {/if}
  </article>
</div>

<style>
  .blog-post-card {
    @apply w-full;
  }

  .card {
    @apply bg-white rounded-lg shadow-md hover:shadow-lg transition-shadow duration-200 overflow-hidden relative;
    border: 1px solid #e5e7eb;
  }

  .card-image {
    @apply aspect-video overflow-hidden bg-gray-100;
  }

  .card-image img {
    @apply w-full h-full object-cover transition-transform duration-200 hover:scale-105;
  }

  .card-content {
    @apply p-6;
  }

  .categories {
    @apply flex flex-wrap gap-2 mb-3;
  }

  .category-badge {
    @apply inline-block px-2 py-1 text-xs font-medium text-white rounded-full;
  }

  .card-title {
    @apply text-xl font-bold mb-3 line-clamp-2;
  }

  .title-link {
    @apply text-gray-900 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .card-excerpt {
    @apply text-gray-600 mb-4 line-clamp-3;
  }

  .tags {
    @apply flex flex-wrap gap-2 mb-4;
  }

  .tag-link {
    @apply text-sm text-blue-600 hover:text-blue-800 transition-colors duration-200 no-underline;
  }

  .card-meta {
    @apply flex items-center justify-between text-sm text-gray-500 mb-4;
  }

  .author {
    @apply flex items-center gap-2;
  }

  .author-avatar {
    @apply w-6 h-6 rounded-full object-cover;
  }

  .author-name {
    @apply font-medium;
  }

  .meta-info {
    @apply flex items-center gap-3;
  }

  .publish-date {
    @apply text-gray-500;
  }

  .reading-time {
    @apply text-gray-500;
  }

  .stats {
    @apply flex items-center gap-3;
  }

  .stat {
    @apply text-xs text-gray-500;
  }

  .card-actions {
    @apply flex justify-end;
  }

  .read-more-btn {
    @apply inline-flex items-center px-4 py-2 text-sm font-medium text-blue-600 bg-blue-50 rounded-lg hover:bg-blue-100 transition-colors duration-200 no-underline;
  }

  .indicators {
    @apply absolute top-3 right-3 flex flex-col gap-2;
  }

  .indicator {
    @apply px-2 py-1 text-xs font-medium rounded-full backdrop-blur-sm;
  }

  .indicator.featured {
    @apply bg-yellow-500/80 text-white;
  }

  .indicator.sticky {
    @apply bg-green-500/80 text-white;
  }

  /* Responsive adjustments */
  @media (max-width: 768px) {
    .card-content {
      @apply p-4;
    }

    .card-meta {
      @apply flex-col items-start gap-2;
    }

    .stats {
      @apply w-full justify-between;
    }
  }
</style>