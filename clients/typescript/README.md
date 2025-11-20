# MetaBase TypeScript/JavaScript Client SDK

Official TypeScript client library for the MetaBase API, providing a complete interface for database operations, real-time subscriptions, file management, and authentication.

## Installation

```bash
npm install @metabase/client
# or
yarn add @metabase/client
# or
pnpm add @metabase/client
```

## Quick Start

```typescript
import { createMetaBaseClient } from '@metabase/client';

// Create client with configuration
const metabase = createMetaBaseClient({
  url: 'https://your-metabase-instance.com',
  apiKey: 'your-api-key-here',
});

// Use the client
async function example() {
  // Health check
  const health = await metabase.client.health();
  console.log('Health:', health.data);

  // Query data
  const users = await metabase.client.query('users', {
    select: 'id,email,created_at',
    where: { status: 'active' },
    order: 'created_at DESC',
    limit: 10
  });
  console.log('Users:', users.data);

  // Real-time subscription
  const subscription = await metabase.realtime.subscribe(
    'posts',
    (event) => {
      console.log('Real-time event:', event);
    },
    { type: 'published' }
  );

  // File upload
  const fileInfo = await metabase.files.upload(file, {
    filename: 'profile.jpg',
    public: true
  });
  console.log('File uploaded:', fileInfo.data);
}

example().catch(console.error);
```

## Core Features

### Database Operations

```typescript
// Query with filtering
const posts = await metabase.client.query('posts', {
  select: 'title,content,author_id',
  where: {
    published: true,
    category: 'tech'
  },
  order: 'published_at DESC',
  limit: 20
});

// Get single record
const post = await metabase.client.get('posts', 123, {
  select: 'title,content,published_at'
});

// Insert new record
const newPost = await metabase.client.insert('posts', {
  title: 'My New Post',
  content: 'This is the content...',
  author_id: 456,
  published: false
}, {
  returning: 'id,title,created_at'
});

// Update record
const updated = await metabase.client.updateOne('posts', 123, {
  title: 'Updated Title',
  content: 'Updated content'
}, {
  returning: 'id,updated_at'
});

// Delete record
const deleted = await metabase.client.deleteOne('posts', 123, {
  returning: 'id'
});
```

### Authentication & API Key Management

```typescript
// Create new API key
const newKey = await metabase.auth.createKey({
  name: 'My Service Key',
  type: 'service',
  scopes: ['read', 'write', 'table:read', 'table:update'],
  expires_at: '2024-12-31T23:59:59Z'
});

// List API keys
const keys = await metabase.auth.listKeys({
  type: 'service',
  status: 'active',
  limit: 10
});

// Get key usage stats
const stats = await metabase.auth.getKeyStats('key-id-here');

// Revoke key
await metabase.auth.revokeKey('key-id-here');

// Utility functions
const isValid = AuthManager.validateKeyFormat('metabase_svc_abc123_...');
const keyType = AuthManager.extractKeyType('metabase_sys_def456_...');
```

### Real-time Subscriptions

```typescript
// Subscribe to table changes
const subscription = await metabase.realtime.subscribe(
  'posts',
  (event) => {
    switch (event.type) {
      case 'INSERT':
        console.log('New post:', event.new);
        break;
      case 'UPDATE':
        console.log('Post updated:', event.new);
        break;
      case 'DELETE':
        console.log('Post deleted:', event.old);
        break;
    }
  },
  {
    // Optional filter for specific events
    type: 'published',
    category: 'tech'
  }
);

// Check subscription status
const status = metabase.realtime.getConnectionStatus(subscription.id);
console.log('Status:', status); // 'connected' | 'disconnected' | 'reconnecting'

// Send message (for bi-directional communication)
metabase.realtime.sendMessage(subscription.id, {
  type: 'ping',
  timestamp: new Date().toISOString()
});

// Unsubscribe
metabase.realtime.unsubscribe(subscription.id);

// Unsubscribe from all
metabase.realtime.unsubscribeAll();
```

### File Management

```typescript
// Upload file
const file = document.querySelector('input[type="file"]').files[0];
const uploaded = await metabase.files.upload(file, {
  filename: 'document.pdf',
  metadata: {
    category: 'documents',
    project: 'website-redesign'
  },
  public: true
});

// Upload from URL
const uploadedFromUrl = await metabase.files.uploadFromUrl(
  'https://example.com/image.jpg',
  {
    filename: 'banner.jpg',
    mimeType: 'image/jpeg',
    public: true
  }
);

// List files
const files = await metabase.files.listFiles({
  search: 'document',
  mimeType: 'application/pdf',
  public: true,
  limit: 20,
  sortBy: 'createdAt',
  sortOrder: 'desc'
});

// Get file info
const fileInfo = await metabase.files.getFileInfo('file-id-here');

// Download file
const blob = await metabase.files.download('file-id-here');
const url = URL.createObjectURL(blob);

// Get data URL
const dataUrl = await metabase.files.getDataUrl('file-id-here');

// Get download URL
const downloadUrl = metabase.files.getDownloadUrl('file-id-here');

// Delete file
await metabase.files.deleteFile('file-id-here');
```

## Configuration Options

```typescript
const client = createMetaBaseClient({
  url: 'https://your-metabase-instance.com',
  apiKey: 'your-api-key-here',
  timeout: 30000, // Request timeout in ms
  headers: {
    'X-Custom-Header': 'value'
  },
  debug: true // Enable debug logging
});
```

## Error Handling

The SDK provides consistent error handling across all operations:

```typescript
const result = await metabase.client.query('posts');

if (result.error) {
  console.error('Error:', {
    code: result.error.code,
    message: result.error.message,
    details: result.error.details,
    timestamp: result.error.timestamp
  });
} else {
  console.log('Success:', result.data);
}
```

## API Key Security

- Store API keys securely (environment variables, secret management)
- Use appropriate scopes for minimum privilege
- Implement key rotation regularly
- Monitor usage with `getKeyStats()`
- Revoke compromised keys immediately

## Browser Usage

```html
<script src="https://unpkg.com/@metabase/client@latest/dist/index.umd.min.js"></script>
<script>
  const { MetaBaseClient } = MetaBaseClient;

  const client = new MetaBaseClient({
    url: 'https://your-metabase-instance.com',
    apiKey: 'your-api-key'
  });
</script>
```

## Node.js Usage

```typescript
import { createMetaBaseClient } from '@metabase/client';

// Or CommonJS
const { createMetaBaseClient } = require('@metabase/client');

const metabase = createMetaBaseClient({
  url: process.env.METABASE_URL,
  apiKey: process.env.METABASE_API_KEY
});
```

## TypeScript Support

Full TypeScript support with comprehensive type definitions:

```typescript
import { MetaBaseClient, APIResponse, FileInfo } from '@metabase/client';

const client = new MetaBaseClient({
  url: 'https://example.com',
  apiKey: 'key'
});

const result: APIResponse<FileInfo[]> = await client.query('files');
```

## React Integration Example

```tsx
import React, { useState, useEffect } from 'react';
import { createMetaBaseClient } from '@metabase/client';

const metabase = createMetaBaseClient({
  url: process.env.REACT_APP_METABASE_URL!,
  apiKey: process.env.REACT_APP_METABASE_API_KEY!
});

function PostsList() {
  const [posts, setPosts] = useState([]);
  const [loading, setLoading] = useState(true);

  useEffect(() => {
    async function loadPosts() {
      const result = await metabase.client.query('posts', {
        where: { published: true },
        order: 'published_at DESC',
        limit: 10
      });

      if (result.data) {
        setPosts(result.data);
      }
      setLoading(false);
    }

    loadPosts();

    // Set up real-time subscription
    const subscription = metabase.realtime.subscribe('posts', (event) => {
      if (event.type === 'INSERT' && event.new?.published) {
        setPosts(prev => [event.new, ...prev.slice(0, 9)]);
      }
    });

    return () => {
      metabase.realtime.unsubscribe(subscription.id);
    };
  }, []);

  if (loading) return <div>Loading...</div>;

  return (
    <div>
      {posts.map((post: any) => (
        <div key={post.id}>
          <h3>{post.title}</h3>
          <p>{post.content}</p>
        </div>
      ))}
    </div>
  );
}
```

## Contributing

1. Clone the repository
2. Install dependencies: `npm install`
3. Run tests: `npm test`
4. Build: `npm run build`

## License

MIT License - see LICENSE file for details.