# UserProvider Interface

## Overview

The UserProvider interface is required for SSE routing in CRUDP. It extracts user context from requests to enable user-based routing and role-based broadcasting.

## Interface Definition

**File: `interfaces.go`**
```go
// UserProvider extracts user context from requests for SSE routing
type UserProvider interface {
    // GetUserID returns unique user identifier from context
    GetUserID(ctx context.Context) string
    
    // GetUserRole returns user role for role-based broadcasting
    // Examples: "admin", "user", "guest"
    GetUserRole(ctx context.Context) string
}
```

## Example Implementation

```go
type AuthMiddleware struct{}

func (a *AuthMiddleware) GetUserID(ctx context.Context) string {
    // Extract from JWT, session, etc.
    if userID, ok := ctx.Value("userID").(string); ok {
        return userID
    }
    return ""
}

func (a *AuthMiddleware) GetUserRole(ctx context.Context) string {
    if role, ok := ctx.Value("role").(string); ok {
        return role
    }
    return "guest"
}
```