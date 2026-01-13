# Redis Cache Strategy

## Overview

Cloud DevBox uses Redis for caching, session management, and real-time data.

## Key Naming Convention

All keys follow the pattern: `devbox:{namespace}:{identifier}`

## Cache Keys

### Session Management
```
devbox:session:{user_id}                    # User session data (TTL: 2h)
devbox:session:{user_id}:refresh            # Refresh token (TTL: 7d)
```

### Environment Status
```
devbox:env:status:{environment_id}          # Environment status (TTL: 60s)
devbox:env:metrics:{environment_id}         # Resource metrics (TTL: 30s)
devbox:env:connections:{environment_id}     # Active connections (TTL: 5m)
```

### Templates
```
devbox:templates:all                        # All templates list (TTL: 1h)
devbox:templates:category:{category}        # Templates by category (TTL: 1h)
devbox:template:{template_id}               # Single template (TTL: 1h)
```

### User Data
```
devbox:user:{user_id}                       # User profile (TTL: 30m)
devbox:user:{user_id}:quotas                # User quotas (TTL: 5m)
devbox:user:{user_id}:environments          # User's environments list (TTL: 5m)
```

### Rate Limiting
```
devbox:rate:{user_id}:{endpoint}            # API rate limit counter (TTL: 1h)
devbox:rate:env_create:{user_id}            # Environment creation limit (TTL: 1h)
```

### Collaboration
```
devbox:collab:{session_id}                  # Collaboration session (TTL: 24h)
devbox:collab:{session_id}:participants     # Active participants (TTL: 5m)
devbox:collab:{session_id}:cursors          # Cursor positions (TTL: 30s)
```

### Meetings
```
devbox:meeting:{meeting_id}                 # Meeting data (TTL: 24h)
devbox:meeting:{meeting_id}:participants    # Meeting participants (TTL: 5m)
```

## Cache Strategies

### Write-Through
Used for critical data that must be consistent:
- User sessions
- Environment status

### Lazy Loading (Cache-Aside)
Used for read-heavy data:
- Templates
- User profiles

### Proactive Refresh
Used for frequently accessed data:
- Environment metrics
- Active connections

## TTL Guidelines

| Data Type | TTL | Reason |
|-----------|-----|--------|
| Session | 2h | Security |
| Refresh Token | 7d | User convenience |
| Environment Status | 60s | Near real-time |
| Metrics | 30s | Real-time monitoring |
| Templates | 1h | Rarely changes |
| User Profile | 30m | Balance freshness/performance |
| Rate Limits | 1h | Per-hour limits |

## Eviction Policy

Redis is configured with `allkeys-lru` eviction policy to automatically remove least recently used keys when memory is full.

## Cluster Configuration

For production, Redis Cluster is recommended with:
- 3 master nodes
- 3 replica nodes
- Automatic failover
