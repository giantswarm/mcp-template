# The tools

Generated from the registered tools (`make tools-doc`); through muster every tool is `x_<server>_<name>`.

| Tool | Kind |
|---|---|
| `things_create` | write |
| `things_get` | read-only |
| `things_list` | read-only |

## `things_create`

```text
Create a new thing.

Each call adds a new thing, even for a name that exists: look it up with things_list first.
```

```json
{
  "properties": {
    "name": {
      "description": "Human-readable name for the new thing.",
      "type": "string"
    }
  },
  "required": [
    "name"
  ],
  "type": "object"
}
```

## `things_get`

```text
Fetch one thing by id.
```

```json
{
  "properties": {
    "id": {
      "description": "Thing id, e.g. thing-3.",
      "type": "string"
    }
  },
  "required": [
    "id"
  ],
  "type": "object"
}
```

## `things_list`

```text
List every thing the server knows about.
```

```json
{
  "properties": {},
  "required": [],
  "type": "object"
}
```
