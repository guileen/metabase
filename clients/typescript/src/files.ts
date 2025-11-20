import { MetaBaseClient } from './client';
import { APIResponse } from './types';

/**
 * File upload options
 */
export interface FileUploadOptions {
  /** File name (optional, will use filename from file if not provided) */
  filename?: string;
  /** MIME type (optional, will be detected if not provided) */
  mimeType?: string;
  /** Additional metadata */
  metadata?: Record<string, any>;
  /** Public access flag */
  public?: boolean;
  /** Expiration time for temporary files */
  expiresAt?: string;
}

/**
 * File information
 */
export interface FileInfo {
  /** File ID */
  id: string;
  /** Original filename */
  filename: string;
  /** File size in bytes */
  size: number;
  /** MIME type */
  mimeType: string;
  /** File hash */
  hash: string;
  /** Public URL (if public) */
  publicUrl?: string;
  /** Download URL */
  downloadUrl: string;
  /** Metadata */
  metadata?: Record<string, any>;
  /** Created at timestamp */
  createdAt: string;
  /** Updated at timestamp */
  updatedAt: string;
  /** Expires at timestamp */
  expiresAt?: string;
  /** Creator ID */
  createdBy: string;
}

/**
 * File list filter options
 */
export interface FileListOptions {
  /** Search query */
  search?: string;
  /** MIME type filter */
  mimeType?: string;
  /** Public files only */
  public?: boolean;
  /** Created by user */
  createdBy?: string;
  /** Minimum file size */
  minSize?: number;
  /** Maximum file size */
  maxSize?: number;
  /** Created after date */
  createdAfter?: string;
  /** Created before date */
  createdBefore?: string;
  /** Pagination limit */
  limit?: number;
  /** Pagination offset */
  offset?: number;
  /** Sort field */
  sortBy?: 'filename' | 'size' | 'createdAt' | 'updatedAt';
  /** Sort direction */
  sortOrder?: 'asc' | 'desc';
}

/**
 * File manager for handling file uploads and downloads
 */
export class FileManager {
  constructor(private client: MetaBaseClient) {}

  /**
   * Upload a file
   */
  async upload(
    file: File | Blob,
    options?: FileUploadOptions
  ): Promise<APIResponse<FileInfo>> {
    const formData = new FormData();

    // Add file
    if (file instanceof File) {
      formData.append('file', file);
      if (options?.filename) {
        formData.append('filename', options.filename);
      } else {
        formData.append('filename', file.name);
      }
    } else {
      // For Blob objects, filename is required
      if (!options?.filename) {
        return {
          error: {
            code: 'invalid_request',
            message: 'Filename is required when uploading Blob objects',
            timestamp: new Date().toISOString(),
          },
        };
      }
      formData.append('file', file);
      formData.append('filename', options.filename);
    }

    // Add optional parameters
    if (options?.mimeType) {
      formData.append('mime_type', options.mimeType);
    }

    if (options?.metadata) {
      formData.append('metadata', JSON.stringify(options.metadata));
    }

    if (options?.public) {
      formData.append('public', 'true');
    }

    if (options?.expiresAt) {
      formData.append('expires_at', options.expiresAt);
    }

    // Create temporary axios instance for file upload
    const config = this.client.getConfig();
    const axios = require('axios').default;

    try {
      const response = await axios.post(
        `${config.url}/files/v1/upload`,
        formData,
        {
          headers: {
            'Authorization': `Bearer ${config.apiKey}`,
            'Content-Type': 'multipart/form-data',
            ...config.headers,
          },
          timeout: config.timeout || 300000, // 5 minutes for file uploads
        }
      );

      return response.data;
    } catch (error: any) {
      if (error.response) {
        return {
          error: {
            code: error.response.data?.error?.code || 'upload_error',
            message: error.response.data?.error?.message || 'File upload failed',
            details: error.response.data?.error?.details,
            timestamp: new Date().toISOString(),
          },
        };
      } else {
        return {
          error: {
            code: 'network_error',
            message: error.message || 'Network error during file upload',
            timestamp: new Date().toISOString(),
          },
        };
      }
    }
  }

  /**
   * Upload file from URL
   */
  async uploadFromUrl(
    url: string,
    options?: FileUploadOptions & { filename: string }
  ): Promise<APIResponse<FileInfo>> {
    if (!options?.filename) {
      return {
        error: {
          code: 'invalid_request',
          message: 'Filename is required when uploading from URL',
          timestamp: new Date().toISOString(),
        },
      };
    }

    try {
      // First, download the file
      const response = await fetch(url);
      if (!response.ok) {
        return {
          error: {
            code: 'download_error',
            message: `Failed to download file from URL: ${response.statusText}`,
            timestamp: new Date().toISOString(),
          },
        };
      }

      const blob = await response.blob();
      return this.upload(blob, {
        ...options,
        mimeType: options.mimeType || blob.type,
      });
    } catch (error: any) {
      return {
        error: {
          code: 'download_error',
          message: error.message || 'Error downloading file from URL',
          timestamp: new Date().toISOString(),
        },
      };
    }
  }

  /**
   * Get file information
   */
  async getFileInfo(fileId: string): Promise<APIResponse<FileInfo>> {
    return this.client.request<FileInfo>({
      method: 'GET',
      url: `/files/v1/${fileId}`,
    });
  }

  /**
   * List files
   */
  async listFiles(options?: FileListOptions): Promise<APIResponse<{ files: FileInfo[]; total: number }>> {
    const params = new URLSearchParams();

    if (options?.search) {
      params.append('search', options.search);
    }
    if (options?.mimeType) {
      params.append('mime_type', options.mimeType);
    }
    if (options?.public !== undefined) {
      params.append('public', options.public.toString());
    }
    if (options?.createdBy) {
      params.append('created_by', options.createdBy);
    }
    if (options?.minSize) {
      params.append('min_size', options.minSize.toString());
    }
    if (options?.maxSize) {
      params.append('max_size', options.maxSize.toString());
    }
    if (options?.createdAfter) {
      params.append('created_after', options.createdAfter);
    }
    if (options?.createdBefore) {
      params.append('created_before', options.createdBefore);
    }
    if (options?.limit) {
      params.append('limit', options.limit.toString());
    }
    if (options?.offset) {
      params.append('offset', options.offset.toString());
    }
    if (options?.sortBy) {
      params.append('sort_by', options.sortBy);
    }
    if (options?.sortOrder) {
      params.append('sort_order', options.sortOrder);
    }

    const url = params.toString()
      ? `/files/v1?${params.toString()}`
      : '/files/v1';

    return this.client.request({
      method: 'GET',
      url,
    });
  }

  /**
   * Delete a file
   */
  async deleteFile(fileId: string): Promise<APIResponse<void>> {
    return this.client.request({
      method: 'DELETE',
      url: `/files/v1/${fileId}`,
    });
  }

  /**
   * Get download URL for a file
   */
  getDownloadUrl(fileId: string): string {
    const config = this.client.getConfig();
    return `${config.url}/files/v1/${fileId}`;
  }

  /**
   * Download a file
   */
  async download(fileId: string): Promise<APIResponse<Blob>> {
    const config = this.client.getConfig();
    const axios = require('axios').default;

    try {
      const response = await axios.get(
        `${config.url}/files/v1/${fileId}`,
        {
          headers: {
            'Authorization': `Bearer ${config.apiKey}`,
            ...config.headers,
          },
          responseType: 'blob',
          timeout: config.timeout || 30000,
        }
      );

      return {
        data: response.data,
      };
    } catch (error: any) {
      if (error.response) {
        return {
          error: {
            code: error.response.data?.error?.code || 'download_error',
            message: error.response.data?.error?.message || 'File download failed',
            details: error.response.data?.error?.details,
            timestamp: new Date().toISOString(),
          },
        };
      } else {
        return {
          error: {
            code: 'network_error',
            message: error.message || 'Network error during file download',
            timestamp: new Date().toISOString(),
          },
        };
      }
    }
  }

  /**
   * Get file as data URL
   */
  async getDataUrl(fileId: string): Promise<APIResponse<string>> {
    const result = await this.download(fileId);

    if (result.error) {
      return result;
    }

    if (result.data) {
      // Get file info to determine MIME type
      const fileInfo = await this.getFileInfo(fileId);
      const mimeType = fileInfo.data?.mimeType || 'application/octet-stream';

      return new Promise((resolve) => {
        const reader = new FileReader();
        reader.onloadend = () => {
          resolve({
            data: reader.result as string,
          });
        };
        reader.onerror = () => {
          resolve({
            error: {
              code: 'conversion_error',
              message: 'Failed to convert file to data URL',
              timestamp: new Date().toISOString(),
            },
          });
        };
        reader.readAsDataURL(result.data as Blob);
      });
    }

    return result;
  }
}