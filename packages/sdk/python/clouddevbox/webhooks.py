"""Webhooks Service"""

from dataclasses import dataclass
from typing import Optional, TYPE_CHECKING

if TYPE_CHECKING:
    from .client import DevBoxClient, AsyncDevBoxClient


@dataclass
class WebhookRetryConfig:
    """Webhook retry configuration"""
    max_retries: int
    retry_delay_ms: int
    backoff_factor: float
    max_delay_ms: int


@dataclass
class Webhook:
    """Webhook subscription"""
    id: str
    name: str
    url: str
    events: list[str]
    is_active: bool
    created_at: str
    updated_at: str
    description: Optional[str] = None
    headers: Optional[dict[str, str]] = None
    retry_config: Optional[WebhookRetryConfig] = None


@dataclass
class WebhookDelivery:
    """Webhook delivery attempt"""
    id: str
    webhook_id: str
    event_type: str
    event_id: str
    payload: str
    status: str
    attempt_count: int
    created_at: str
    response_code: Optional[int] = None
    response_body: Optional[str] = None
    next_retry_at: Optional[str] = None
    delivered_at: Optional[str] = None
    error_message: Optional[str] = None
    duration_ms: Optional[int] = None


class WebhooksService:
    """Webhooks service for sync client"""

    def __init__(self, client: "DevBoxClient"):
        self._client = client

    def list(self, page: int = 1, page_size: int = 20) -> dict:
        """List webhooks"""
        return self._client.request("GET", "/webhooks", params={"page": page, "page_size": page_size})

    def get(self, id: str) -> dict:
        """Get webhook by ID"""
        return self._client.request("GET", f"/webhooks/{id}")

    def create(self, name: str, url: str, events: list[str], **kwargs) -> dict:
        """Create a new webhook"""
        data = {"name": name, "url": url, "events": events, **kwargs}
        return self._client.request("POST", "/webhooks", json=data)

    def update(self, id: str, **kwargs) -> dict:
        """Update a webhook"""
        return self._client.request("PUT", f"/webhooks/{id}", json=kwargs)

    def delete(self, id: str) -> None:
        """Delete a webhook"""
        self._client.request("DELETE", f"/webhooks/{id}")

    def set_active(self, id: str, active: bool) -> None:
        """Set webhook active status"""
        self._client.request("PUT", f"/webhooks/{id}/active", json={"active": active})

    def regenerate_secret(self, id: str) -> str:
        """Regenerate webhook secret"""
        resp = self._client.request("POST", f"/webhooks/{id}/secret")
        return resp["secret"]

    def test(self, id: str) -> None:
        """Send test event to webhook"""
        self._client.request("POST", f"/webhooks/{id}/test")

    def list_deliveries(self, webhook_id: str, page: int = 1, page_size: int = 20) -> dict:
        """List webhook deliveries"""
        return self._client.request(
            "GET", f"/webhooks/{webhook_id}/deliveries",
            params={"page": page, "page_size": page_size}
        )

    def retry_delivery(self, webhook_id: str, delivery_id: str) -> None:
        """Retry a failed delivery"""
        self._client.request("POST", f"/webhooks/{webhook_id}/deliveries/{delivery_id}/retry")

    def get_available_events(self) -> list[str]:
        """Get available webhook events"""
        return self._client.request("GET", "/webhooks/events")


class AsyncWebhooksService:
    """Webhooks service for async client"""

    def __init__(self, client: "AsyncDevBoxClient"):
        self._client = client

    async def list(self, page: int = 1, page_size: int = 20) -> dict:
        """List webhooks"""
        return await self._client.request("GET", "/webhooks", params={"page": page, "page_size": page_size})

    async def get(self, id: str) -> dict:
        """Get webhook by ID"""
        return await self._client.request("GET", f"/webhooks/{id}")

    async def create(self, name: str, url: str, events: list[str], **kwargs) -> dict:
        """Create a new webhook"""
        data = {"name": name, "url": url, "events": events, **kwargs}
        return await self._client.request("POST", "/webhooks", json=data)

    async def update(self, id: str, **kwargs) -> dict:
        """Update a webhook"""
        return await self._client.request("PUT", f"/webhooks/{id}", json=kwargs)

    async def delete(self, id: str) -> None:
        """Delete a webhook"""
        await self._client.request("DELETE", f"/webhooks/{id}")

    async def set_active(self, id: str, active: bool) -> None:
        """Set webhook active status"""
        await self._client.request("PUT", f"/webhooks/{id}/active", json={"active": active})

    async def regenerate_secret(self, id: str) -> str:
        """Regenerate webhook secret"""
        resp = await self._client.request("POST", f"/webhooks/{id}/secret")
        return resp["secret"]

    async def test(self, id: str) -> None:
        """Send test event to webhook"""
        await self._client.request("POST", f"/webhooks/{id}/test")

    async def list_deliveries(self, webhook_id: str, page: int = 1, page_size: int = 20) -> dict:
        """List webhook deliveries"""
        return await self._client.request(
            "GET", f"/webhooks/{webhook_id}/deliveries",
            params={"page": page, "page_size": page_size}
        )

    async def retry_delivery(self, webhook_id: str, delivery_id: str) -> None:
        """Retry a failed delivery"""
        await self._client.request("POST", f"/webhooks/{webhook_id}/deliveries/{delivery_id}/retry")

    async def get_available_events(self) -> list[str]:
        """Get available webhook events"""
        return await self._client.request("GET", "/webhooks/events")
