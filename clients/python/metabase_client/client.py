"""
Main MetaBase client for API interactions.
"""

import json
import time
from typing import Any, Dict, List, Optional, Union
import requests
from urllib.parse import urljoin, urlencode

from .types import (
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
)


class MetaBaseClient:
    """
    Main client for interacting with MetaBase API.
    """

    def __init__(self, config: MetaBaseConfig):
        """
        Initialize the MetaBase client.

        Args:
            config: Configuration object containing API settings
        """
        self.config = config
        self.session = requests.Session()

        # Set up session
        self.session.timeout = config.timeout / 1000  # Convert to seconds
        self.session.headers.update({
            'Authorization': f'Bearer {config.api_key}',
            'Content-Type': 'application/json',
            'User-Agent': 'metabase-client-python/1.0.0',
            **config.headers
        })

    def _handle_error(self, response: requests.Response) -> APIError:
        """
        Handle API error responses.

        Args:
            response: HTTP response object

        Returns:
            APIError object
        """
        try:
            error_data = response.json()
            error_info = error_data.get('error', {})

            return APIError(
                code=error_info.get('code', f'http_{response.status_code}'),
                message=error_info.get('message', response.reason),
                details=error_info.get('details'),
                timestamp=error_info.get('timestamp')
            )
        except (json.JSONDecodeError, KeyError):
            return APIError(
                code=f'http_{response.status_code}',
                message=response.reason,
                timestamp=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
            )

    def _request(
        self,
        method: str,
        endpoint: str,
        data: Optional[Dict[str, Any]] = None,
        params: Optional[Dict[str, Any]] = None,
        **kwargs
    ) -> APIResponse:
        """
        Make HTTP request to API.

        Args:
            method: HTTP method
            endpoint: API endpoint
            data: Request body data
            params: URL parameters
            **kwargs: Additional request arguments

        Returns:
            APIResponse object
        """
        url = urljoin(self.config.url + '/', endpoint.lstrip('/'))

        if self.config.debug:
            print(f"MetaBase Request: {method.upper()} {url}")
            if data:
                print(f"Data: {json.dumps(data, indent=2)}")
            if params:
                print(f"Params: {params}")

        try:
            response = self.session.request(
                method=method,
                url=url,
                json=data,
                params=params,
                **kwargs
            )

            if self.config.debug:
                print(f"Response Status: {response.status_code}")

            if response.status_code >= 400:
                return APIResponse(error=self._handle_error(response))

            response_data = response.json()

            # Convert response data to APIResponse format
            if isinstance(response_data, dict):
                return APIResponse(
                    data=response_data.get('data'),
                    count=response_data.get('count'),
                    limit=response_data.get('limit'),
                    offset=response_data.get('offset'),
                    has_next=response_data.get('has_next')
                )
            else:
                return APIResponse(data=response_data)

        except requests.exceptions.RequestException as e:
            return APIResponse(
                error=APIError(
                    code='network_error',
                    message=str(e),
                    timestamp=time.strftime('%Y-%m-%dT%H:%M:%SZ', time.gmtime())
                )
            )

    # Health Check Methods
    def health(self) -> APIResponse[HealthResponse]:
        """
        Check API health status.

        Returns:
            Health check response
        """
        return self._request('GET', '/rest/health')

    def ping(self) -> APIResponse[str]:
        """
        Simple ping to check connectivity.

        Returns:
            Pong response
        """
        return self._request('GET', '/ping')

    # Database Methods
    def query(
        self,
        table: str,
        options: Optional[QueryOptions] = None
    ) -> APIResponse[List[Dict[str, Any]]]:
        """
        Query data from a table.

        Args:
            table: Table name
            options: Query options

        Returns:
            Query response
        """
        params = {}
        if options:
            if options.select:
                params['select'] = ','.join(options.select)

            if options.where:
                for key, value in options.where.items():
                    if isinstance(value, (dict, list)):
                        params[key] = json.dumps(value)
                    else:
                        params[key] = str(value)

            if options.order:
                params['order'] = options.order

            if options.limit:
                params['limit'] = options.limit

            if options.offset:
                params['offset'] = options.offset

        return self._request('GET', f'/rest/v1/{table}', params=params)

    def get(
        self,
        table: str,
        id: Union[str, int],
        select: Optional[List[str]] = None
    ) -> APIResponse[Dict[str, Any]]:
        """
        Get a single record by ID.

        Args:
            table: Table name
            id: Record ID
            select: Fields to select

        Returns:
            Single record response
        """
        params = {}
        if select:
            params['select'] = ','.join(select)

        return self._request('GET', f'/rest/v1/{table}/{id}', params=params)

    def insert(
        self,
        table: str,
        data: Union[Dict[str, Any], List[Dict[str, Any]]],
        options: Optional[InsertOptions] = None
    ) -> APIResponse[Union[Dict[str, Any], List[Dict[str, Any]]]]:
        """
        Insert data into a table.

        Args:
            table: Table name
            data: Data to insert
            options: Insert options

        Returns:
            Insert response
        """
        params = {}
        if options and options.returning:
            params['returning'] = ','.join(options.returning)

        return self._request('POST', f'/rest/v1/{table}', data=data, params=params)

    def update(
        self,
        table: str,
        data: Dict[str, Any],
        options: Optional[UpdateOptions] = None,
        **where_conditions
    ) -> APIResponse[List[Dict[str, Any]]]:
        """
        Update data in a table.

        Args:
            table: Table name
            data: Data to update
            options: Update options
            **where_conditions: WHERE conditions

        Returns:
            Update response
        """
        params = {}

        # Add where conditions to params
        for key, value in where_conditions.items():
            if isinstance(value, (dict, list)):
                params[key] = json.dumps(value)
            else:
                params[key] = str(value)

        if options and options.returning:
            params['returning'] = ','.join(options.returning)

        return self._request('PATCH', f'/rest/v1/{table}', data=data, params=params)

    def update_one(
        self,
        table: str,
        id: Union[str, int],
        data: Dict[str, Any],
        options: Optional[UpdateOptions] = None
    ) -> APIResponse[Dict[str, Any]]:
        """
        Update a single record by ID.

        Args:
            table: Table name
            id: Record ID
            data: Data to update
            options: Update options

        Returns:
            Update response
        """
        params = {}
        if options and options.returning:
            params['returning'] = ','.join(options.returning)

        return self._request('PATCH', f'/rest/v1/{table}/{id}', data=data, params=params)

    def delete(
        self,
        table: str,
        options: Optional[UpdateOptions] = None,
        **where_conditions
    ) -> APIResponse[List[Dict[str, Any]]]:
        """
        Delete data from a table.

        Args:
            table: Table name
            options: Delete options
            **where_conditions: WHERE conditions

        Returns:
            Delete response
        """
        params = {}

        # Add where conditions to params
        for key, value in where_conditions.items():
            if isinstance(value, (dict, list)):
                params[key] = json.dumps(value)
            else:
                params[key] = str(value)

        if options and options.returning:
            params['returning'] = ','.join(options.returning)

        return self._request('DELETE', f'/rest/v1/{table}', params=params)

    def delete_one(
        self,
        table: str,
        id: Union[str, int],
        options: Optional[UpdateOptions] = None
    ) -> APIResponse[Dict[str, Any]]:
        """
        Delete a single record by ID.

        Args:
            table: Table name
            id: Record ID
            options: Delete options

        Returns:
            Delete response
        """
        params = {}
        if options and options.returning:
            params['returning'] = ','.join(options.returning)

        return self._request('DELETE', f'/rest/v1/{table}/{id}', params=params)

    # Table Management
    def list_tables(self) -> APIResponse[List[str]]:
        """
        List all tables.

        Returns:
            List of table names
        """
        return self._request('GET', '/rest/v1')

    def get_table_schema(self, table: str) -> APIResponse[Dict[str, Any]]:
        """
        Get table schema.

        Args:
            table: Table name

        Returns:
            Table schema
        """
        return self._request('GET', f'/rest/v1/{table}/schema')

    # Utility Methods
    def close(self):
        """Close the session."""
        self.session.close()

    def __enter__(self):
        """Context manager entry."""
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        """Context manager exit."""
        self.close()