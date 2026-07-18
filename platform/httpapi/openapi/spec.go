// Package openapi contains the documented discord-bot HTTP contract.
package openapi

// Spec is the OpenAPI 3.0 document served by the HTTP API.
const Spec = `{
  "openapi": "3.0.3",
  "info": {
    "title": "discord-bot API",
    "description": "Operational and managed static message API.",
    "version": "1.0.0"
  },
  "servers": [{"url": "/"}],
  "paths": {
    "/status": {
      "get": {
        "summary": "Read service and dependency status",
        "tags": ["Public"],
        "responses": {"200": {"description": "Service status", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/StatusResponse"}}}}}
      }
    },
    "/api/messages": {
      "post": {
        "summary": "Create a managed static message",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [{"$ref": "#/components/parameters/IdempotencyKey"}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Definition"}}}},
        "responses": {"201": {"description": "Created", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Record"}}}}, "400": {"$ref": "#/components/responses/BadRequest"}, "401": {"$ref": "#/components/responses/Unauthorized"}, "409": {"$ref": "#/components/responses/Conflict"}}
      },
      "get": {
        "summary": "List managed static messages",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [
          {"name": "state", "in": "query", "schema": {"type": "string"}},
          {"name": "guildId", "in": "query", "schema": {"type": "string"}},
          {"name": "channelId", "in": "query", "schema": {"type": "string"}},
          {"name": "limit", "in": "query", "schema": {"type": "integer", "minimum": 1, "maximum": 100, "default": 50}},
          {"name": "offset", "in": "query", "schema": {"type": "integer", "minimum": 0, "default": 0}}
        ],
        "responses": {"200": {"description": "Managed message page", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Page"}}}}, "400": {"$ref": "#/components/responses/BadRequest"}, "401": {"$ref": "#/components/responses/Unauthorized"}}
      }
    },
    "/api/messages/{key}": {
      "parameters": [{"$ref": "#/components/parameters/MessageKey"}],
      "get": {
        "summary": "Read a managed static message",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "responses": {"200": {"description": "Managed message", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Record"}}}}, "401": {"$ref": "#/components/responses/Unauthorized"}, "404": {"$ref": "#/components/responses/NotFound"}}
      },
      "put": {
        "summary": "Replace desired message state",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [{"$ref": "#/components/parameters/IdempotencyKey"}, {"$ref": "#/components/parameters/IfMatch"}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Definition"}}}},
        "responses": {"200": {"description": "Updated", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Record"}}}}, "400": {"$ref": "#/components/responses/BadRequest"}, "401": {"$ref": "#/components/responses/Unauthorized"}, "404": {"$ref": "#/components/responses/NotFound"}, "409": {"$ref": "#/components/responses/Conflict"}}
      },
      "delete": {
        "summary": "Archive a managed message",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [{"$ref": "#/components/parameters/IdempotencyKey"}, {"$ref": "#/components/parameters/IfMatch"}],
        "responses": {"200": {"description": "Archived"}, "401": {"$ref": "#/components/responses/Unauthorized"}, "404": {"$ref": "#/components/responses/NotFound"}, "409": {"$ref": "#/components/responses/Conflict"}}
      }
    },
    "/api/messages/{key}/assignment": {
      "put": {
        "summary": "Assign a managed message to another channel",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [{"$ref": "#/components/parameters/MessageKey"}, {"$ref": "#/components/parameters/IdempotencyKey"}, {"$ref": "#/components/parameters/IfMatch"}],
        "requestBody": {"required": true, "content": {"application/json": {"schema": {"type": "object", "required": ["guildId", "channelId"], "properties": {"guildId": {"type": "string"}, "channelId": {"type": "string"}, "deleteReplacedMessage": {"type": "boolean", "default": false}}}}}},
        "responses": {"200": {"description": "Reassigned"}, "401": {"$ref": "#/components/responses/Unauthorized"}, "409": {"$ref": "#/components/responses/Conflict"}, "422": {"description": "Unsupported cleanup policy"}}
      }
    },
    "/api/messages/{key}/reconcile": {
      "post": {
        "summary": "Schedule immediate message reconciliation",
        "tags": ["Messages"],
        "security": [{"BearerAuth": []}],
        "parameters": [{"$ref": "#/components/parameters/MessageKey"}],
        "responses": {"202": {"description": "Reconciliation scheduled"}, "401": {"$ref": "#/components/responses/Unauthorized"}, "404": {"$ref": "#/components/responses/NotFound"}}
      }
    }
  },
  "components": {
    "securitySchemes": {"BearerAuth": {"type": "http", "scheme": "bearer"}},
    "parameters": {
      "MessageKey": {"name": "key", "in": "path", "required": true, "schema": {"type": "string", "pattern": "^[a-z0-9][a-z0-9_-]{0,63}$"}},
      "IdempotencyKey": {"name": "Idempotency-Key", "in": "header", "required": true, "schema": {"type": "string", "maxLength": 128}},
      "IfMatch": {"name": "If-Match", "in": "header", "required": true, "schema": {"type": "integer", "minimum": 1}}
    },
    "responses": {
      "BadRequest": {"description": "Invalid request", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}},
      "Unauthorized": {"description": "Missing or invalid bearer token", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}},
      "NotFound": {"description": "Unknown managed message", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}},
      "Conflict": {"description": "Idempotency or revision conflict", "content": {"application/json": {"schema": {"$ref": "#/components/schemas/Error"}}}}
    },
    "schemas": {
      "Error": {"type": "object", "required": ["error"], "properties": {"error": {"type": "string"}}},
      "StatusResponse": {
        "type": "object",
        "required": ["status", "environment", "version", "dependencies"],
        "properties": {"status": {"type": "string"}, "environment": {"type": "string"}, "version": {"type": "string"}, "dependencies": {"type": "object", "additionalProperties": {"type": "string", "enum": ["available", "unavailable"]}}}
      },
      "AllowedMentions": {
        "type": "object",
        "required": ["parse"],
        "properties": {"parse": {"type": "array", "items": {"type": "string", "enum": ["everyone", "roles", "users"]}}, "roles": {"type": "array", "items": {"type": "string"}}, "users": {"type": "array", "items": {"type": "string"}}, "repliedUser": {"type": "boolean"}}
      },
      "EmbedField": {
        "type": "object",
        "required": ["name", "value"],
        "properties": {"name": {"type": "string", "maxLength": 256}, "value": {"type": "string", "maxLength": 1024}, "inline": {"type": "boolean"}}
      },
      "EmbedMedia": {
        "type": "object",
        "required": ["url"],
        "properties": {"url": {"type": "string", "format": "uri"}}
      },
      "EmbedAuthor": {
        "type": "object",
        "required": ["name"],
        "properties": {"name": {"type": "string", "maxLength": 256}, "url": {"type": "string", "format": "uri"}, "iconUrl": {"type": "string", "format": "uri"}}
      },
      "EmbedFooter": {
        "type": "object",
        "required": ["text"],
        "properties": {"text": {"type": "string", "maxLength": 2048}, "iconUrl": {"type": "string", "format": "uri"}}
      },
      "Embed": {
        "type": "object",
        "properties": {"title": {"type": "string", "maxLength": 256}, "description": {"type": "string", "maxLength": 4096}, "url": {"type": "string", "format": "uri"}, "timestamp": {"type": "string", "format": "date-time"}, "color": {"type": "integer", "minimum": 0, "maximum": 16777215}, "footer": {"$ref": "#/components/schemas/EmbedFooter"}, "image": {"$ref": "#/components/schemas/EmbedMedia"}, "thumbnail": {"$ref": "#/components/schemas/EmbedMedia"}, "author": {"$ref": "#/components/schemas/EmbedAuthor"}, "fields": {"type": "array", "maxItems": 25, "items": {"$ref": "#/components/schemas/EmbedField"}}}
      },
      "Payload": {
        "type": "object",
        "required": ["content", "embeds", "allowedMentions"],
        "properties": {"content": {"type": "string", "maxLength": 2000}, "embeds": {"type": "array", "maxItems": 10, "items": {"$ref": "#/components/schemas/Embed"}}, "allowedMentions": {"$ref": "#/components/schemas/AllowedMentions"}}
      },
      "Definition": {
        "type": "object",
        "required": ["key", "guildId", "channelId", "payload"],
        "properties": {"key": {"type": "string"}, "guildId": {"type": "string"}, "channelId": {"type": "string"}, "payload": {"$ref": "#/components/schemas/Payload"}}
      },
      "Record": {
        "allOf": [{"$ref": "#/components/schemas/Definition"}, {"type": "object", "required": ["id", "desiredHash", "revision", "state", "failureCount", "createdAt", "updatedAt"], "properties": {"id": {"type": "string", "format": "uuid"}, "discordMessageId": {"type": "string"}, "desiredHash": {"type": "string"}, "observedHash": {"type": "string"}, "revision": {"type": "integer"}, "state": {"type": "string", "enum": ["pending", "healthy", "drifted", "repairing", "blocked", "archived"]}, "failureCount": {"type": "integer"}, "lastCheckedAt": {"type": "string", "format": "date-time"}, "lastRepairedAt": {"type": "string", "format": "date-time"}, "lastError": {"type": "string"}, "createdAt": {"type": "string", "format": "date-time"}, "updatedAt": {"type": "string", "format": "date-time"}}}]
      },
      "Page": {
        "type": "object",
        "required": ["items", "total", "limit", "offset"],
        "properties": {"items": {"type": "array", "items": {"$ref": "#/components/schemas/Record"}}, "total": {"type": "integer"}, "limit": {"type": "integer"}, "offset": {"type": "integer"}}
      }
    }
  }
}`
