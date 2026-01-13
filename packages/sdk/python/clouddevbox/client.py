"""Cloud DevBox API Client"""

from typing import Any, Optional, TypeVar
import httpx

from .types import APIError, DevBoxError

T = TypeVar("T")

DEFAULT_BASE_URL = "https://api.clouddevbox.io/api/v1"
DEFAULT_TIMEOUT = 30.0


class DevBoxClient:
    """Cloud DevBox API client"""

    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT,
    ):
        self.api_key = api_key
        self.base_url = base_url
        self.timeout = timeout
        self._client = httpx.Client(
            base_url=base_url,
            timeout=timeout,
            headers={
                "X-API-Key": api_key,
                "Content-Type": "application/json",
                "Accept": "application/json",
                "User-Agent": "CloudDevBox-Python-SDK/1.0",
            },
        )

        # Initialize services
        from .environments import EnvironmentsService
        from .templates import TemplatesService
        from .deployments import DeploymentsService
        from .webhooks import WebhooksService
        from .apikeys import APIKeysService

        self.environments = EnvironmentsService(self)
        self.templates = TemplatesService(self)
        self.deployments = DeploymentsService(self)
        self.webhooks = WebhooksService(self)
        self.api_keys = APIKeysService(self)

    def request(
        self,
        method: str,
        path: str,
        json: Optional[dict[str, Any]] = None,
        params: Optional[dict[str, Any]] = None,
    ) -> Any:
        """Make an HTTP request to the API"""
        response = self._client.request(
            method=method,
            url=path,
            json=json,
            params=params,
        )

        if response.status_code >= 400:
            error_data = response.json()
            raise DevBoxError(APIError(**error_data))

        if response.status_code == 204:
            return None

        return response.json()

    def close(self):
        """Close the HTTP client"""
        self._client.close()

    def __enter__(self):
        return self

    def __exit__(self, exc_type, exc_val, exc_tb):
        self.close()


class AsyncDevBoxClient:
    """Async Cloud DevBox API client"""

    def __init__(
        self,
        api_key: str,
        base_url: str = DEFAULT_BASE_URL,
        timeout: float = DEFAULT_TIMEOUT,
    ):
        self.api_key = api_key
        self.base_url = base_url
        self.timeout = timeout
        self._client = httpx.AsyncClient(
            base_url=base_url,
            timeout=timeout,
            headers={
                "X-API-Key": api_key,
                "Content-Type": "application/json",
                "Accept": "application/json",
                "User-Agent": "CloudDevBox-Python-SDK/1.0",
            },
        )

        # Initialize async services
        from .environments import AsyncEnvironmentsService
        from .templates import AsyncTemplatesService
        from .deployments import AsyncDeploymentsService
        from .webhooks import AsyncWebhooksService
        from .apikeys import AsyncAPIKeysService

        self.environments = AsyncEnvironmentsService(self)
        self.templates = AsyncTemplatesService(self)
        self.deployments = AsyncDeploymentsService(self)
        self.webhooks = AsyncWebhooksService(self)
        self.api_keys = AsyncAPIKeysService(self)

    async def request(
        self,
        method: str,
        path: str,
        json: Optional[dict[str, Any]] = None,
        params: Optional[dict[str, Any]] = None,
    ) -> Any:
        """Make an async HTTP request to the API"""
        response = await self._client.request(
            method=method,
            url=path,
            json=json,
            params=params,
        )

        if response.status_code >= 400:
            error_data = response.json()
            raise DevBoxError(APIError(**error_data))

        if response.status_code == 204:
            return None

        return response.json()

    async def close(self):
        """Close the HTTP client"""
        await self._client.aclose()

    async def __aenter__(self):
        return self

    async def __aexit__(self, exc_type, exc_val, exc_tb):
        await self.close()
