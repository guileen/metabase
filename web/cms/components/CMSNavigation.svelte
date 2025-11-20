<script lang="ts">
  import { createEventDispatcher } from 'svelte';
  import type { ContentType, Category } from '../types/cms';

  export let currentPath = '';
  export let contentTypes: ContentType[] = [];
  export let categories: Category[] = [];

  const dispatch = createEventDispatcher();

  function navigate(path: string) {
    dispatch('navigate', { path });
  }

  function toggleMobileMenu() {
    dispatch('toggle-mobile-menu');
  }

  // Get content type icon
  function getContentTypeIcon(slug: string): string {
    const icons: Record<string, string> = {
      'blog-posts': '📝',
      'pages': '📄',
      'forum-topics': '💬',
      'news': '📰',
      'events': '📅',
      'products': '🛍️',
      'portfolio': '💼',
      'testimonials': '⭐',
    };
    return icons[slug] || '📄';
  }

  // Get navigation items from content types
  $: navItems = contentTypes.map(type => ({
    slug: type.slug,
    name: type.name,
    icon: type.icon || getContentTypeIcon(type.slug),
    color: type.color || '#3b82f6',
    hasCategories: type.has_categories,
  }));
</script>

<nav class="cms-navigation">
  <!-- Desktop Navigation -->
  <div class="nav-desktop">
    <div class="nav-brand">
      <a href="/" class="brand-link" on:click={() => navigate('/')}>
        🏠 Home
      </a>
    </div>

    <div class="nav-menu">
      {#each navItems as item}
        <div class="nav-item">
          <a
            href="/{item.slug}"
            class="nav-link {currentPath?.startsWith(`/${item.slug}`) ? 'active' : ''}"
            on:click={() => navigate(`/${item.slug}`)}
          >
            <span class="nav-icon">{item.icon}</span>
            <span class="nav-text">{item.name}</span>
          </a>

          <!-- Dropdown for categories -->
          {#if item.hasCategories}
            <div class="nav-dropdown">
              {#each categories.filter(c => !c.content_type_id || c.content_type_id === item.slug) as category}
                <a
                  href="/{item.slug}/category/{category.slug}"
                  class="dropdown-item"
                  on:click={() => navigate(`/${item.slug}/category/${category.slug}`)}
                >
                  {#if category.color}
                    <span
                      class="category-dot"
                      style="background-color: {category.color}"
                    ></span>
                  {/if}
                  {category.name}
                </a>
              {/each}
            </div>
          {/if}
        </div>
      {/each}

      <!-- Additional navigation items -->
      <div class="nav-item">
        <a
          href="/search"
          class="nav-link {currentPath === '/search' ? 'active' : ''}"
          on:click={() => navigate('/search')}
        >
          <span class="nav-icon">🔍</span>
          <span class="nav-text">Search</span>
        </a>
      </div>

      <div class="nav-item">
        <a
          href="/contact"
          class="nav-link {currentPath === '/contact' ? 'active' : ''}"
          on:click={() => navigate('/contact')}
        >
          <span class="nav-icon">📧</span>
          <span class="nav-text">Contact</span>
        </a>
      </div>
    </div>
  </div>

  <!-- Mobile Navigation -->
  <div class="nav-mobile">
    <div class="mobile-header">
      <div class="nav-brand">
        <a href="/" class="brand-link" on:click={() => navigate('/')}>
          🏠 Home
        </a>
      </div>

      <button
        type="button"
        class="mobile-menu-toggle"
        on:click={toggleMobileMenu}
        aria-label="Toggle menu"
      >
        ☰
      </button>
    </div>

    <!-- Mobile menu (hidden by default) -->
    <div class="mobile-menu" class:hidden={true}>
      <div class="mobile-menu-content">
        {#each navItems as item}
          <div class="mobile-nav-item">
            <a
              href="/{item.slug}"
              class="mobile-nav-link {currentPath?.startsWith(`/${item.slug}`) ? 'active' : ''}"
              on:click={() => navigate(`/${item.slug}`)}
            >
              <span class="mobile-nav-icon">{item.icon}</span>
              <span class="mobile-nav-text">{item.name}</span>
            </a>

            <!-- Mobile categories -->
            {#if item.hasCategories}
              <div class="mobile-categories">
                {#each categories.filter(c => !c.content_type_id || c.content_type_id === item.slug) as category}
                  <a
                    href="/{item.slug}/category/{category.slug}"
                    class="mobile-category-link"
                    on:click={() => navigate(`/${item.slug}/category/${category.slug}`)}
                  >
                    {#if category.color}
                      <span
                        class="category-dot"
                        style="background-color: {category.color}"
                      ></span>
                    {/if}
                    {category.name}
                  </a>
                {/each}
              </div>
            {/if}
          </div>
        {/each}

        <div class="mobile-nav-divider"></div>

        <div class="mobile-nav-item">
          <a
            href="/search"
            class="mobile-nav-link {currentPath === '/search' ? 'active' : ''}"
            on:click={() => navigate('/search')}
          >
            <span class="mobile-nav-icon">🔍</span>
            <span class="mobile-nav-text">Search</span>
          </a>
        </div>

        <div class="mobile-nav-item">
          <a
            href="/contact"
            class="mobile-nav-link {currentPath === '/contact' ? 'active' : ''}"
            on:click={() => navigate('/contact')}
          >
            <span class="mobile-nav-icon">📧</span>
            <span class="mobile-nav-text">Contact</span>
          </a>
        </div>
      </div>
    </div>
  </div>
</nav>

<style>
  .cms-navigation {
    @apply bg-white border-b border-gray-200 sticky top-0 z-50;
  }

  /* Desktop Navigation */
  .nav-desktop {
    @apply hidden lg:block;
  }

  .nav-brand {
    @apply inline-block;
  }

  .brand-link {
    @apply inline-flex items-center px-4 py-4 text-lg font-bold text-gray-900 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .nav-menu {
    @apply flex flex-wrap items-center;
  }

  .nav-item {
    @apply relative group;
  }

  .nav-link {
    @apply inline-flex items-center px-4 py-4 text-gray-700 hover:text-blue-600 transition-colors duration-200 no-underline border-b-2 border-transparent hover:border-blue-600;
  }

  .nav-link.active {
    @apply text-blue-600 border-blue-600;
  }

  .nav-icon {
    @apply mr-2 text-lg;
  }

  .nav-text {
    @apply font-medium;
  }

  /* Dropdown Menu */
  .nav-dropdown {
    @apply absolute left-0 top-full min-w-[200px] bg-white border border-gray-200 rounded-lg shadow-lg opacity-0 invisible group-hover:opacity-100 group-hover:visible transition-all duration-200;
  }

  .dropdown-item {
    @apply block px-4 py-2 text-gray-700 hover:bg-gray-50 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .dropdown-item:first-child {
    @apply rounded-t-lg;
  }

  .dropdown-item:last-child {
    @apply rounded-b-lg;
  }

  .category-dot {
    @apply inline-block w-2 h-2 rounded-full mr-2;
  }

  /* Mobile Navigation */
  .nav-mobile {
    @apply lg:hidden;
  }

  .mobile-header {
    @apply flex items-center justify-between px-4 py-3 border-b border-gray-200;
  }

  .mobile-menu-toggle {
    @apply p-2 text-gray-700 hover:text-blue-600 transition-colors duration-200;
    font-size: 1.25rem;
  }

  .mobile-menu {
    @apply border-b border-gray-200;
  }

  .mobile-menu-content {
    @apply px-4 py-2;
  }

  .mobile-nav-item {
    @apply border-b border-gray-100 last:border-b-0;
  }

  .mobile-nav-link {
    @apply flex items-center px-3 py-3 text-gray-700 hover:text-blue-600 hover:bg-gray-50 transition-colors duration-200 no-underline;
  }

  .mobile-nav-link.active {
    @apply text-blue-600 bg-blue-50;
  }

  .mobile-nav-icon {
    @apply mr-3 text-lg;
  }

  .mobile-nav-text {
    @apply font-medium;
  }

  .mobile-categories {
    @apply pl-10 pr-3 py-2 bg-gray-50;
  }

  .mobile-category-link {
    @apply block px-3 py-2 text-sm text-gray-600 hover:text-blue-600 transition-colors duration-200 no-underline;
  }

  .mobile-nav-divider {
    @apply h-px bg-gray-200 my-2;
  }

  /* Responsive adjustments */
  @media (max-width: 1024px) {
    .nav-desktop {
      @apply hidden;
    }

    .nav-mobile {
      @apply block;
    }
  }

  /* Animation for mobile menu */
  .mobile-menu[class*="hidden"] {
    @apply hidden;
  }

  .mobile-menu:not([class*="hidden"]) {
    @apply block;
  }
</style>