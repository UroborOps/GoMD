# 🤖 Agent Integration Guide (MCP)

GoMD isn't just a web app—it's designed to function as the **perfect long-term memory backend for AI agents**. 

It achieves this through a **native Model Context Protocol (MCP) server** built directly into the Go backend. It operates entirely over **Server-Sent Events (SSE)**, meaning there are no extra Python scripts or dependencies to run! 

By connecting your favorite AI assistant to GoMD's SSE endpoints, the AI gains the ability to:
- **Read** your Markdown notes and metadata.
- **Search** through your entire vault using text search.
- **Write** and update files autonomously.
- **Explore** backlinks to understand relationships.

---

## 🔌 How to Connect Your AI Agents

### 1. Cursor IDE

Cursor natively supports connecting to MCP servers over SSE.

1. In Cursor, open **Settings** > **Features** > **MCP Servers**.
2. Click **Add New MCP Server**.
3. Configure it as follows:
   - **Type**: `sse`
   - **Name**: `GoMD`
   - **URL**: `http://localhost:3000/mcp/sse`
4. Click **Save**. When chatting with Cursor (`Cmd/Ctrl + L`), you can now ask it to "Search my GoMD vault for architecture notes" or "Save this code snippet to my GoMD vault."

### 2. Claude Desktop

Currently, Claude Desktop natively supports `stdio` (command-line) based MCP servers. To connect Claude Desktop to GoMD's SSE server, you can use the official `mcp-proxy` tool to bridge the connection:

1. Install `mcp-proxy` via npm: `npm install -g @modelcontextprotocol/proxy`
2. Update your `claude_desktop_config.json`:
```json
{
  "mcpServers": {
    "gomd": {
      "command": "mcp-proxy",
      "args": ["http://localhost:3000/mcp/sse"]
    }
  }
}
```

### 3. Custom Agents / Frameworks

If you are building your own agent framework, you can connect your standard MCP client directly to the GoMD SSE endpoints:

- **SSE Connection URL**: `http://localhost:3000/mcp/sse`
- **POST Message URL**: `http://localhost:3000/mcp/message`

GoMD strictly follows the standard JSON-RPC 2.0 over HTTP transport specified by the MCP protocol.

---

## 🛠️ Available MCP Tools

Once connected, your AI agent will automatically understand how to use the following tools:

| Tool Name | Description |
|---|---|
| `gomd_read_file` | Read the contents of a markdown file in the vault. |
| `gomd_write_file` | Create or update a markdown file in the vault. |
| `gomd_search` | Search the vault for files containing a specific query. |
| `gomd_list_backlinks` | List all files that link to a given file. |

---

## 🧠 Using GoMD in Headless Mode

If you are deploying GoMD *exclusively* for an AI agent to use (without needing the web UI for yourself), you can run it in Headless Mode to save resources.

In your `docker-compose.yml` or environment variables, set:
```env
GOMD_DISABLE_UI=true
```
The Go backend will stop serving the React frontend and act purely as a high-performance REST API, File Watcher, and SSE MCP server.
