// Package openapi contains the documented discord-bot HTTP contract.
package openapi

// Spec is the OpenAPI 3.0 document served by the HTTP API.
const Spec = `{
  "openapi":"3.0.3",
  "info":{"title":"discord-bot API","description":"Managed Discord messages and SARP business performance.","version":"1.2.0"},
  "paths":{
    "/status":{"get":{"tags":["Public"],"summary":"Read dependency status","responses":{"200":{"description":"Service status"}}}},
    "/api/performance":{"get":{"tags":["Performance"],"summary":"Read the persisted business performance aggregate","security":[{"BearerAuth":[]}],"responses":{"200":{"description":"Current performance state"},"404":{"$ref":"#/components/responses/NotFound"},"503":{"description":"Performance collection is disabled"}}}},
    "/api/performance/refresh":{"post":{"tags":["Performance"],"summary":"Collect SARP counters and update the Discord dashboard now","security":[{"BearerAuth":[]}],"responses":{"200":{"description":"Updated performance state"},"502":{"description":"Upstream or publication failed"},"503":{"description":"Performance collection is disabled"}}}},
    "/api/messages":{"post":{"tags":["Messages"],"summary":"Create a managed Components V2 message","security":[{"BearerAuth":[]}],"parameters":[{"$ref":"#/components/parameters/IdempotencyKey"}],"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/MessageDefinition"}}}},"responses":{"201":{"description":"Created"},"400":{"$ref":"#/components/responses/BadRequest"},"409":{"$ref":"#/components/responses/Conflict"}}},"get":{"tags":["Messages"],"summary":"List managed messages","security":[{"BearerAuth":[]}],"responses":{"200":{"description":"Message page"}}}},
    "/api/messages/{key}":{"parameters":[{"$ref":"#/components/parameters/MessageKey"}],"get":{"tags":["Messages"],"summary":"Read a managed message","security":[{"BearerAuth":[]}],"responses":{"200":{"description":"Message"},"404":{"$ref":"#/components/responses/NotFound"}}},"put":{"tags":["Messages"],"summary":"Replace Components V2 desired state","security":[{"BearerAuth":[]}],"parameters":[{"$ref":"#/components/parameters/IdempotencyKey"},{"$ref":"#/components/parameters/IfMatch"}],"requestBody":{"required":true,"content":{"application/json":{"schema":{"$ref":"#/components/schemas/MessageDefinition"}}}},"responses":{"200":{"description":"Updated"},"409":{"$ref":"#/components/responses/Conflict"}}},"delete":{"tags":["Messages"],"summary":"Archive a managed message","security":[{"BearerAuth":[]}],"parameters":[{"$ref":"#/components/parameters/IdempotencyKey"},{"$ref":"#/components/parameters/IfMatch"}],"responses":{"200":{"description":"Archived"}}}},
    "/api/messages/{key}/assignment":{"put":{"tags":["Messages"],"summary":"Assign a message to a channel in the configured guild","security":[{"BearerAuth":[]}],"parameters":[{"$ref":"#/components/parameters/MessageKey"},{"$ref":"#/components/parameters/IdempotencyKey"},{"$ref":"#/components/parameters/IfMatch"}],"responses":{"200":{"description":"Assigned"}}}},
    "/api/messages/{key}/reconcile":{"post":{"tags":["Messages"],"summary":"Schedule message reconciliation","security":[{"BearerAuth":[]}],"parameters":[{"$ref":"#/components/parameters/MessageKey"}],"responses":{"202":{"description":"Scheduled"}}}}
  },
  "components":{
    "securitySchemes":{"BearerAuth":{"type":"http","scheme":"bearer"}},
    "parameters":{
      "MessageKey":{"name":"key","in":"path","required":true,"schema":{"type":"string","pattern":"^[a-z0-9][a-z0-9_-]{0,63}$"}},
      "IdempotencyKey":{"name":"Idempotency-Key","in":"header","required":true,"schema":{"type":"string","maxLength":128}},
      "IfMatch":{"name":"If-Match","in":"header","required":true,"schema":{"type":"integer","minimum":1}}
    },
    "responses":{"BadRequest":{"description":"Invalid request"},"Unauthorized":{"description":"Missing or invalid API key"},"NotFound":{"description":"Unknown record"},"Conflict":{"description":"Duplicate or stale record"}},
    "schemas":{
      "AllowedMentions":{"type":"object","required":["parse"],"properties":{"parse":{"type":"array","items":{"type":"string","enum":["everyone","roles","users"]}},"roles":{"type":"array","maxItems":100,"items":{"type":"string"}},"users":{"type":"array","maxItems":100,"items":{"type":"string"}},"repliedUser":{"type":"boolean"}}},
      "V2Component":{"type":"object","description":"Discord Components V2 object."},
      "V2Payload":{"type":"object","required":["components","allowedMentions"],"properties":{"components":{"type":"array","minItems":1,"items":{"$ref":"#/components/schemas/V2Component"}},"allowedMentions":{"$ref":"#/components/schemas/AllowedMentions"}}},
      "MessageDefinition":{"type":"object","required":["key","guildId","channelId","payload"],"properties":{"key":{"type":"string"},"guildId":{"type":"string"},"channelId":{"type":"string"},"payload":{"$ref":"#/components/schemas/V2Payload"}}}
    }
  }
}`
