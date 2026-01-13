"""Deployments Service"""

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .client import DevBoxClient, AsyncDevBoxClient


@dataclass
class Deployment:
    """Deployment"""
    id: str
    environment_id: str
    version: str
    status: str
    created_at: str
    image_tag: Optional[str] = None
    commit_sha: Optional[str] = None
    commit_message: Optional[str] = None
    deployed_by: Optional[str] = None
    started_at: Optional[str] = None
    completed_at: Optional[str] = None


@dataclass
class LogEntry:
    """Log entry"""
    timestamp: str
    level: str
    message: str
    stage: Optional[str] = None


@dataclass
class DeploymentLogs:
    """Deployment logs"""
    logs: list[LogEntry]


class DeploymentsService:
    """Deployments service for sync client"""

    def __init__(self, client: "DevBoxClient"):
        self._client = client

    def list(
        self,
        page: int = 1,
        page_size: int = 20,
        environment_id: Optional[str] = None,
        status: Optional[str] = None,
    ) -> dict:
        """List deployments"""
        params = {"page": page, "page_size": page_size}
        if environment_id:
            params["environment_id"] = environment_id
        if status:
            params["status"] = status
        return self._client.request("GET", "/deployments", params=params)

    def get(self, id: str) -> dict:
        """Get deployment by ID"""
        return self._client.request("GET", f"/deployments/{id}")

    def create(
        self,
        environment_id: str,
        branch: Optional[str] = None,
        commit_sha: Optional[str] = None,
        build_args: Optional[dict[str, str]] = None,
    ) -> dict:
        """Create a new deployment"""
        data = {"environment_id": environment_id}
        if branch:
            data["branch"] = branch
        if commit_sha:
            data["commit_sha"] = commit_sha
        if build_args:
            data["build_args"] = build_args
        return self._client.request("POST", "/deployments", json=data)

    def rollback(self, id: str) -> dict:
        """Rollback a deployment"""
        return self._client.request("POST", f"/deployments/{id}/rollback")

    def get_logs(self, id: str) -> dict:
        """Get deployment logs"""
        return self._client.request("GET", f"/deployments/{id}/logs")


class AsyncDeploymentsService:
    """Deployments service for async client"""

    def __init__(self, client: "AsyncDevBoxClient"):
        self._client = client

    async def list(
        self,
        page: int = 1,
        page_size: int = 20,
        environment_id: Optional[str] = None,
        status: Optional[str] = None,
    ) -> dict:
        """List deployments"""
        params = {"page": page, "page_size": page_size}
        if environment_id:
            params["environment_id"] = environment_id
        if status:
            params["status"] = status
        return await self._client.request("GET", "/deployments", params=params)

    async def get(self, id: str) -> dict:
        """Get deployment by ID"""
        return await self._client.request("GET", f"/deployments/{id}")

    async def create(
        self,
        environment_id: str,
        branch: Optional[str] = None,
        commit_sha: Optional[str] = None,
        build_args: Optional[dict[str, str]] = None,
    ) -> dict:
        """Create a new deployment"""
        data = {"environment_id": environment_id}
        if branch:
            data["branch"] = branch
        if commit_sha:
            data["commit_sha"] = commit_sha
        if build_args:
            data["build_args"] = build_args
        return await self._client.request("POST", "/deployments", json=data)

    async def rollback(self, id: str) -> dict:
        """Rollback a deployment"""
        return await self._client.request("POST", f"/deployments/{id}/rollback")

    async def get_logs(self, id: str) -> dict:
        """Get deployment logs"""
        return await self._client.request("GET", f"/deployments/{id}/logs")
