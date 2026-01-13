"""Environments Service"""

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING
from datetime import datetime

from .types import ResourceConfig, PortMapping

if TYPE_CHECKING:
    from .client import DevBoxClient, AsyncDevBoxClient


@dataclass
class Environment:
    """Development environment"""
    id: str
    name: str
    template_id: str
    status: str
    created_at: str
    updated_at: str
    description: Optional[str] = None
    resources: Optional[ResourceConfig] = None
    env_vars: Optional[dict[str, str]] = None
    ports: Optional[list[PortMapping]] = None
    preview_url: Optional[str] = None
    ssh_host: Optional[str] = None
    ssh_port: Optional[int] = None
    last_active_at: Optional[str] = None


@dataclass
class SSHConfig:
    """SSH connection configuration"""
    host: str
    port: int
    username: str
    private_key: str
    config_snippet: str


@dataclass
class MetricPoint:
    """Single metric data point"""
    timestamp: str
    value: float


@dataclass
class EnvironmentMetrics:
    """Environment metrics"""
    cpu_usage: list[MetricPoint]
    memory_usage: list[MetricPoint]
    storage_usage: list[MetricPoint]
    network_rx: list[MetricPoint]
    network_tx: list[MetricPoint]


class EnvironmentsService:
    """Environments service for sync client"""

    def __init__(self, client: "DevBoxClient"):
        self._client = client

    def list(
        self,
        page: int = 1,
        page_size: int = 20,
        status: Optional[str] = None,
        template_id: Optional[str] = None,
    ) -> dict:
        """List environments"""
        params = {"page": page, "page_size": page_size}
        if status:
            params["status"] = status
        if template_id:
            params["template_id"] = template_id
        return self._client.request("GET", "/environments", params=params)

    def get(self, id: str) -> dict:
        """Get environment by ID"""
        return self._client.request("GET", f"/environments/{id}")

    def create(
        self,
        name: str,
        template_id: str,
        description: Optional[str] = None,
        resources: Optional[dict] = None,
        env_vars: Optional[dict[str, str]] = None,
        ports: Optional[list[dict]] = None,
        git_repo_url: Optional[str] = None,
    ) -> dict:
        """Create a new environment"""
        data = {"name": name, "template_id": template_id}
        if description:
            data["description"] = description
        if resources:
            data["resources"] = resources
        if env_vars:
            data["env_vars"] = env_vars
        if ports:
            data["ports"] = ports
        if git_repo_url:
            data["git_repo_url"] = git_repo_url
        return self._client.request("POST", "/environments", json=data)

    def update(self, id: str, **kwargs) -> dict:
        """Update an environment"""
        return self._client.request("PUT", f"/environments/{id}", json=kwargs)

    def delete(self, id: str) -> None:
        """Delete an environment"""
        self._client.request("DELETE", f"/environments/{id}")

    def start(self, id: str) -> dict:
        """Start an environment"""
        return self._client.request("POST", f"/environments/{id}/start")

    def stop(self, id: str) -> dict:
        """Stop an environment"""
        return self._client.request("POST", f"/environments/{id}/stop")

    def restart(self, id: str) -> dict:
        """Restart an environment"""
        return self._client.request("POST", f"/environments/{id}/restart")

    def get_ssh_config(self, id: str) -> dict:
        """Get SSH configuration"""
        return self._client.request("GET", f"/environments/{id}/ssh-config")

    def get_metrics(
        self,
        id: str,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
    ) -> dict:
        """Get environment metrics"""
        params = {}
        if start_time:
            params["start_time"] = start_time.isoformat()
        if end_time:
            params["end_time"] = end_time.isoformat()
        return self._client.request("GET", f"/environments/{id}/metrics", params=params)


class AsyncEnvironmentsService:
    """Environments service for async client"""

    def __init__(self, client: "AsyncDevBoxClient"):
        self._client = client

    async def list(
        self,
        page: int = 1,
        page_size: int = 20,
        status: Optional[str] = None,
        template_id: Optional[str] = None,
    ) -> dict:
        """List environments"""
        params = {"page": page, "page_size": page_size}
        if status:
            params["status"] = status
        if template_id:
            params["template_id"] = template_id
        return await self._client.request("GET", "/environments", params=params)

    async def get(self, id: str) -> dict:
        """Get environment by ID"""
        return await self._client.request("GET", f"/environments/{id}")

    async def create(
        self,
        name: str,
        template_id: str,
        description: Optional[str] = None,
        resources: Optional[dict] = None,
        env_vars: Optional[dict[str, str]] = None,
        ports: Optional[list[dict]] = None,
        git_repo_url: Optional[str] = None,
    ) -> dict:
        """Create a new environment"""
        data = {"name": name, "template_id": template_id}
        if description:
            data["description"] = description
        if resources:
            data["resources"] = resources
        if env_vars:
            data["env_vars"] = env_vars
        if ports:
            data["ports"] = ports
        if git_repo_url:
            data["git_repo_url"] = git_repo_url
        return await self._client.request("POST", "/environments", json=data)

    async def update(self, id: str, **kwargs) -> dict:
        """Update an environment"""
        return await self._client.request("PUT", f"/environments/{id}", json=kwargs)

    async def delete(self, id: str) -> None:
        """Delete an environment"""
        await self._client.request("DELETE", f"/environments/{id}")

    async def start(self, id: str) -> dict:
        """Start an environment"""
        return await self._client.request("POST", f"/environments/{id}/start")

    async def stop(self, id: str) -> dict:
        """Stop an environment"""
        return await self._client.request("POST", f"/environments/{id}/stop")

    async def restart(self, id: str) -> dict:
        """Restart an environment"""
        return await self._client.request("POST", f"/environments/{id}/restart")

    async def get_ssh_config(self, id: str) -> dict:
        """Get SSH configuration"""
        return await self._client.request("GET", f"/environments/{id}/ssh-config")

    async def get_metrics(
        self,
        id: str,
        start_time: Optional[datetime] = None,
        end_time: Optional[datetime] = None,
    ) -> dict:
        """Get environment metrics"""
        params = {}
        if start_time:
            params["start_time"] = start_time.isoformat()
        if end_time:
            params["end_time"] = end_time.isoformat()
        return await self._client.request("GET", f"/environments/{id}/metrics", params=params)
