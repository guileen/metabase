"""
MetaBase Python Client Library

Official Python client for interacting with the MetaBase API.
Provides a simple and intuitive interface for database operations,
real-time subscriptions, file management, and authentication.
"""

from .client import MetaBaseClient
from .auth import AuthManager
from .files import FileManager
from .realtime import RealtimeManager
from .types import (
    # Core types
    MetaBaseConfig,
    APIResponse,
    APIError,
    QueryOptions,
    JoinClause,
    InsertOptions,
    UpdateOptions,
    HealthResponse,
    DatabaseStatus,
    CacheStatus,

    # Authentication types
    APIKey,
    CreateKeyRequest,
    UpdateKeyRequest,
    KeyFilter,
    KeyUsageStats,
    EndpointUsage,
    KeyType,
    KeyStatus,

    # Realtime types
    RealtimeSubscription,
    RealtimeEvent,

    # File types
    FileUploadOptions,
    FileInfo,
    FileListOptions,
)

__version__ = "1.0.0"
__author__ = "MetaBase Team"
__email__ = "support@metabase.dev"

__all__ = [
    # Main client
    "MetaBaseClient",

    # Managers
    "AuthManager",
    "RealtimeManager",
    "FileManager",

    # Types
    "MetaBaseConfig",
    "APIResponse",
    "APIError",
    "QueryOptions",
    "JoinClause",
    "InsertOptions",
    "UpdateOptions",
    "HealthResponse",
    "DatabaseStatus",
    "CacheStatus",
    "APIKey",
    "CreateKeyRequest",
    "UpdateKeyRequest",
    "KeyFilter",
    "KeyUsageStats",
    "EndpointUsage",
    "KeyType",
    "KeyStatus",
    "RealtimeSubscription",
    "RealtimeEvent",
    "FileUploadOptions",
    "FileInfo",
    "FileListOptions",
]

def create_client(config: MetaBaseConfig) -> dict:
    """
    Factory function to create a complete client with all managers.

    Args:
        config: Configuration for the MetaBase client

    Returns:
        Dictionary containing client and managers:
        {
            'client': MetaBaseClient,
            'auth': AuthManager,
            'realtime': RealtimeManager,
            'files': FileManager
        }
    """
    client = MetaBaseClient(config)

    return {
        'client': client,
        'auth': AuthManager(client),
        'realtime': RealtimeManager(client),
        'files': FileManager(client),
    }