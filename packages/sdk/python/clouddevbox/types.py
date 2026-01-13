"""Common types for the Cloud DevBox SDK"""

from dataclasses import dataclass
from typing import Optional


@dataclass
class APIError:
    """API error response"""
    code: str
    message: str
    details: Optional[dict[str, str]] = None
    request_id: Optional[str] = None
    timestamp: Optional[str] = None


class DevBoxError(Exception):
    """Cloud DevBox API error"""
    def __init__(self, error: APIError):
        super().__init__(error.message)
        self.code = error.code
        self.details = error.details
        self.request_id = error.request_id


@dataclass
class Pagination:
    """Pagination metadata"""
    page: int
    page_size: int
    total_items: int
    total_pages: int
    has_next: bool
    has_prev: bool


@dataclass
class ResourceConfig:
    """Resource configuration"""
    cpu: str
    memory: str
    storage: str


@dataclass
class PortMapping:
    """Port mapping configuration"""
    container_port: int
    protocol: str = "tcp"
    public: bool = False
