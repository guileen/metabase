<script lang="ts">
  import { onMount } from 'svelte';
  import { page } from '$app/stores';
  import CMSNavigation from '../components/CMSNavigation.svelte';
  import { CMSApiClient } from '../utils/api';
  import type { Content, Comment } from '../types/cms';

  let post: Content | null = null;
  let loading = true;
  let error: string | null = null;
  let comments: Comment[] = [];
  let commentsLoading = false;

  // Comment form
  let commentForm = {
    author_name: '',
    author_email: '',
    content: '',
    parent_id: null as string | null,
  };

  let commentSubmitting = false;
  let commentSuccess = false;
  let commentError: string | null = null;

  // Social sharing
  let shareUrl = '';
  let shareTitle = '';

  // Related posts
  let relatedPosts: Content[] = [];

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
      const slug = $page.params.slug;
      if (!slug) {
        error = 'Post not found';
        return;
      }

      // Load the post
      post = await api.getContentBySlug(slug, 'blog-posts');

      // Update meta tags
      if (post) {
        shareUrl = window.location.href;
        shareTitle = post.title;

        // Update page title
        document.title = `${post.title} - Blog`;

        // Update meta description
        const metaDescription = document.querySelector('meta[name="description"]');
        if (metaDescription) {
          metaDescription.setAttribute('content', post.meta_description || post.excerpt || '');
        }

        // Increment view count
        await api.incrementViewCount(post.id);

        // Load comments
        if (post.content_type?.has_comments) {
          await loadComments();
        }

        // Load related posts
        await loadRelatedPosts();
      }

    } catch (err) {
      error = err instanceof Error ? err.message : 'Failed to load post';
    } finally {
      loading = false;
    }
  });

  async function loadComments() {
    if (!post) return;

    try {
      commentsLoading = true;
      comments = await api.getComments(post.id, 'approved');
    } catch (err) {
      console.error('Failed to load comments:', err);
    } finally {
      commentsLoading = false;
    }
  }

  async function loadRelatedPosts() {
    if (!post) return;

    try {
      // Get posts from the same categories or with similar tags
      const categoryIds = post.categories?.map(c => c.id) || [];
      const tagIds = post.tags?.map(t => t.id) || [];

      const related = await api.search('', {
        content_type_id: 'blog-posts',
        category_id: categoryIds[0], // Use first category
        limit: 3,
      });

      // Filter out the current post
      relatedPosts = related.items.filter(p => p.id !== post!.id).slice(0, 3);
    } catch (err) {
      console.error('Failed to load related posts:', err);
    }
  }

  async function submitComment() {
    if (!post || !commentForm.content.trim() || !commentForm.author_name.trim()) {
      commentError = 'Please fill in all required fields';
      return;
    }

    try {
      commentSubmitting = true;
      commentError = null;

      const newComment = await api.createComment({
        content_id: post.id,
        author_name: commentForm.author_name.trim(),
        author_email: commentForm.author_email.trim() || undefined,
        content: commentForm.content.trim(),
        parent_id: commentForm.parent_id || undefined,
      });

      if (newComment.status === 'approved') {
        comments = [...comments, newComment];
        commentSuccess = true;
      } else {
        commentSuccess = true; // Comment submitted for moderation
      }

      // Reset form
      commentForm = {
        author_name: '',
        author_email: '',
        content: '',
        parent_id: null,
      };

      // Clear success message after 3 seconds
      setTimeout(() => {
        commentSuccess = false;
      }, 3000);

    } catch (err) {
      commentError = err instanceof Error ? err.message : 'Failed to submit comment';
    } finally {
      commentSubmitting = false;
    }
  }

  function replyToComment(comment: Comment) {
    commentForm.parent_id = comment.id;
    commentForm.author_name = '';
    commentForm.author_email = '';
    commentForm.content = '';

    // Focus on comment form
    const commentFormElement = document.getElementById('comment-form');
    commentFormElement?.scrollIntoView({ behavior: 'smooth' });
  }

  function formatDate(dateString: string) {
    return new Date(dateString).toLocaleDateString('en-US', {
      year: 'numeric',
      month: 'long',
      day: 'numeric',
    });
  }

  function getReadingTime(content: string): string {
    const wordsPerMinute = 200;
    const words = content.replace(/<[^>]*>/g, '').trim().split(/\s+/).length;
    const minutes = Math.ceil(words / wordsPerMinute);
    return `${minutes} min read`;
  }

  function shareOnTwitter() {
    const text = encodeURIComponent(`${shareTitle} ${shareUrl}`);
    window.open(`https://twitter.com/intent/tweet?text=${text}`, '_blank');
  }

  function shareOnFacebook() {
    window.open(`https://www.facebook.com/sharer/sharer.php?u=${encodeURIComponent(shareUrl)}`, '_blank');
  }

  function shareOnLinkedIn() {
    window.open(`https://www.linkedin.com/sharing/share-offsite/?url=${encodeURIComponent(shareUrl)}`, '_blank');
  }

  function copyShareLink() {
    navigator.clipboard.writeText(shareUrl).then(() => {
      // Show success message (could be enhanced with toast notification)
      alert('Link copied to clipboard!');
    });
  }
</script>

<svelte:head>
  {#if post}
    <title>{post.meta_title || `${post.title} - Blog`}</title>
    <meta name="description" content={post.meta_description || post.excerpt || ''} />
    {#if post.meta_keywords}
      <meta name="keywords" content={post.meta_keywords} />
    {/if}

    <!-- Open Graph tags -->
    <meta property="og:title" content={post.title} />
    <meta property="og:description" content={post.meta_description || post.excerpt || ''} />
    <meta property="og:type" content="article" />
    <meta property="og:url" content={shareUrl} />
    {#if post.custom_fields?.featured_image}
      <meta property="og:image" content={post.custom_fields.featured_image} />
    {/if}

    <!-- Twitter Card tags -->
    <meta name="twitter:card" content="summary_large_image" />
    <meta name="twitter:title" content={post.title} />
    <meta name="twitter:description" content={post.meta_description || post.excerpt || ''} />
    {#if post.custom_fields?.featured_image}
      <meta name="twitter:image" content={post.custom_fields.featured_image} />
    {/if}
  {/if}
</svelte:head>

<div class="blog-post-page">
  <!-- Navigation -->
  <CMSNavigation currentPath="/blog/{$page.params.slug}" />

  <main class="post-container">
    {#if loading}
      <div class="loading-state">
        <div class="spinner"></div>
        <p>Loading post...</p>
      </div>
    {:else if error}
      <div class="error-state">
        <h1>Post Not Found</h1>
        <p>{error}</p>
        <a href="/blog" class="back-link">← Back to Blog</a>
      </div>
    {:else if post}
      <article class="blog-post">
        <!-- Post Header -->
        <header class="post-header">
          <!-- Categories -->
          {#if post.categories && post.categories.length > 0}
            <div class="post-categories">
              {#each post.categories as category}
                <span
                  class="category-badge"
                  style="background-color: {category.color || '#3b82f6'}"
                >
                  <a href="/blog/category/{category.slug}" class="category-link">
                    {category.name}
                  </a>
                </span>
              {/each}
            </div>
          {/if}

          <h1 class="post-title">{post.title}</h1>

          <!-- Post Meta -->
          <div class="post-meta">
            <div class="meta-left">
              {#if post.author}
                <div class="author-info">
                  <img
                    src={post.author.avatar || '/default-avatar.png'}
                    alt={post.author.name}
                    class="author-avatar"
                  />
                  <div class="author-details">
                    <span class="author-name">{post.author.name}</span>
                    {#if post.author.bio}
                      <p class="author-bio">{post.author.bio}</p>
                    {/if}
                  </div>
                </div>
              {/if}
            </div>

            <div class="meta-right">
              <time datetime={post.published_at} class="publish-date">
                {formatDate(post.published_at!)}
              </time>

              {#if post.content}
                <span class="reading-time">
                  📖 {getReadingTime(post.content)}
                </span>
              {/if}
            </div>
          </div>

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
        </header>

        <!-- Post Content -->
        <div class="post-content">
          {@html post.content || ''}
        </div>

        <!-- Tags -->
        {#if post.tags && post.tags.length > 0}
          <footer class="post-tags">
            <h3>Tags</h3>
            <div class="tags-list">
              {#each post.tags as tag}
                <a href="/blog/tag/{tag.slug}" class="tag-link">
                  #{tag.name}
                </a>
              {/each}
            </div>
          </footer>
        {/if}
      </article>

      <!-- Share Section -->
      <section class="share-section">
        <h3>Share this post</h3>
        <div class="share-buttons">
          <button type="button" class="share-btn twitter" on:click={shareOnTwitter}>
            🐦 Twitter
          </button>
          <button type="button" class="share-btn facebook" on:click={shareOnFacebook}>
            📘 Facebook
          </button>
          <button type="button" class="share-btn linkedin" on:click={shareOnLinkedIn}>
            💼 LinkedIn
          </button>
          <button type="button" class="share-btn copy" on:click={copyShareLink}>
            📋 Copy Link
          </button>
        </div>
      </section>

      <!-- Comments Section -->
      {#if post.content_type?.has_comments}
        <section class="comments-section">
          <h3>Comments ({post.comment_count})</h3>

          <!-- Comment Form -->
          <div class="comment-form-container">
            <h4>Leave a Comment</h4>

            {#if commentSuccess}
              <div class="comment-success">
                {#if commentSuccess && comments.some(c => c.status === 'pending')}
                  Your comment has been submitted for moderation.
                {:else}
                  Your comment has been posted successfully!
                {/if}
              </div>
            {/if}

            {#if commentError}
              <div class="comment-error">
                {commentError}
              </div>
            {/if}

            <form id="comment-form" class="comment-form" on:submit|preventDefault={submitComment}>
              <div class="form-row">
                <div class="form-group">
                  <label for="author_name">Name *</label>
                  <input
                    id="author_name"
                    type="text"
                    bind:value={commentForm.author_name}
                    required
                    class="form-input"
                  />
                </div>

                <div class="form-group">
                  <label for="author_email">Email (optional)</label>
                  <input
                    id="author_email"
                    type="email"
                    bind:value={commentForm.author_email}
                    class="form-input"
                  />
                </div>
              </div>

              <div class="form-group">
                <label for="comment_content">Comment *</label>
                <textarea
                  id="comment_content"
                  bind:value={commentForm.content}
                  required
                  rows="4"
                  class="form-textarea"
                  placeholder="Share your thoughts..."
                ></textarea>
              </div>

              {#if commentForm.parent_id}
                <div class="replying-to">
                  Replying to comment.
                  <button
                    type="button"
                    class="cancel-reply"
                    on:click={() => commentForm.parent_id = null}
                  >
                    Cancel
                  </button>
                </div>
              {/if}

              <button
                type="submit"
                class="submit-btn"
                disabled={commentSubmitting}
              >
                {#if commentSubmitting}
                  <div class="spinner-small"></div>
                  Posting...
                {:else}
                  Post Comment
                {/if}
              </button>
            </form>
          </div>

          <!-- Comments List -->
          <div class="comments-list">
            {#if commentsLoading}
              <div class="loading-comments">
                <div class="spinner-small"></div>
                <p>Loading comments...</p>
              </div>
            {:else if comments.length === 0}
              <p class="no-comments">No comments yet. Be the first to share your thoughts!</p>
            {:else}
              {#each comments as comment}
                <div class="comment">
                  <div class="comment-header">
                    <div class="comment-author">
                      <img
                        src={comment.author?.avatar || '/default-avatar.png'}
                        alt={comment.author_name}
                        class="comment-avatar"
                      />
                      <div class="author-info">
                        <span class="comment-name">{comment.author_name}</span>
                        <time datetime={comment.created_at} class="comment-date">
                          {formatDate(comment.created_at)}
                        </time>
                      </div>
                    </div>
                  </div>

                  <div class="comment-content">
                    {@html comment.content}
                  </div>

                  <div class="comment-actions">
                    <button
                      type="button"
                      class="reply-btn"
                      on:click={() => replyToComment(comment)}
                    >
                      Reply
                    </button>
                  </div>
                </div>
              {/each}
            {/if}
          </div>
        </section>
      {/if}

      <!-- Related Posts -->
      {#if relatedPosts.length > 0}
        <section class="related-posts">
          <h3>Related Posts</h3>
          <div class="related-posts-grid">
            {#each relatedPosts as relatedPost}
              <article class="related-post">
                {#if relatedPost.custom_fields?.featured_image}
                  <div class="related-post-image">
                    <img
                      src={relatedPost.custom_fields.featured_image}
                      alt={relatedPost.title}
                      loading="lazy"
                    />
                  </div>
                {/if}

                <div class="related-post-content">
                  <h4>
                    <a href="/blog/{relatedPost.slug}" class="related-post-link">
                      {relatedPost.title}
                    </a>
                  </h4>
                  <time datetime={relatedPost.published_at} class="related-post-date">
                    {formatDate(relatedPost.published_at!)}
                  </time>
                </div>
              </article>
            {/each}
          </div>
        </section>
      {/if}
    {/if}
  </main>
</div>

<style>
  .blog-post-page {
    @apply min-h-screen bg-gray-50;
  }

  .post-container {
    @apply max-w-4xl mx-auto px-4 py-8;
  }

  .loading-state,
  .error-state {
    @apply flex flex-col items-center justify-center py-16 text-center;
  }

  .spinner {
    @apply w-8 h-8 border-4 border-gray-200 border-t-blue-500 rounded-full animate-spin mb-4;
  }

  .spinner-small {
    @apply w-4 h-4 border-2 border-gray-200 border-t-blue-500 rounded-full animate-spin inline-block mr-2;
  }

  .error-state h1 {
    @apply text-2xl font-bold text-gray-900 mb-4;
  }

  .back-link {
    @apply inline-flex items-center text-blue-600 hover:text-blue-800 transition-colors duration-200 no-underline;
  }

  .blog-post {
    @apply bg-white rounded-lg shadow-sm overflow-hidden mb-8;
  }

  .post-header {
    @apply p-6 lg:p-8;
  }

  .post-categories {
    @apply flex flex-wrap gap-2 mb-4;
  }

  .category-badge {
    @apply inline-flex items-center px-3 py-1 text-xs font-medium text-white rounded-full;
  }

  .category-link {
    @apply text-white hover:text-white/80 transition-colors duration-200 no-underline;
  }

  .post-title {
    @apply text-3xl lg:text-4xl font-bold text-gray-900 mb-6 leading-tight;
  }

  .post-meta {
    @apply flex flex-col lg:flex-row lg:items-center lg:justify-between gap-4 mb-6 pb-6 border-b border-gray-200;
  }

  .author-info {
    @apply flex items-center gap-3;
  }

  .author-avatar {
    @apply w-12 h-12 rounded-full object-cover;
  }

  .author-details {
    @apply flex flex-col;
  }

  .author-name {
    @apply font-semibold text-gray-900;
  }

  .author-bio {
    @apply text-sm text-gray-600 m-0;
  }

  .meta-right {
    @apply flex flex-col gap-2 text-sm text-gray-600;
  }

  .publish-date,
  .reading-time {
    @apply m-0;
  }

  .featured-image {
    @apply -mx-6 lg:-mx-8 mb-6;
  }

  .featured-image img {
    @apply w-full h-auto max-h-[500px] object-cover;
  }

  .post-content {
    @apply p-6 lg:p-8 prose prose-lg max-w-none;
  }

  .post-tags {
    @apply px-6 lg:px-8 pb-6 lg:pb-8 border-t border-gray-200;
  }

  .post-tags h3 {
    @apply text-lg font-semibold text-gray-900 mb-4;
  }

  .tags-list {
    @apply flex flex-wrap gap-2;
  }

  .tag-link {
    @apply inline-block px-3 py-1 text-sm bg-gray-100 text-gray-700 rounded-full hover:bg-gray-200 transition-colors duration-200 no-underline;
  }

  .share-section {
    @apply bg-white rounded-lg shadow-sm p-6 mb-8;
  }

  .share-section h3 {
    @apply text-lg font-semibold text-gray-900 mb-4;
  }

  .share-buttons {
    @apply flex flex-wrap gap-3;
  }

  .share-btn {
    @apply px-4 py-2 text-sm font-medium text-white rounded-lg transition-colors duration-200;
  }

  .share-btn.twitter {
    @apply bg-blue-400 hover:bg-blue-500;
  }

  .share-btn.facebook {
    @apply bg-blue-600 hover:bg-blue-700;
  }

  .share-btn.linkedin {
    @apply bg-blue-700 hover:bg-blue-800;
  }

  .share-btn.copy {
    @apply bg-gray-600 hover:bg-gray-700;
  }

  .comments-section {
    @apply bg-white rounded-lg shadow-sm p-6 mb-8;
  }

  .comments-section h3 {
    @apply text-lg font-semibold text-gray-900 mb-6;
  }

  .comment-form-container {
    @apply mb-8 p-6 bg-gray-50 rounded-lg;
  }

  .comment-form h4 {
    @apply text-lg font-semibold text-gray-900 mb-4;
  }

  .comment-success {
    @apply p-4 mb-4 text-green-700 bg-green-100 rounded-lg;
  }

  .comment-error {
    @apply p-4 mb-4 text-red-700 bg-red-100 rounded-lg;
  }

  .form-row {
    @apply grid grid-cols-1 md:grid-cols-2 gap-4 mb-4;
  }

  .form-group {
    @apply flex flex-col;
  }

  .form-group label {
    @apply text-sm font-medium text-gray-700 mb-1;
  }

  .form-input,
  .form-textarea {
    @apply px-3 py-2 border border-gray-300 rounded-lg focus:ring-2 focus:ring-blue-500 focus:border-transparent;
  }

  .form-textarea {
    @apply resize-none;
  }

  .replying-to {
    @apply mb-4 p-3 bg-blue-50 border border-blue-200 rounded-lg text-sm text-blue-800;
  }

  .cancel-reply {
    @apply ml-2 text-blue-600 hover:text-blue-800 underline;
  }

  .submit-btn {
    @apply inline-flex items-center px-6 py-2 bg-blue-600 text-white font-medium rounded-lg hover:bg-blue-700 transition-colors duration-200 disabled:opacity-50 disabled:cursor-not-allowed;
  }

  .comments-list {
    @apply space-y-6;
  }

  .loading-comments {
    @apply flex items-center gap-2 text-gray-600;
  }

  .no-comments {
    @apply text-gray-600 italic;
  }

  .comment {
    @apply p-4 bg-gray-50 rounded-lg;
  }

  .comment-header {
    @apply mb-3;
  }

  .comment-author {
    @apply flex items-center gap-3;
  }

  .comment-avatar {
    @apply w-8 h-8 rounded-full object-cover;
  }

  .comment-name {
    @apply font-semibold text-gray-900;
  }

  .comment-date {
    @apply text-sm text-gray-600 ml-2;
  }

  .comment-content {
    @apply text-gray-800 mb-3;
  }

  .comment-actions {
    @apply flex gap-4;
  }

  .reply-btn {
    @apply text-sm text-blue-600 hover:text-blue-800 transition-colors duration-200;
  }

  .related-posts {
    @apply bg-white rounded-lg shadow-sm p-6;
  }

  .related-posts h3 {
    @apply text-lg font-semibold text-gray-900 mb-6;
  }

  .related-posts-grid {
    @apply grid grid-cols-1 md:grid-cols-3 gap-6;
  }

  .related-post {
    @apply group;
  }

  .related-post-image {
    @apply aspect-video overflow-hidden rounded-lg mb-3 bg-gray-100;
  }

  .related-post-image img {
    @apply w-full h-full object-cover transition-transform duration-200 group-hover:scale-105;
  }

  .related-post-link {
    @apply text-gray-900 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .related-post-date {
    @apply text-sm text-gray-600;
  }
</style>