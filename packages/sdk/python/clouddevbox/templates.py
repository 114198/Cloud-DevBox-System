"""Templates Service"""

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

from .types import ResourceConfig

if TYPE_CHECKING:
    from .client import DevBoxClient, AsyncDevBoxClient


@dataclass
class Template:
    """Environment template"""
    id: str
    name: str
    is_official: bool
    is_public: bool
    created_at: str
    updated_at: str
    description: Optional[str] = None
    category: Optional[str] = None
    tags: Optional[list[str]] = None
    icon_url: Optional[str] = None
    default_resources: Optional[ResourceConfig] = None
    dockerfile: Optional[str] = None
    setup_commands: Optional[list[str]] = None
    version: Optional[str] = None


class TemplatesService:
    """Templates service for sync client"""

    def __init__(self, client: "DevBoxClient"):
        self._client = client

    def list(
        self,
        page: int = 1,
        page_size: int = 20,
        category: Optional[str] = None,
        search: Optional[str] = None,
    ) -> dict:
        """List templates"""
        params = {"page": page, "page_size": page_size}
        if category:
            params["category"] = category
        if search:
            params["search"] = search
        return self._client.request("GET", "/templates", params=params)

    def get(self, id: str) -> dict:
        """Get template by ID"""
        return self._client.request("GET", f"/templates/{id}")

    def create(self, name: str, dockerfile: str, **kwargs) -> dict:
        """Create a new template"""
        data = {"name": name, "dockerfile": dockerfile, **kwargs}
        return self._client.request("POST", "/templates", json=data)

    def update(self, id: str, **kwargs) -> dict:
        """Update a template"""
        return self._client.request("PUT", f"/templates/{id}", json=kwargs)

    def delete(self, id: str) -> None:
        """Delete a template"""
        self._client.request("DELETE", f"/templates/{id}")


class AsyncTemplatesService:
    """Templates service for async client"""

    def __init__(self, client: "AsyncDevBoxClient"):
        self._client = client

    async def list(
        self,
        page: int = 1,
        page_size: int = 20,
        category: Optional[str] = None,
        search: Optional[str] = None,
    ) -> dict:
        """List templates"""
        params = {"page": page, "page_size": page_size}
        if category:
            params["category"] = category
        if search:
            params["search"] = search
        return await self._client.request("GET", "/templates", params=params)

    async def get(self, id: str) -> dict:
        """Get template by ID"""
        return await self._client.request("GET", f"/templates/{id}")

    async def create(self, name: str, dockerfile: str, **kwargs) -> dict:
        """Create a new template"""
        data = {"name": name, "dockerfile": dockerfile, **kwargs}
        return await self._client.request("POST", "/templates", json=data)

    async def update(self, id: str, **kwargs) -> dict:
        """Update a template"""
        return await self._client.request("PUT", f"/templates/{id}", json=kwargs)

    async def delete(self, id: str) -> None:
        """Delete a template"""
        await self._client.request("DELETE", f"/templates/{id}")
