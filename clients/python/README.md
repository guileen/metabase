# MetaBase Python Client Library

Official Python client library for the MetaBase API, providing a complete interface for database operations, real-time subscriptions, file management, and authentication.

## Installation

```bash
pip install metabase-client
```

## Quick Start

```python
from metabase_client import create_client, MetaBaseConfig

# Create client with configuration
config = MetaBaseConfig(
    url="https://your-metabase-instance.com",
    api_key="your-api-key-here",
    debug=True
)

metabase = create_client(config)

# Use the client
try:
    # Health check
    health = metabase["client"].health()
    print(f"Health Status: {health.data.status}")

    # Query data
    users = metabase["client"].query(
        "users",
        select=["id", "email", "created_at"],
        where={"status": "active"},
        order="created_at DESC",
        limit=10
    )
    print(f"Found {len(users.data)} users")

    # Insert new record
    new_user = metabase["client"].insert("users", {
        "email": "user@example.com",
        "name": "John Doe",
        "status": "active"
    })
    print(f"Created user: {new_user.data}")

finally:
    # Close the client when done
    metabase["client"].close()
```

## Core Features

### Database Operations

```python
from metabase_client import MetaBaseClient, MetaBaseConfig, QueryOptions

config = MetaBaseConfig(
    url="https://your-metabase-instance.com",
    api_key="your-api-key"
)

client = MetaBaseClient(config)

try:
    # Query with filtering
    posts = client.query(
        "posts",
        QueryOptions(
            select=["title", "content", "author_id"],
            where={
                "published": True,
                "category": "tech"
            },
            order="published_at DESC",
            limit=20
        )
    )

    # Get single record
    post = client.get("posts", 123, ["title", "content"])

    # Insert new record
    new_post = client.insert("posts", {
        "title": "My New Post",
        "content": "This is the content...",
        "author_id": 456,
        "published": False
    }, returning=["id", "title", "created_at"])

    # Update record
    updated = client.update_one("posts", 123, {
        "title": "Updated Title",
        "content": "Updated content"
    })

    # Delete record
    deleted = client.delete_one("posts", 123)

finally:
    client.close()
```

### Authentication & API Key Management

```python
from metabase_client import AuthManager

auth = metabase["auth"]

# Create new API key
new_key = auth.create_key({
    "name": "My Service Key",
    "type": "service",
    "scopes": ["read", "write", "table:read", "table:update"],
    "expires_at": "2024-12-31T23:59:59Z"
})

# List API keys
keys = auth.list_keys({
    "type": "service",
    "status": "active",
    "limit": 10
})

# Get key usage stats
stats = auth.get_key_stats("key-id-here")

# Revoke key
auth.revoke_key("key-id-here")

# Utility functions
from metabase_client import AuthManager

is_valid = AuthManager.validate_key_format("metabase_svc_abc123_...")
key_type = AuthManager.extract_key_type("metabase_sys_def456_...")
```

### File Management

```python
from metabase_client import FileManager

files = metabase["files"]

# Upload file
with open("document.pdf", "rb") as f:
    uploaded = files.upload(
        f,
        FileUploadOptions(
            filename="document.pdf",
            metadata={
                "category": "documents",
                "project": "website-redesign"
            },
            public=True
        )
    )

# List files
file_list = files.list_files({
    "search": "document",
    "mime_type": "application/pdf",
    "public": True,
    "limit": 20,
    "sort_by": "created_at",
    "sort_order": "desc"
})

# Get file info
file_info = files.get_file_info("file-id-here")

# Download file
downloaded_data = files.download("file-id-here")

# Get download URL
download_url = files.get_download_url("file-id-here")

# Delete file
files.delete_file("file-id-here")
```

## Configuration

```python
from metabase_client import MetaBaseConfig

config = MetaBaseConfig(
    url="https://your-metabase-instance.com",
    api_key="your-api-key-here",
    timeout=30000,  # Request timeout in milliseconds
    headers={
        "X-Custom-Header": "value"
    },
    debug=True  # Enable debug logging
)
```

## Error Handling

The SDK provides comprehensive error handling:

```python
result = client.query("posts")

if result.error:
    print(f"Error: {result.error.code}")
    print(f"Message: {result.error.message}")
    if result.error.details:
        print(f"Details: {result.error.details}")
else:
    print(f"Success: {len(result.data)} posts found")
```

## Context Manager Support

For automatic resource cleanup:

```python
with MetaBaseClient(config) as client:
    result = client.query("posts")
    # Client automatically closed when exiting context
```

## Advanced Querying

```python
# Complex query with JOINs
from metabase_client import QueryOptions, JoinClause

options = QueryOptions(
    select=["posts.title", "posts.content", "users.name"],
    joins=[
        JoinClause(
            type="inner",
            table="users",
            alias="u",
            condition="posts.author_id = u.id"
        )
    ],
    where={"posts.published": True},
    order="posts.created_at DESC",
    limit=10
)

result = client.query("posts", options)
```

## Batch Operations

```python
# Batch insert
data = [
    {"title": "Post 1", "content": "Content 1"},
    {"title": "Post 2", "content": "Content 2"},
    {"title": "Post 3", "content": "Content 3"}
]

result = client.insert("posts", data)

# Batch update
client.update(
    "posts",
    {"status": "published"},
    status="draft",
    where={"created_at": {"op": ">=", "value": "2024-01-01"}}
)
```

## Development

```bash
# Clone the repository
git clone https://github.com/metabase/metabase.git
cd metabase/clients/python

# Install development dependencies
pip install -e ".[dev]"

# Run tests
pytest

# Run with coverage
pytest --cov=metabase_client

# Format code
black metabase_client/
isort metabase_client/

# Type checking
mypy metabase_client/

# Linting
flake8 metabase_client/
```

## License

MIT License - see LICENSE file for details.