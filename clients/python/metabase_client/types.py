"""
Type definitions for MetaBase client library.
"""

from typing import Dict, List, Optional, Any, Union
from datetime import datetime
from enum import Enum


# Core Types
class KeyType(str, Enum):
    """API key types."""
    SYSTEM = "system"
    USER = "user"
    SERVICE = "service"


class KeyStatus(str, Enum):
    """API key status."""
    ACTIVE = "active"
    INACTIVE = "inactive"
    REVOKED = "revoked"
    EXPIRED = "expired"


# Data Structures
class MetaBaseConfig:
    """Configuration for MetaBase client."""

    def __init__(
        self,
        url: str,
        api_key: str,
        timeout: int = 30000,
        headers: Optional[Dict[str, str]] = None,
        debug: bool = False
    ):
        self.url = url.rstrip('/')
        self.api_key = api_key
        self.timeout = timeout
        self.headers = headers or {}
        self.debug = debug


class APIResponse:
    """Generic API response."""

    def __init__(
        self,
        data: Optional[Any] = None,
        count: Optional[int] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        has_next: Optional[bool] = None,
        error: Optional["APIError"] = None
    ):
        self.data = data
        self.count = count
        self.limit = limit
        self.offset = offset
        self.has_next = has_next
        self.error = error


class APIError:
    """API error information."""

    def __init__(
        self,
        code: str,
        message: str,
        details: Optional[str] = None,
        timestamp: Optional[str] = None
    ):
        self.code = code
        self.message = message
        self.details = details
        self.timestamp = timestamp or datetime.utcnow().isoformat()


class QueryOptions:
    """Query options for database operations."""

    def __init__(
        self,
        select: Optional[List[str]] = None,
        where: Optional[Dict[str, Any]] = None,
        order: Optional[str] = None,
        limit: Optional[int] = None,
        offset: Optional[int] = None,
        joins: Optional[List["JoinClause"]] = None,
        group_by: Optional[List[str]] = None,
        having: Optional[Dict[str, Any]] = None
    ):
        self.select = select or []
        self.where = where or {}
        self.order = order
        self.limit = limit
        self.offset = offset
        self.joins = joins or []
        self.group_by = group_by or []
        self.having = having or {}


class JoinClause:
    """JOIN clause for complex queries."""

    def __init__(
        self,
        type: str,  # 'inner', 'left', 'right', 'outer'
        table: str,
        alias: Optional[str] = None,
        condition: str = ""
    ):
        self.type = type
        self.table = table
        self.alias = alias
        self.condition = condition


class InsertOptions:
    """Options for insert operations."""

    def __init__(self, returning: Optional[List[str]] = None):
        self.returning = returning or []


class UpdateOptions:
    """Options for update operations."""

    def __init__(self, returning: Optional[List[str]] = None):
        self.returning = returning or []


# Health Check Types
class DatabaseStatus:
    """Database status information."""

    def __init__(self, connected: bool, version: str):
        self.connected = connected
        self.version = version


class CacheStatus:
    """Cache status information."""

    def __init__(self, connected: bool, type: str):
        self.connected = connected
        self.type = type


class HealthResponse:
    """Health check response."""

    def __init__(
        self,
        status: str,
        version: str,
        uptime: str,
        database: DatabaseStatus,
        cache: CacheStatus,
        timestamp: str
    ):
        self.status = status
        self.version = version
        self.uptime = uptime
        self.database = database
        self.cache = cache
        self.timestamp = timestamp


# Authentication Types
class APIKey:
    """API key information."""

    def __init__(
        self,
        id: str,
        name: str,
        type: KeyType,
        status: KeyStatus,
        scopes: List[str],
        tenant_id: Optional[str] = None,
        project_id: Optional[str] = None,
        created_by: str = "",
        user_id: Optional[str] = None,
        expires_at: Optional[str] = None,
        created_at: Optional[str] = None,
        updated_at: Optional[str] = None,
        last_used_at: Optional[str] = None,
        usage_count: int = 0,
        metadata: Optional[Dict[str, Any]] = None
    ):
        self.id = id
        self.name = name
        self.type = type
        self.status = status
        self.scopes = scopes
        self.tenant_id = tenant_id
        self.project_id = project_id
        self.created_by = created_by
        self.user_id = user_id
        self.expires_at = expires_at
        self.created_at = created_at
        self.updated_at = updated_at
        self.last_used_at = last_used_at
        self.usage_count = usage_count
        self.metadata = metadata or {}


class CreateKeyRequest:
    """Request to create a new API key."""

    def __init__(
        self,
        name: str,
        type: KeyType,
        scopes: Optional[List[str]] = None,
        tenant_id: Optional[str] = None,
        project_id: Optional[str] = None,
        user_id: Optional[str] = None,
        expires_at: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ):
        self.name = name
        self.type = type
        self.scopes = scopes or []
        self.tenant_id = tenant_id
        self.project_id = project_id
        self.user_id = user_id
        self.expires_at = expires_at
        self.metadata = metadata or {}


class UpdateKeyRequest:
    """Request to update an API key."""

    def __init__(
        self,
        name: Optional[str] = None,
        status: Optional[KeyStatus] = None,
        scopes: Optional[List[str]] = None,
        expires_at: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None
    ):
        self.name = name
        self.status = status
        self.scopes = scopes
        self.expires_at = expires_at
        self.metadata = metadata


class KeyFilter:
    """Filter for listing API keys."""

    def __init__(
        self,
        tenant_id: Optional[str] = None,
        project_id: Optional[str] = None,
        type: Optional[KeyType] = None,
        status: Optional[KeyStatus] = None,
        user_id: Optional[str] = None,
        limit: int = 50,
        offset: int = 0
    ):
        self.tenant_id = tenant_id
        self.project_id = project_id
        self.type = type
        self.status = status
        self.user_id = user_id
        self.limit = limit
        self.offset = offset


class EndpointUsage:
    """Endpoint usage statistics."""

    def __init__(self, endpoint: str, count: int):
        self.endpoint = endpoint
        self.count = count


class KeyUsageStats:
    """API key usage statistics."""

    def __init__(
        self,
        key_id: str,
        usage_count: int,
        last_used_at: Optional[str] = None,
        top_endpoints: Optional[List[EndpointUsage]] = None
    ):
        self.key_id = key_id
        self.usage_count = usage_count
        self.last_used_at = last_used_at
        self.top_endpoints = top_endpoints or []


# Realtime Types
class RealtimeEvent:
    """Real-time event."""

    def __init__(
        self,
        type: str,  # 'INSERT', 'UPDATE', 'DELETE'
        table: str,
        record: Optional[Any] = None,
        old: Optional[Any] = None,
        new: Optional[Any] = None,
        timestamp: str = "",
        metadata: Optional[Dict[str, Any]] = None
    ):
        self.type = type
        self.table = table
        self.record = record
        self.old = old
        self.new = new
        self.timestamp = timestamp or datetime.utcnow().isoformat()
        self.metadata = metadata or {}


class RealtimeSubscription:
    """Real-time subscription."""

    def __init__(
        self,
        id: str,
        table: str,
        filter: Optional[Dict[str, Any]] = None,
        callback: Optional[callable] = None,
        ws: Optional[Any] = None,
        active: bool = False
    ):
        self.id = id
        self.table = table
        self.filter = filter or {}
        self.callback = callback
        self.ws = ws
        self.active = active


# File Types
class FileUploadOptions:
    """Options for file upload."""

    def __init__(
        self,
        filename: Optional[str] = None,
        mime_type: Optional[str] = None,
        metadata: Optional[Dict[str, Any]] = None,
        public: bool = False,
        expires_at: Optional[str] = None
    ):
        self.filename = filename
        self.mime_type = mime_type
        self.metadata = metadata or {}
        self.public = public
        self.expires_at = expires_at


class FileInfo:
    """File information."""

    def __init__(
        self,
        id: str,
        filename: str,
        size: int,
        mime_type: str,
        hash: str,
        public_url: Optional[str] = None,
        download_url: str = "",
        metadata: Optional[Dict[str, Any]] = None,
        created_at: Optional[str] = None,
        updated_at: Optional[str] = None,
        expires_at: Optional[str] = None,
        created_by: str = ""
    ):
        self.id = id
        self.filename = filename
        self.size = size
        self.mime_type = mime_type
        self.hash = hash
        self.public_url = public_url
        self.download_url = download_url
        self.metadata = metadata or {}
        self.created_at = created_at
        self.updated_at = updated_at
        self.expires_at = expires_at
        self.created_by = created_by


class FileListOptions:
    """Options for listing files."""

    def __init__(
        self,
        search: Optional[str] = None,
        mime_type: Optional[str] = None,
        public: Optional[bool] = None,
        created_by: Optional[str] = None,
        min_size: Optional[int] = None,
        max_size: Optional[int] = None,
        created_after: Optional[str] = None,
        created_before: Optional[str] = None,
        limit: int = 50,
        offset: int = 0,
        sort_by: str = "created_at",
        sort_order: str = "desc"
    ):
        self.search = search
        self.mime_type = mime_type
        self.public = public
        self.created_by = created_by
        self.min_size = min_size
        self.max_size = max_size
        self.created_after = created_after
        self.created_before = created_before
        self.limit = limit
        self.offset = offset
        self.sort_by = sort_by
        self.sort_order = sort_order