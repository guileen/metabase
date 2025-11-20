import { CMSConfig } from '../types/cms';

/**
 * CMS Configuration Manager
 * Handles configuration and initialization of the CMS system
 */
export class CMSConfigManager {
  private static config: CMSConfig | null = null;
  private static isInitialized = false;

  /**
   * Initialize CMS configuration
   */
  static async initialize(): Promise<CMSConfig> {
    if (this.isInitialized) {
      return this.config!;
    }

    try {
      // Load configuration from environment variables or window.METABASE_CMS
      const config = this.loadConfig();

      // Validate configuration
      this.validateConfig(config);

      this.config = config;
      this.isInitialized = true;

      return config;
    } catch (error) {
      console.error('Failed to initialize CMS configuration:', error);
      throw error;
    }
  }

  /**
   * Get current configuration
   */
  static getConfig(): CMSConfig {
    if (!this.isInitialized) {
      throw new Error('CMS configuration not initialized. Call initialize() first.');
    }

    return this.config!;
  }

  /**
   * Load configuration from various sources
   */
  private static loadConfig(): CMSConfig {
    // Try to get config from window object (for client-side)
    if (typeof window !== 'undefined' && (window as any).METABASE_CMS) {
      return (window as any).METABASE_CMS as CMSConfig;
    }

    // Try environment variables (for server-side)
    if (typeof process !== 'undefined' && process.env) {
      return {
        tenant_id: process.env.METABASE_TENANT_ID || '',
        project_id: process.env.METABASE_PROJECT_ID,
        api_base_url: process.env.METABASE_API_URL || 'http://localhost:7609',
        api_key: process.env.METABASE_API_KEY || '',
        theme: {
          primary_color: process.env.CMS_THEME_PRIMARY_COLOR || '#3b82f6',
          secondary_color: process.env.CMS_THEME_SECONDARY_COLOR || '#64748b',
          font_family: process.env.CMS_THEME_FONT_FAMILY || 'system-ui, -apple-system, sans-serif',
          header_style: (process.env.CMS_THEME_HEADER_STYLE as any) || 'default',
        },
        features: {
          comments: process.env.CMS_FEATURES_COMMENTS === 'true',
          ratings: process.env.CMS_FEATURES_RATINGS === 'true',
          search: process.env.CMS_FEATURES_SEARCH !== 'false', // Default true
          categories: process.env.CMS_FEATURES_CATEGORIES !== 'false', // Default true
          tags: process.env.CMS_FEATURES_TAGS !== 'false', // Default true
          media_library: process.env.CMS_FEATURES_MEDIA_LIBRARY !== 'false', // Default true
          seo: process.env.CMS_FEATURES_SEO !== 'false', // Default true
        },
        routing: {
          base_path: process.env.CMS_ROUTING_BASE_PATH || '',
          content_patterns: this.parseContentPatterns(process.env.CMS_ROUTING_PATTERNS),
        },
      };
    }

    // Default configuration for development
    return {
      tenant_id: '00000000-0000-0000-0000-000000000001',
      api_base_url: 'http://localhost:7609',
      api_key: 'metabase_sys_demo123_example_key_here',
      theme: {
        primary_color: '#3b82f6',
        secondary_color: '#64748b',
        font_family: 'system-ui, -apple-system, sans-serif',
        header_style: 'default',
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
      routing: {
        base_path: '',
        content_patterns: {
          'blog-posts': '/blog/{slug}',
          'pages': '/{slug}',
          'forum-topics': '/forum/{slug}',
        },
      },
    };
  }

  /**
   * Parse content patterns from environment variable
   */
  private static parseContentPatterns(patterns?: string): Record<string, string> {
    if (!patterns) {
      return {
        'blog-posts': '/blog/{slug}',
        'pages': '/{slug}',
        'forum-topics': '/forum/{slug}',
      };
    }

    try {
      return JSON.parse(patterns);
    } catch {
      console.warn('Invalid CMS_ROUTING_PATTERNS format, using defaults');
      return {
        'blog-posts': '/blog/{slug}',
        'pages': '/{slug}',
        'forum-topics': '/forum/{slug}',
      };
    }
  }

  /**
   * Validate configuration
   */
  private static validateConfig(config: CMSConfig): void {
    const errors: string[] = [];

    if (!config.tenant_id) {
      errors.push('tenant_id is required');
    }

    if (!config.api_base_url) {
      errors.push('api_base_url is required');
    }

    if (!config.api_key) {
      errors.push('api_key is required');
    }

    if (!config.api_base_url.startsWith('http://') && !config.api_base_url.startsWith('https://')) {
      errors.push('api_base_url must start with http:// or https://');
    }

    if (errors.length > 0) {
      throw new Error(`CMS configuration validation failed:\n${errors.join('\n')}`);
    }
  }

  /**
   * Update configuration (for runtime updates)
   */
  static updateConfig(updates: Partial<CMSConfig>): void {
    if (!this.isInitialized) {
      throw new Error('CMS configuration not initialized. Call initialize() first.');
    }

    this.config = {
      ...this.config!,
      ...updates,
    };

    // Validate the updated configuration
    this.validateConfig(this.config);
  }

  /**
   * Reset configuration (for testing)
   */
  static reset(): void {
    this.config = null;
    this.isInitialized = false;
  }

  /**
   * Generate CSS variables from theme configuration
   */
  static generateThemeCSS(): string {
    if (!this.isInitialized) {
      return '';
    }

    const theme = this.config!.theme;

    return `
      :root {
        --cms-primary-color: ${theme.primary_color};
        --cms-secondary-color: ${theme.secondary_color};
        --cms-font-family: ${theme.font_family};
        --cms-header-style: ${theme.header_style};
      }
    `;
  }

  /**
   * Check if a feature is enabled
   */
  static isFeatureEnabled(feature: keyof CMSConfig['features']): boolean {
    if (!this.isInitialized) {
      return false;
    }

    return this.config!.features[feature] || false;
  }

  /**
   * Get URL for a content item based on routing patterns
   */
  static getContentUrl(contentTypeSlug: string, slug: string): string {
    if (!this.isInitialized) {
      return `/${contentTypeSlug}/${slug}`;
    }

    const patterns = this.config!.routing.content_patterns;
    const pattern = patterns[contentTypeSlug] || '/{slug}';
    const basePath = this.config!.routing.base_path;

    const url = pattern.replace('{slug}', slug);
    return basePath ? `${basePath}${url}` : url;
  }

  /**
   * Get theme configuration
   */
  static getTheme() {
    if (!this.isInitialized) {
      throw new Error('CMS configuration not initialized');
    }

    return this.config!.theme;
  }

  /**
   * Get feature flags
   */
  static getFeatures() {
    if (!this.isInitialized) {
      throw new Error('CMS configuration not initialized');
    }

    return this.config!.features;
  }
}