// Package openapi contains the documented discord-bot HTTP contract.
package openapi

// Spec is the OpenAPI 3.0 document served by the HTTP API.
const Spec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "discord-bot API",
    "description": "Operational API for the Discord bot boilerplate.",
    "version": "1.0.0"
  },
  "servers": [{"url": "/"}],
  "paths": {
    "/status": {
      "get": {
        "summary": "Read service and dependency status",
        "tags": ["Public"],
        "responses": {
          "200": {
            "description": "Service status.",
            "content": {
              "application/json": {
                "schema": {"$ref": "#/components/schemas/StatusResponse"}
              }
            }
          }
        }
      }
    }
  },
  "components": {
    "schemas": {
      "StatusResponse": {
        "type": "object",
        "required": ["status", "environment", "version", "dependencies"],
        "properties": {
          "status": {"type": "string", "example": "ok"},
          "environment": {"type": "string", "enum": ["development", "test", "production"]},
          "version": {"type": "string", "example": "1.0.0"},
          "dependencies": {
            "type": "object",
            "additionalProperties": {"type": "string", "enum": ["available", "unavailable"]}
          }
        }
      }
    }
  }
}`
