---
name: gomd-mcp
description: MCP server development for GoMD — expose vault operations as MCP tools and resources for AI agent integration
tags: [gomd, mcp]
---

# GoMD MCP Skill

Create a Model Context Protocol (MCP) server that exposes GoMD vault operations to AI agents.

## MCP Server Structure

```
mcp/
  main.py          — MCP server entry point
    gomd.py          — MCP tools and resources
    requirements.txt — dependencies (httpx or urllib)
```

## MCP Tools

### `gomd_read_file`
Read a file from the vault.

```python
@mcp.tool(
    description="Read a markdown file from the GoMD vault. Returns the full content and frontmatter metadata."
)
def read_file(path: str) -> dict:
    """Read a file from the GoMD vault."""
    resp = requests.get(f"{GOMD_URL}/api/files/{urllib.parse.quote(path)}")
    if resp.status_code == 404:
        raise FileNotFoundError(f"File not found: {path}")
    return resp.json()  # { "path": "...", "content": "...", "frontmatter": {...} }
```

### `gomd_write_file`
Create or update a file in the vault.

```python
@mcp.tool(
    description="Create or update a markdown file in the GoMD vault. Creates parent directories if needed."
)
def write_file(path: str, content: str) -> dict:
    """Create or update a file in the GoMD vault."""
    resp = requests.put(f"{GOMD_URL}/api/files/{urllib.parse.quote(path)}", json={"content": content})
    if resp.status_code in (200, 201):
        return {"success": True, "path": path}
    raise Exception(f"Failed to write file: {resp.text}")
```

### `gomd_delete_file`
Delete a file from the vault.

```python
@mcp.tool(
    description="Delete a markdown file from the GoMD vault."
)
def delete_file(path: str) -> dict:
    """Delete a file from the GoMD vault."""
    resp = requests.delete(f"{GOMD_URL}/api/files/{urllib.parse.quote(path)}")
    if resp.status_code == 200:
        return {"success": True, "path": path}
    raise Exception(f"Failed to delete file: {resp.text}")
```

### `gomd_search`
Search the vault.

```python
@mcp.tool(
    description="Search the GoMD vault. Supports fuzzy matching and tag filtering. Use 'tag:design' to filter by tag."
)
def search(query: str) -> dict:
    """Search the GoMD vault."""
    resp = requests.get(f"{GOMD_URL}/api/search", params={"q": query})
    return resp.json()  # { "results": [{ "path": "...", "snippet": "...", "score": 0.95 }] }
```

### `gomd_backlinks`
Get backlinks for a file.

```python
@mcp.tool(
    description="Find all files that link to the specified file in the GoMD vault."
)
def backlinks(path: str) -> dict:
    """Get backlinks for a file."""
    resp = requests.get(f"{GOMD_URL}/api/backlinks/{urllib.parse.quote(path)}")
    return resp.json()  # { "path": "...", "backlinks": [{"path": "...", "heading": "..."}] }
```

### `gomd_list_files`
List files in the vault.

```python
@mcp.tool(
    description="List all markdown files in the GoMD vault. Optionally filter by directory path."
)
def list_files(path: str = "") -> dict:
    """List files in the GoMD vault."""
    resp = requests.get(f"{GOMD_URL}/api/files", params={"path": path} if path else {})
    return resp.json()  # { "files": [{"path": "...", "name": "..."}, ...] }
```

## MCP Resources

### `gomd://files/{path}`
Readable resource for file contents.

```python
@mcp.resource("gomd://files/{path}")
def get_file_resource(path: str) -> str:
    """Provides the content of a file from the GoMD vault."""
    resp = requests.get(f"{GOMD_URL}/api/files/{urllib.parse.quote(path)}")
    return resp.json()["content"]
```

### `gomd://graph`
Readable resource for the knowledge graph.

```python
@mcp.resource("gomd://graph")
def get_graph_resource() -> str:
    """Provides the knowledge graph data (nodes and edges) from the GoMD vault."""
    resp = requests.get(f"{GOMD_URL}/api/graph")
    return json.dumps(resp.json())  # { "nodes": [...], "edges": [...] }
```

## MCP Prompt

### `gomd-context`
Get context about a topic by searching the vault.

```python
@mcp.prompt()
def gomd_context(topic: str) -> list[dict]:
    """Search the GoMD vault for information about a topic and return the most relevant file contents as context."""
    results = search(topic)
    messages = []
    for r in results[:3]:  # Top 3 results
        content = read_file(r["path"])
        messages.append({
            "role": "user",
            "content": f"Context from {r['path']}:\n\n{content['content']}"
        })
    return messages
```

## Configuration

The MCP server reads config from:
- Env var: `GOMD_URL` (default: `http://localhost:3000`)
- Config file: `~/.gomd/mcp.json`

```json
{
  "url": "http://localhost:3000",
  "vault_path": "/home/user/Documents/vault",
  "timeout": 30
}
```

## Integration

Add to your MCP client config (e.g., Claude Desktop, Cursor):

```json
{
  "mcpServers": {
    "gomd": {
      "command": "python",
      "args": ["~/code/gomd/mcp/main.py"],
      "env": {
        "GOMD_URL": "http://192.168.1.XXX:3000"
      }
    }
  }
}
```

## Error Handling

- Network errors → `ConnectionError` with vault URL
- 404 → `FileNotFoundError`
- Auth errors → `AuthenticationError` (if vault is behind auth)
- Rate limits → `RateLimitError` with retry-after
