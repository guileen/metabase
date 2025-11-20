// CMS Types and Interfaces

export interface ContentType {
  id: string;
  tenant_id: string;
  project_id?: string;
  name: string;
  slug: string;
  description?: string;
  icon?: string;
  color?: string;
  is_hierarchical: boolean;
  has_categories: boolean;
  has_tags: boolean;
  has_comments: boolean;
  has_media: boolean;
  has_ratings: boolean;
  has_workflow: boolean;
  auto_publish: boolean;
  has_seo: boolean;
  default_meta_title?: string;
  default_meta_description?: string;
  status: 'active' | 'inactive' | 'archived';
  created_at: string;
  updated_at: string;
  created_by: string;
  updated_by?: string;
}

export interface ContentField {
  id: string;
  content_type_id: string;
  name: string;
  slug: string;
  type: 'text' | 'textarea' | 'rich_text' | 'number' | 'decimal' | 'boolean' |
        'date' | 'datetime' | 'email' | 'url' | 'image' | 'file' | 'select' |
        'multiselect' | 'relationship' | 'json';
  required: boolean;
  default_value?: string;
  options?: Record<string, any>;
  validation?: Record<string, any>;
  placeholder?: string;
  help_text?: string;
  order_index: number;
  is_filterable: boolean;
  is_searchable: boolean;
  created_at: string;
  updated_at: string;
}

export interface Category {
  id: string;
  tenant_id: string;
  project_id?: string;
  content_type_id?: string;
  name: string;
  slug: string;
  description?: string;
  image_id?: string;
  color?: string;
  parent_id?: string;
  order_index: number;
  meta_title?: string;
  meta_description?: string;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
  created_by: string;
  updated_by?: string;
  children?: Category[]; // For hierarchical display
  post_count?: number; // Computed field
}

export interface Tag {
  id: string;
  tenant_id: string;
  project_id?: string;
  name: string;
  slug: string;
  description?: string;
  color?: string;
  usage_count: number;
  status: 'active' | 'inactive';
  created_at: string;
  updated_at: string;
}

export interface Content {
  id: string;
  tenant_id: string;
  project_id?: string;
  content_type_id: string;
  title: string;
  slug: string;
  excerpt?: string;
  content?: string;
  status: 'draft' | 'review' | 'published' | 'scheduled' | 'archived';
  featured: boolean;
  sticky: boolean;
  author_id: string;
  created_by: string;
  updated_by?: string;
  published_at?: string;
  expires_at?: string;
  meta_title?: string;
  meta_description?: string;
  meta_keywords?: string;
  og_image_id?: string;
  custom_fields?: Record<string, any>;
  view_count: number;
  like_count: number;
  comment_count: number;
  created_at: string;
  updated_at: string;

  // Expanded relationships
  content_type?: ContentType;
  author?: any; // User object
  categories?: Category[];
  tags?: Tag[];
  comments?: Comment[];
  featured_image?: any; // File object
}

export interface Comment {
  id: string;
  tenant_id: string;
  project_id?: string;
  content_id: string;
  author_name: string;
  author_email?: string;
  author_id?: string;
  content: string;
  parent_id?: string;
  thread_id?: string;
  status: 'pending' | 'approved' | 'rejected' | 'spam';
  ip_address?: string;
  created_at: string;
  updated_at: string;

  // Expanded relationships
  author?: any; // User object
  replies?: Comment[];
}

export interface Media {
  id: string;
  tenant_id: string;
  project_id?: string;
  file_id: string;
  title?: string;
  description?: string;
  alt_text?: string;
  caption?: string;
  file_type: string;
  file_size: number;
  width?: number;
  height?: number;
  duration?: number;
  folder_path?: string;
  meta_title?: string;
  meta_description?: string;
  created_at: string;
  updated_at: string;
  created_by: string;
  updated_by?: string;

  // Expanded relationships
  file?: any; // File object
  created_by_user?: any; // User object
}

export interface CMSSetting {
  id: string;
  tenant_id: string;
  project_id?: string;
  key: string;
  value: any;
  description?: string;
  type: 'string' | 'number' | 'boolean' | 'json' | 'array';
  category: string;
  created_at: string;
  updated_at: string;
}

// Form types
export interface CreateContentRequest {
  content_type_id: string;
  title: string;
  slug?: string;
  excerpt?: string;
  content?: string;
  status?: Content['status'];
  featured?: boolean;
  sticky?: boolean;
  published_at?: string;
  expires_at?: string;
  meta_title?: string;
  meta_description?: string;
  meta_keywords?: string;
  og_image_id?: string;
  custom_fields?: Record<string, any>;
  category_ids?: string[];
  tag_ids?: string[];
}

export interface UpdateContentRequest extends Partial<CreateContentRequest> {}

export interface CreateCommentRequest {
  content_id: string;
  author_name: string;
  author_email?: string;
  author_id?: string;
  content: string;
  parent_id?: string;
}

export interface SearchFilters {
  query?: string;
  content_type_id?: string;
  category_id?: string;
  tag_id?: string;
  author_id?: string;
  status?: Content['status'];
  featured?: boolean;
  sticky?: boolean;
  date_from?: string;
  date_to?: string;
  sort_by?: 'created_at' | 'updated_at' | 'published_at' | 'title' | 'view_count' | 'like_count';
  sort_order?: 'asc' | 'desc';
  limit?: number;
  offset?: number;
}

export interface ContentListResponse {
  items: Content[];
  total: number;
  limit: number;
  offset: number;
  has_next: boolean;
}

// CMS Configuration
export interface CMSConfig {
  tenant_id: string;
  project_id?: string;
  api_base_url: string;
  api_key: string;
  theme?: {
    primary_color: string;
    secondary_color: string;
    font_family: string;
    header_style: 'default' | 'minimal' | 'bold';
  };
  features: {
    comments: boolean;
    ratings: boolean;
    search: boolean;
    categories: boolean;
    tags: boolean;
    media_library: boolean;
    seo: boolean;
  };
  routing: {
    base_path: string;
    content_patterns: Record<string, string>; // content_type_slug -> url pattern
  };
}

// Site settings
export interface SiteSettings {
  site_title: string;
  site_description: string;
  site_url: string;
  timezone: string;
  language: string;
  blog_posts_per_page: number;
  blog_excerpt_length: number;
  blog_show_author: boolean;
  blog_show_date: boolean;
  blog_allow_comments: boolean;
  comments_require_approval: boolean;
  comments_allow_guest: boolean;
  comments_max_length: number;
  seo_default_title_length: number;
  seo_default_description_length: number;
  seo_auto_generate: boolean;
}