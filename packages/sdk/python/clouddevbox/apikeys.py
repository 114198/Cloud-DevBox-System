"""API Keys Service"""

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .client import DevBoxClient, AsyncDevBoxClient


@dataclass
class APIKey:
    """API key"""
    id: str
    name: str
    key_prefix: str
    scopes: list[str]
    is_active: bool
    created_at: str
    description: Optional[str] = None
    expires_at: Optional[str] = None
    last_used_at: Optional[str] = None
    last_used_ip: Optional[str] = None


class APIKeysService:
    """API Keys service for sync client"""

    def __init__(self, client: "DevBoxClient"):
        self._client = client

    def list(self, page: int = 1, page_size: int = 20) -> dict:
        """List API keys"""
        return self._client.request("GET", "/api-keys", params={"page": page, "page_size": page_size})

    def get(self, id: str) -> dict:
        """Get API key by ID"""
        return self._client.request("GET", f"/api-keys/{id}")

    def create(self, name: str, scopes: list[str], **kwargs) -> dict:
        """Create a new API key"""
        data = {"name": name, "scopes": scopes, **kwargs}
        return self._client.request("POST", "/api-keys", json=data)

    def update(self, id: str, **kwargs) -> dict:
        """Update an API key"""
        return self._client.request("PUT", f"/api-keys/{id}", json=kwargs)

    def delete(self, id: str) -> None:
        """Delete an API key"""
        self._client.request("DELETE", f"/api-keys/{id}")

    def revoke(self, id: str) -> None:
        """Revoke an API key"""
        self._client.request("POST", f"/api-keys/{id}/revoke")

    def get_available_scopes(self) -> list[str]:
        """Get available API key scopes"""
        return self._client.request("GET", "/api-keys/scopes")


class AsyncAPIKeysService:
    """API Keys service for async client"""

    def __init__(self, client: "AsyncDevBoxClient"):
        self._client = client

    async def list(self, page: int = 1, page_size: int = 20) -> dict:
        """List API keys"""
        return await self._client.request("GET", "/api-keys", params={"page": page, "page_size": page_size})

    async def get(self, id: str) -> dict:
        """Get API key by ID"""
        return await self._client.request("GET", f"/api-keys/{id}")

    async def create(self, name: str, scopes: list[str], **kwargs) -> dict:
        """Create a new API key"""
        data = {"name": name, "scopes": scopes, **kwargs}
        return await self._client.request("POST", "/api-keys", json=data)

    async def update(self, id: str, **kwargs) -> dict:
        """Update an API key"""
        return await self._client.request("PUT", f"/api-keys/{id}", json=kwargs)

    async def delete(self, id: str) -> None:
        """Delete an API key"""
        await self._client.request("DELETE", f"/api-keys/{id}")

    async def revoke(self, id: str) -> None:
        """Revoke an API key"""
        await self._client.request("POST", f"/api-keys/{id}/revoke")

    async def get_available_scopes(self) -> list[str]:
        """Get available API key scopes"""
        return await self._client.request("GET", "/api-keys/scopes")
