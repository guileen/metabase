import { CMSConfig, Content, ContentType, Category, Tag, Media, Comment, CreateContentRequest, UpdateContentRequest, CreateCommentRequest, SearchFilters, ContentListResponse, SiteSettings } from '../types/cms';

/**
 * CMS API Client
 * Simplified API client for web frontend CMS functionality
 */
export class CMSApiClient {
  private config: CMSConfig;
  private baseUrl: string;

  constructor(config: CMSConfig) {
    this.config = config;
    this.baseUrl = config.api_base_url.replace(/\/$/, ''); // Remove trailing slash
  }

  private async request<T>(endpoint: string, options: RequestInit = {}): Promise<T> {
    const url = `${this.baseUrl}${endpoint}`;

    const defaultHeaders = {
      'Authorization': `Bearer ${this.config.api_key}`,
      'Content-Type': 'application/json',
      'X-Tenant-ID': this.config.tenant_id,
      ...(this.config.project_id && { 'X-Project-ID': this.config.project_id }),
    };

    const response = await fetch(url, {
      ...options,
      headers: {
        ...defaultHeaders,
        ...options.headers,
      },
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || `HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  // Content Type operations
  async getContentTypes(): Promise<ContentType[]> {
    return this.request<ContentType[]>('/api/v1/cms/content-types');
  }

  async getContentType(slug: string): Promise<ContentType> {
    return this.request<ContentType>(`/api/v1/cms/content-types/${slug}`);
  }

  // Content operations
  async getContent(filters: SearchFilters = {}): Promise<ContentListResponse> {
    const params = new URLSearchParams();

    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        params.append(key, String(value));
      }
    });

    const query = params.toString();
    const endpoint = query ? `/api/v1/cms/content?${query}` : '/api/v1/cms/content';

    return this.request<ContentListResponse>(endpoint);
  }

  async getContentBySlug(slug: string, contentTypeSlug?: string): Promise<Content> {
    const params = new URLSearchParams();
    if (contentTypeSlug) {
      params.append('content_type', contentTypeSlug);
    }

    const query = params.toString();
    const endpoint = query ? `/api/v1/cms/content/slug/${slug}?${query}` : `/api/v1/cms/content/slug/${slug}`;

    return this.request<Content>(endpoint);
  }

  async getContentById(id: string): Promise<Content> {
    return this.request<Content>(`/api/v1/cms/content/${id}`);
  }

  async createContent(data: CreateContentRequest): Promise<Content> {
    return this.request<Content>('/api/v1/cms/content', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  async updateContent(id: string, data: UpdateContentRequest): Promise<Content> {
    return this.request<Content>(`/api/v1/cms/content/${id}`, {
      method: 'PATCH',
      body: JSON.stringify(data),
    });
  }

  async deleteContent(id: string): Promise<void> {
    return this.request<void>(`/api/v1/cms/content/${id}`, {
      method: 'DELETE',
    });
  }

  async incrementViewCount(id: string): Promise<void> {
    return this.request<void>(`/api/v1/cms/content/${id}/view`, {
      method: 'POST',
    });
  }

  async likeContent(id: string): Promise<void> {
    return this.request<void>(`/api/v1/cms/content/${id}/like`, {
      method: 'POST',
    });
  }

  // Category operations
  async getCategories(contentTypeId?: string): Promise<Category[]> {
    const params = contentTypeId ? `?content_type_id=${contentTypeId}` : '';
    return this.request<Category[]>(`/api/v1/cms/categories${params}`);
  }

  async getCategoryBySlug(slug: string): Promise<Category> {
    return this.request<Category>(`/api/v1/cms/categories/slug/${slug}`);
  }

  // Tag operations
  async getTags(popular = true): Promise<Tag[]> {
    const params = popular ? '?popular=true' : '';
    return this.request<Tag[]>(`/api/v1/cms/tags${params}`);
  }

  async getTagBySlug(slug: string): Promise<Tag> {
    return this.request<Tag>(`/api/v1/cms/tags/slug/${slug}`);
  }

  // Comment operations
  async getComments(contentId: string, status: 'approved' | 'all' = 'approved'): Promise<Comment[]> {
    const params = status !== 'approved' ? `?status=${status}` : '';
    return this.request<Comment[]>(`/api/v1/cms/content/${contentId}/comments${params}`);
  }

  async createComment(data: CreateCommentRequest): Promise<Comment> {
    return this.request<Comment>('/api/v1/cms/comments', {
      method: 'POST',
      body: JSON.stringify(data),
    });
  }

  // Media operations
  async getMedia(filters: { folder?: string; type?: string; limit?: number } = {}): Promise<Media[]> {
    const params = new URLSearchParams();
    Object.entries(filters).forEach(([key, value]) => {
      if (value !== undefined && value !== null) {
        params.append(key, String(value));
      }
    });

    const query = params.toString();
    const endpoint = query ? `/api/v1/cms/media?${query}` : '/api/v1/cms/media';

    return this.request<Media[]>(endpoint);
  }

  async uploadMedia(file: File, metadata?: { title?: string; description?: string; alt_text?: string }): Promise<Media> {
    const formData = new FormData();
    formData.append('file', file);

    if (metadata) {
      Object.entries(metadata).forEach(([key, value]) => {
        if (value !== undefined && value !== null) {
          formData.append(key, String(value));
        }
      });
    }

    const response = await fetch(`${this.baseUrl}/api/v1/cms/media/upload`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.config.api_key}`,
        'X-Tenant-ID': this.config.tenant_id,
        ...(this.config.project_id && { 'X-Project-ID': this.config.project_id }),
      },
      body: formData,
    });

    if (!response.ok) {
      const errorData = await response.json().catch(() => ({}));
      throw new Error(errorData.error?.message || `HTTP ${response.status}: ${response.statusText}`);
    }

    return response.json();
  }

  // Settings operations
  async getSiteSettings(): Promise<SiteSettings> {
    return this.request<SiteSettings>('/api/v1/cms/settings/site');
  }

  async updateSiteSettings(settings: Partial<SiteSettings>): Promise<SiteSettings> {
    return this.request<SiteSettings>('/api/v1/cms/settings/site', {
      method: 'PATCH',
      body: JSON.stringify(settings),
    });
  }

  // Search operations
  async search(query: string, filters: Omit<SearchFilters, 'query'> = {}): Promise<ContentListResponse> {
    return this.getContent({
      ...filters,
      query,
    });
  }

  // Utility methods
  getPublicUrl(content: Content): string {
    const contentTypeSlug = content.content_type?.slug || 'content';
    return `${this.config.routing.base_path}/${contentTypeSlug}/${content.slug}`;
  }

  getMediaUrl(media: Media): string {
    // This should return the public URL for the media file
    return `${this.baseUrl}/api/v1/cms/media/${media.id}/download`;
  }

  getCategoryUrl(category: Category): string {
    return `${this.config.routing.base_path}/category/${category.slug}`;
  }

  getTagUrl(tag: Tag): string {
    return `${this.config.routing.base_path}/tag/${tag.slug}`;
  }

  getAuthorUrl(authorId: string): string {
    return `${this.config.routing.base_path}/author/${authorId}`;
  }

  formatDate(dateString: string, format: 'short' | 'long' | 'relative' = 'short'): string {
    const date = new Date(dateString);

    switch (format) {
      case 'short':
        return date.toLocaleDateString('en-US', {
          year: 'numeric',
          month: 'short',
          day: 'numeric',
        });
      case 'long':
        return date.toLocaleDateString('en-US', {
          year: 'numeric',
          month: 'long',
          day: 'numeric',
        });
      case 'relative':
        const now = new Date();
        const diffMs = now.getTime() - date.getTime();
        const diffDays = Math.floor(diffMs / (1000 * 60 * 60 * 24));

        if (diffDays === 0) {
          return 'Today';
        } else if (diffDays === 1) {
          return 'Yesterday';
        } else if (diffDays < 7) {
          return `${diffDays} days ago`;
        } else if (diffDays < 30) {
          const weeks = Math.floor(diffDays / 7);
          return `${weeks} week${weeks > 1 ? 's' : ''} ago`;
        } else if (diffDays < 365) {
          const months = Math.floor(diffDays / 30);
          return `${months} month${months > 1 ? 's' : ''} ago`;
        } else {
          const years = Math.floor(diffDays / 365);
          return `${years} year${years > 1 ? 's' : ''} ago`;
        }
      default:
        return date.toISOString();
    }
  }

  getReadingTime(content: string): string {
    const wordsPerMinute = 200; // Average reading speed
    const words = content.trim().split(/\s+/).length;
    const minutes = Math.ceil(words / wordsPerMinute);
    return `${minutes} min read`;
  }

  getExcerpt(content: string, maxLength: number = 200): string {
    const plainText = content.replace(/<[^>]*>/g, ''); // Remove HTML tags
    if (plainText.length <= maxLength) {
      return plainText;
    }
    return plainText.substring(0, maxLength).trim() + '...';
  }
}