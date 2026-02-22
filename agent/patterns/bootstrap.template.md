# MCP Server Bootstrap Pattern

## Overview

This document describes the organizational patterns for bootstrapping...

## Core Principles

Core principles...

## Project Structure

```
project-root/
├── src/
│   ├── index.ts                    # CLI entry point (bundled)
│   ├── server.ts                   # Server class (for standalone)
│   ├── server-factory.ts           # Factory function (for multi-tenant)
│   ├── client.ts                   # External API client wrapper
│   ├── types.ts                    # Shared type definitions
│   │
│   ├── tools/                      # Tool definitions
│   │   ├── index.ts                # Tool exports
│   │   ├── tool-one.ts             # Individual tool (definition + handler)
│   │   ├── tool-two.ts
│   │   └── ...
│   │
│   ├── types/                      # Type definitions (optional subdirectory)
│   │   ├── mcp.ts                  # MCP-specific types
│   │   ├── api.ts                  # External API types
│   │   └── ...
│   │
│   └── utils/                      # Utilities
│       ├── logger.ts               # Logging (stdio-safe for MCP)
│       ├── error-serializer.ts     # Error handling
│       └── ...
│
├── agent/                          # Documentation & planning
│   ├── patterns/                   # Architecture patterns
│   ├── tasks/                      # Task tracking
│   └── ...
│
├── package.json                    # Package configuration
├── tsconfig.json                   # TypeScript configuration
├── esbuild.build.js                # Build script
├── esbuild.watch.js                # Watch mode script
├── .gitignore
└── README.md
```

## Configuration Files

### package.json Structure

For a simple MCP server:

```json
{
  "name": "remember-mcp",
  "version": "0.1.0",
  "description": "Multi-tenant memory system MCP server with vector search and relationships",
  "main": "dist/server.js",
  "type": "module",
  "repository": {
    "type": "git",
    "url": "git+https://github.com/prmichaelsen/remember-mcp.git"
  },
  "bugs": {
    "url": "https://github.com/prmichaelsen/remember-mcp/issues"
  },
  "homepage": "https://github.com/prmichaelsen/remember-mcp#readme",
  "scripts": {
    "build": "node esbuild.build.js",
    "build:watch": "node esbuild.watch.js",
    "clean": "rm -rf dist",
    "dev": "tsx watch src/server.ts",
    "start": "node dist/server.js",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:e2e": "jest --config jest.e2e.config.js",
    "test:e2e:watch": "jest --config jest.e2e.config.js --watch",
    "test:all": "npm test && npm run test:e2e",
    "lint": "eslint src/**/*.ts",
    "typecheck": "tsc --noEmit",
    "prepublishOnly": "npm run clean && npm run build"
  },
  "keywords": [
    "mcp",
    "memory",
    "vector-search",
    "weaviate",
    "firebase"
  ],
  "author": "Patrick Michaelsen",
  "license": "MIT"
}
```

For a library with multiple exports:

```json
{
  "name": "@scope/package-name",
  "version": "1.0.0",
  "description": "MCP server for [purpose]",
  "type": "module",
  "main": "dist/index.js",
  "types": "dist/index.d.ts",
  
  "repository": {
    "type": "git",
    "url": "git+https://github.com/username/repo.git"
  },
  "bugs": {
    "url": "https://github.com/username/repo/issues"
  },
  "homepage": "https://github.com/username/repo#readme",
  
  "exports": {
    ".": {
      "types": "./dist/index.d.ts",
      "import": "./dist/index.js"
    },
    "./factory": {
      "types": "./dist/server-factory.d.ts",
      "import": "./dist/server-factory.js"
    },
    "./client": {
      "types": "./dist/client.d.ts",
      "import": "./dist/client.js"
    },
    "./tools": {
      "types": "./dist/tools/index.d.ts",
      "import": "./dist/tools/index.js"
    },
    "./types": {
      "types": "./dist/types.d.ts",
      "import": "./dist/types.js"
    }
  },
  
  "files": [
    "dist",
    "README.md",
    "LICENSE"
  ],
  
  "scripts": {
    "build": "npm run build:types && npm run build:bundle",
    "build:types": "tsc --emitDeclarationOnly",
    "build:bundle": "node esbuild.build.js",
    "build:watch": "node esbuild.watch.js",
    "start": "node dist/index.js",
    "clean": "rm -rf dist",
    "test": "jest",
    "test:watch": "jest --watch",
    "test:e2e": "jest --config jest.e2e.config.js",
    "test:e2e:watch": "jest --config jest.e2e.config.js --watch",
    "test:all": "npm test && npm run test:e2e",
    "prepublishOnly": "npm run clean && npm run build"
  },
  
  "keywords": [
    "mcp",
    "model-context-protocol",
    "[domain-specific-keywords]"
  ],
  
  "dependencies": {
    "@modelcontextprotocol/sdk": "^1.0.0"
  },
  
  "devDependencies": {
    "@types/jest": "^30.0.0",
    "@types/node": "^20.0.0",
    "esbuild": "^0.25.0",
    "jest": "^30.0.0",
    "ts-jest": "^29.0.0",
    "typescript": "^5.3.0"
  },
  
  "engines": {
    "node": ">=18.0.0"
  }
}
```

**Key Points:**
- `"type": "module"` required for ESM
- `repository`, `bugs`, `homepage` for GitHub integration
- `main` points to the built entry file
- `exports` field for libraries with multiple entry points
- `files` array specifies what to publish to npm
- `build:watch` script for development
- `clean` script removes build artifacts
- `prepublishOnly` ensures clean build before publishing
- `test` scripts for jest (unit and e2e)
- `author` field for attribution
- `engines` specifies minimum Node.js version

### tsconfig.json Structure

```json
{
  "compilerOptions": {
    "target": "ES2022",
    "module": "Node16",
    "moduleResolution": "Node16",
    "lib": ["ES2022"],
    "types": ["node"],
    
    "outDir": "./dist",
    "rootDir": "./src",
    
    "strict": true,
    "esModuleInterop": true,
    "skipLibCheck": true,
    "forceConsistentCasingInFileNames": true,
    "resolveJsonModule": true,
    
    "declaration": true,
    "declarationMap": true,
    "sourceMap": true,
    
    "baseUrl": ".",
    "paths": {
      "@/*": ["src/*"]
    }
  },
  
  "include": ["src/**/*"],
  "exclude": ["node_modules", "dist"]
}
```

**Key Points:**
- `Node16` module resolution for proper ESM support
- `declaration: true` for type definitions
- `strict: true` for type safety
- Source maps for debugging
- `baseUrl` and `paths` for module name mapping (`@/` → `src/`)

### Jest Configuration

For projects with colocated tests (`.spec.ts` and `.e2e.ts` files alongside source code):

#### jest.config.js - Unit Tests

```javascript
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  roots: ['<rootDir>/src'],
  testMatch: ['**/*.spec.ts'],
  moduleFileExtensions: ['ts', 'js'],
  collectCoverage: true,
  coverageDirectory: 'coverage',
  coverageReporters: ['text', 'lcov', 'html'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.d.ts',
    '!src/**/*.spec.ts',
    '!src/**/*.e2e.ts',
    '!src/index.ts',              // Barrel export only
    '!src/types/**/*.ts',         // Type definitions only
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};
```

#### jest.e2e.config.js - E2E Tests

```javascript
module.exports = {
  preset: 'ts-jest',
  testEnvironment: 'node',
  testMatch: ['**/*.e2e.ts'],
  testTimeout: 30000, // 30 seconds for real API calls
  roots: ['<rootDir>/src'],
  collectCoverageFrom: [
    'src/**/*.ts',
    '!src/**/*.spec.ts',
    '!src/**/*.e2e.ts',
    '!src/types/**/*.ts',
    '!src/index.ts',
  ],
  moduleNameMapper: {
    '^@/(.*)$': '<rootDir>/src/$1',
  },
};
```

**Key Points:**
- Separate configs for unit tests (`.spec.ts`) and e2e tests (`.e2e.ts`)
- E2E tests have longer timeout for real API calls
- Coverage excludes test files and type definitions
- `moduleNameMapper` matches TypeScript path aliases
- Tests are colocated with source files in `src/`

**Package.json Scripts:**
```json
{
  "scripts": {
    "test": "jest --config jest.config.js",
    "test:e2e": "jest --config jest.e2e.config.js",
    "test:watch": "jest --config jest.config.js --watch",
    "test:coverage": "jest --config jest.config.js --coverage"
  },
  "devDependencies": {
    "@types/jest": "^29.0.0",
    "jest": "^29.0.0",
    "ts-jest": "^29.0.0"
  }
}
```

### esbuild.build.js Structure

For a simple MCP server (single entry point):

```javascript
import * as esbuild from 'esbuild';

await esbuild.build({
  entryPoints: ['src/server.ts'],
  bundle: true,
  platform: 'node',
  target: 'node20',
  format: 'esm',
  outfile: 'dist/server.js',
  sourcemap: true,
  external: [
    'weaviate-client',
    'firebase-admin',
    '@modelcontextprotocol/sdk'
  ],
  banner: {
    js: "import { createRequire } from 'module'; const require = createRequire(import.meta.url);"
  },
  alias: {
    '@': './src'
  }
});

console.log('✓ Build complete');
```

For a library with multiple entry points:

```javascript
import * as esbuild from 'esbuild';
import { readdir } from 'fs/promises';
import { join } from 'path';

// Option 1: Find all entry points dynamically
async function findEntryPoints(dir, base = 'src') {
  const entries = [];
  const files = await readdir(dir, { withFileTypes: true });
  
  for (const file of files) {
    const fullPath = join(dir, file.name);
    if (file.isDirectory()) {
      entries.push(...await findEntryPoints(fullPath, base));
    } else if (file.name.endsWith('.ts') && !file.name.endsWith('.d.ts')) {
      entries.push(fullPath);
    }
  }
  
  return entries;
}

// Option 2: Explicit entry points
const explicitEntryPoints = [
  'src/server-factory.ts',
  'src/client.ts',
  'src/types.ts',
  'src/tools/tool-one.ts',
  'src/tools/tool-two.ts'
];

// Build CLI entry point (bundled)
await esbuild.build({
  entryPoints: ['src/index.ts'],
  bundle: true,
  outfile: 'dist/index.js',
  platform: 'node',
  target: 'node18',
  format: 'esm',
  sourcemap: true,
  external: [
    '@modelcontextprotocol/sdk',
    // Add other peer dependencies
  ],
  banner: {
    js: "import { createRequire } from 'module'; const require = createRequire(import.meta.url);"
  },
  alias: {
    '@': './src'
  },
  minify: false,
  keepNames: true
});

// Build library exports (unbundled, preserves module structure)
await esbuild.build({
  entryPoints: await findEntryPoints('src'), // or explicitEntryPoints
  bundle: false,  // Key: don't bundle for library
  outdir: 'dist',
  outbase: 'src', // Preserve directory structure
  platform: 'node',
  target: 'node18',
  format: 'esm',
  sourcemap: true,
  alias: {
    '@': './src'
  }
});

console.log('Build complete!');
```

**Key Points:**
- **Simple servers**: Single bundled entry point
- **Library exports**: Dual build strategy (bundle CLI, preserve modules)
- `bundle: true` for standalone executable
- `bundle: false` + `outbase: 'src'` for library exports
- `external` array lists dependencies not to bundle (peer dependencies)
- `banner` adds CommonJS compatibility for ESM bundles
- `alias` enables path alias resolution (`@/` → `src/`)
- `target` specifies Node.js version compatibility
- Dynamic or explicit entry point discovery for libraries

### esbuild.watch.js Structure

```javascript
import * as esbuild from 'esbuild';

const ctx = await esbuild.context({
  entryPoints: ['src/server.ts'],
  bundle: true,
  platform: 'node',
  target: 'node20',
  format: 'esm',
  outfile: 'dist/server.js',
  sourcemap: true,
  external: [
    'weaviate-client',
    'firebase-admin',
    '@modelcontextprotocol/sdk'
  ],
  banner: {
    js: "import { createRequire } from 'module'; const require = createRequire(import.meta.url);"
  },
  alias: {
    '@': './src'
  }
});

await ctx.watch();
console.log('👀 Watching for changes...');
```

**Key Points:**
- Uses `esbuild.context()` API for watch mode
- Same configuration as `esbuild.build.js` for consistency
- Automatically rebuilds on file changes
- Includes all the same options: `external`, `banner`, `alias`, etc.

## Source Code Patterns

### Tool Definition Pattern

Each tool file exports both definition and handler:

```typescript
// src/tools/example-tool.ts
import { ClientWrapper } from '../client.js';

export const exampleTool = {
  name: 'prefix_tool_name',
  description: 'Clear description of what the tool does',
  inputSchema: {
    type: 'object',
    properties: {
      param1: {
        type: 'string',
        description: 'Parameter description'
      },
      param2: {
        type: 'number',
        description: 'Optional parameter',
        default: 10
      }
    },
    required: ['param1']
  }
};

export async function handleExampleTool(
  client: ClientWrapper,
  args: any
): Promise<string> {
  try {
    const result = await client.doSomething(args.param1, args.param2);
    return JSON.stringify(result, null, 2);
  } catch (error) {
    throw new Error(`Failed to execute: ${error instanceof Error ? error.message : String(error)}`);
  }
}
```

**Key Points:**
- Tool definition is a plain object (MCP Tool schema)
- Handler is a separate async function
- Handler receives client instance and args
- Returns JSON string for MCP response
- Proper error handling

### Server Factory Pattern (Multi-Tenant)

For use with `mcp-auth` or other multi-tenant wrappers:

```typescript
// src/server-factory.ts
import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { ClientWrapper } from './client.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
  ErrorCode,
  McpError
} from '@modelcontextprotocol/sdk/types.js';

// Import all tools
import { toolOne, handleToolOne } from './tools/tool-one.js';
import { toolTwo, handleToolTwo } from './tools/tool-two.js';

export interface ServerOptions {
  name?: string;
  version?: string;
}

/**
 * Create a server instance for a specific user/tenant
 * 
 * @param accessToken - User's access token for external API
 * @param userId - User identifier
 * @param options - Optional server configuration
 * @returns Configured MCP Server instance
 */
export function createServer(
  accessToken: string,
  userId: string,
  options: ServerOptions = {}
): Server {
  if (!accessToken) {
    throw new Error('accessToken is required');
  }
  
  if (!userId) {
    throw new Error('userId is required');
  }
  
  // Initialize client with user's credentials
  const client = new ClientWrapper(accessToken);
  
  // Create MCP server
  const server = new Server(
    {
      name: options.name || 'mcp-server',
      version: options.version || '1.0.0'
    },
    {
      capabilities: {
        tools: {}
      }
    }
  );
  
  // Register list_tools handler
  server.setRequestHandler(ListToolsRequestSchema, async () => {
    return {
      tools: [
        toolOne,
        toolTwo,
        // ... all tool definitions
      ]
    };
  });
  
  // Register call_tool handler
  server.setRequestHandler(CallToolRequestSchema, async (request) => {
    const { name, arguments: args } = request.params;
    
    try {
      let result: string;
      
      switch (name) {
        case 'prefix_tool_one':
          result = await handleToolOne(client, args);
          break;
        
        case 'prefix_tool_two':
          result = await handleToolTwo(client, args);
          break;
        
        default:
          throw new McpError(
            ErrorCode.MethodNotFound,
            `Unknown tool: ${name}`
          );
      }
      
      return {
        content: [
          {
            type: 'text',
            text: result
          }
        ]
      };
    } catch (error) {
      if (error instanceof McpError) {
        throw error;
      }
      
      throw new McpError(
        ErrorCode.InternalError,
        `Tool execution failed: ${error instanceof Error ? error.message : String(error)}`
      );
    }
  });
  
  return server;
}
```

**Key Points:**
- Factory function creates isolated server instances
- Each instance has its own client with user credentials
- No shared state between instances
- Compatible with `mcp-auth` wrapping pattern

### Standalone Server Pattern

For direct stdio usage without multi-tenancy:

```typescript
// src/server.ts
#!/usr/bin/env node

import { Server } from '@modelcontextprotocol/sdk/server/index.js';
import { StdioServerTransport } from '@modelcontextprotocol/sdk/server/stdio.js';
import {
  CallToolRequestSchema,
  ListToolsRequestSchema,
} from '@modelcontextprotocol/sdk/types.js';
import { config } from 'dotenv';
import { ClientWrapper } from './client.js';
import { logger } from './utils/logger.js';

// Import tools
import { ToolOne } from './tools/tool-one.js';
import { ToolTwo } from './tools/tool-two.js';

// Load environment variables
config();

class MCPServer {
  private server: Server;
  private client: ClientWrapper;
  private toolOne: ToolOne;
  private toolTwo: ToolTwo;

  constructor() {
    // Initialize server
    this.server = new Server(
      {
        name: 'mcp-server',
        version: '1.0.0',
      },
      {
        capabilities: {
          tools: {},
        },
      }
    );

    // Initialize client
    const apiKey = process.env.API_KEY;
    if (!apiKey) {
      throw new Error('API_KEY environment variable is required');
    }
    
    this.client = new ClientWrapper(apiKey);
    
    // Initialize tools
    this.toolOne = new ToolOne(this.client);
    this.toolTwo = new ToolTwo(this.client);

    this.setupHandlers();
  }

  private setupHandlers(): void {
    // List available tools
    this.server.setRequestHandler(ListToolsRequestSchema, async () => {
      return {
        tools: [
          this.toolOne.getToolDefinition(),
          this.toolTwo.getToolDefinition(),
        ],
      };
    });

    // Handle tool calls
    this.server.setRequestHandler(CallToolRequestSchema, async (request) => {
      const { name, arguments: args } = request.params;

      try {
        let result: any;
        
        switch (name) {
          case 'prefix_tool_one':
            result = await this.toolOne.execute(args);
            break;

          case 'prefix_tool_two':
            result = await this.toolTwo.execute(args);
            break;

          default:
            throw new Error(`Unknown tool: ${name}`);
        }

        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify(result, null, 2),
            },
          ],
        };
      } catch (error) {
        logger.error(`Tool execution failed for ${name}:`, error);
        return {
          content: [
            {
              type: 'text',
              text: JSON.stringify({
                error: error instanceof Error ? error.message : 'Unknown error',
                tool: name
              }, null, 2),
            },
          ],
          isError: true,
        };
      }
    });
  }

  async start(): Promise<void> {
    try {
      logger.info('Starting MCP Server...');

      // Connect to external service
      await this.client.connect();

      // Start MCP server with stdio transport
      const transport = new StdioServerTransport();
      await this.server.connect(transport);

      // Don't log to stdout/stderr when using stdio transport
      // It interferes with MCP JSON protocol

    } catch (error) {
      process.exit(1);
    }
  }

  async stop(): Promise<void> {
    await this.server.close();
  }
}

// Handle graceful shutdown
const server = new MCPServer();

process.on('SIGINT', async () => {
  await server.stop();
  process.exit(0);
});

process.on('SIGTERM', async () => {
  await server.stop();
  process.exit(0);
});

// Start the server
server.start().catch((error) => {
  process.exit(1);
});
```

**Key Points:**
- Class-based server for encapsulation
- Environment variable configuration
- Graceful shutdown handling
- **Critical**: No stdout/stderr logging when using stdio transport
- Tool instances as class properties

### Client Wrapper Pattern

```typescript
// src/client.ts
export interface ClientConfig {
  apiKey: string;
  baseUrl?: string;
  timeout?: number;
}

export class ClientWrapper {
  private config: ClientConfig;
  private isConnected = false;

  constructor(apiKey: string, options?: Partial<ClientConfig>) {
    this.config = {
      apiKey,
      baseUrl: options?.baseUrl || 'https://api.example.com',
      timeout: options?.timeout || 30000
    };
  }

  async connect(): Promise<void> {
    // Initialize connection, validate credentials, etc.
    this.isConnected = true;
  }

  async doSomething(param: string): Promise<any> {
    if (!this.isConnected) {
      throw new Error('Client not connected');
    }
    
    // Make API call
    const response = await fetch(`${this.config.baseUrl}/endpoint`, {
      method: 'POST',
      headers: {
        'Authorization': `Bearer ${this.config.apiKey}`,
        'Content-Type': 'application/json'
      },
      body: JSON.stringify({ param })
    });
    
    if (!response.ok) {
      throw new Error(`API error: ${response.statusText}`);
    }
    
    return response.json();
  }

  isClientConnected(): boolean {
    return this.isConnected;
  }
}
```

**Key Points:**
- Encapsulates external API communication
- Accepts credentials in constructor (for multi-tenant)
- Connection state management
- Error handling

### Logger Pattern (Stdio-Safe)

```typescript
// src/utils/logger.ts
export enum LogLevel {
  ERROR = 0,
  WARN = 1,
  INFO = 2,
  DEBUG = 3
}

class Logger {
  // No-op logger to avoid interfering with stdio MCP transport
  // All logging methods do nothing to prevent JSON corruption

  error(message: string, ...args: any[]): void {
    // No-op when using stdio
    // Could write to file or use process.stderr in non-stdio mode
  }

  warn(message: string, ...args: any[]): void {
    // No-op
  }

  info(message: string, ...args: any[]): void {
    // No-op
  }

  debug(message: string, ...args: any[]): void {
    // No-op
  }
}

export const logger = new Logger();
```

**Key Points:**
- **Critical**: No console output when using stdio transport
- Stdio transport uses stdout/stdin for JSON-RPC
- Any console output corrupts the protocol
- Alternative: Write to file or use stderr carefully

### Error Serializer Pattern

```typescript
// src/utils/error-serializer.ts
export function serializeError(error: unknown): any {
  if (error instanceof Error) {
    return {
      name: error.name,
      message: error.message,
      stack: error.stack,
      ...(error as any) // Include any additional properties
    };
  }
  
  return {
    message: String(error)
  };
}
```

## Integration with mcp-auth

### Using AuthenticatedMCPServer

If building a new server with tool-level auth:

```typescript
// src/index.ts
import { AuthenticatedMCPServer } from '@prmichaelsen/mcp-auth/server';
import { EnvAuthProvider } from '@prmichaelsen/mcp-auth/providers/env';
import { SimpleTokenResolver } from '@prmichaelsen/mcp-auth';
import { withAuth } from '@prmichaelsen/mcp-auth/server';

const server = new AuthenticatedMCPServer({
  name: 'my-server',
  authProvider: new EnvAuthProvider(),
  tokenResolver: new SimpleTokenResolver({ tokenEnvVar: 'API_TOKEN' }),
  resourceType: 'myapi',
  transport: { type: 'stdio' }
});

server.registerTool('get_data', withAuth(async (args, accessToken, userId) => {
  const client = new ClientWrapper(accessToken);
  return client.getData(args);
}));

await server.start();
```

### Using Server Wrapping Pattern

If wrapping an existing server factory:

```typescript
// Wrapper server using mcp-auth
import { wrapServer } from '@prmichaelsen/mcp-auth/wrapper';
import { createServer } from './server-factory.js';

const wrappedServer = wrapServer({
  serverFactory: createServer,
  authProvider: new JWTAuthProvider({ secret: process.env.JWT_SECRET }),
  tokenResolver: new APITokenResolver({ apiUrl: process.env.API_URL }),
  resourceType: 'myapi',
  transport: { type: 'sse', port: 3000 }
});

await wrappedServer.start();
```

## Directory Organization Best Practices

### Agent Directory

The `agent/` directory contains documentation and planning:

```
agent/
├── patterns/                    # Architecture patterns
│   ├── bootstrap.md             # This document
│   ├── library-services.md      # Service layer patterns
│   └── ...
│
├── tasks/                       # Task tracking
│   ├── task-001.md
│   └── ...
│
├── milestones/                  # Milestone planning
│   ├── milestone-1.md
│   └── ...
│
├── progress.yaml                # Progress tracking
└── requirements.md              # Requirements document
```

### Types Organization

Types can be organized in two ways:

**Option 1: Flat structure** (simple projects)
```
src/
├── types.ts                     # All types in one file
└── ...
```

**Option 2: Types directory** (complex projects)
```
src/
├── types/
│   ├── mcp.ts                   # MCP-specific types
│   ├── api.ts                   # External API types
│   ├── domain.ts                # Domain types
│   └── index.ts                 # Re-exports
└── ...
```

### Utils Organization

```
src/
├── utils/
│   ├── logger.ts                # Logging utility
│   ├── error-serializer.ts      # Error handling
│   ├── validation.ts            # Input validation
│   └── index.ts                 # Re-exports
└── ...
```

## Build Output Structure

After building, the output should mirror the source structure:

```
dist/
├── index.js                     # Bundled CLI entry
├── index.d.ts
├── server-factory.js            # Unbundled library exports
├── server-factory.d.ts
├── client.js
├── client.d.ts
├── types.js
├── types.d.ts
├── tools/
│   ├── tool-one.js
│   ├── tool-one.d.ts
│   ├── tool-two.js
│   ├── tool-two.d.ts
│   └── index.js
└── utils/
    ├── logger.js
    ├── logger.d.ts
    └── ...
```

**Key Points:**
- `index.js` is bundled (single file)
- Other exports preserve module structure
- Type definitions (`.d.ts`) for all modules
- Source maps (`.js.map`) for debugging

## Import Patterns

### ESM Import Extensions

Always include `.js` extension in imports (even for `.ts` files):

```typescript
// ✅ Correct
import { ClientWrapper } from './client.js';
import { toolOne } from './tools/tool-one.js';

// ❌ Wrong
import { ClientWrapper } from './client';
import { toolOne } from './tools/tool-one';
```

### Re-export Patterns

```typescript
// src/tools/index.ts
export * from './tool-one.js';
export * from './tool-two.js';

// Usage
import { toolOne, toolTwo } from './tools/index.js';
```

## Environment Configuration

### .env.example

```bash
# API Configuration
API_KEY=your_api_key_here
API_URL=https://api.example.com

# Server Configuration
PORT=3000
NODE_ENV=development

# Logging
LOG_LEVEL=info
```

### Environment Loading

```typescript
import { config } from 'dotenv';

// Load at server startup
config();

// Access variables
const apiKey = process.env.API_KEY;
if (!apiKey) {
  throw new Error('API_KEY is required');
}
```

## Testing Considerations

While not covered in detail, consider:

```
src/
├── tools/
│   ├── tool-one.ts
│   ├── tool-one.test.ts         # Co-located tests
│   └── ...
```

Or separate test directory:

```
tests/
├── tools/
│   ├── tool-one.test.ts
│   └── ...
└── integration/
    └── ...
```

## Common Patterns Summary

### 1. Tool Organization
- One file per tool
- Export definition and handler separately
- Handler receives client and args
- Return JSON strings

### 2. Server Patterns
- **Factory**: For multi-tenant (returns Server instance)
- **Class**: For standalone (manages lifecycle)
- Both patterns supported

### 3. Build Strategy
- **Bundle**: CLI entry point (single file)
- **Preserve**: Library exports (module structure)
- TypeScript declarations always generated

### 4. Client Pattern
- Wrapper class for external API
- Accept credentials in constructor
- Stateful connection management

### 5. Logging Pattern
- No-op for stdio transport
- File or stderr for other transports
- Never use console.log with stdio

### 6. Type Safety
- Strong typing throughout
- Separate type files or directories
- Export types for library consumers

### 7. Error Handling
- Serialize errors for MCP responses
- Proper error types (McpError)
- Graceful degradation

## Compatibility Checklist

When building a server compatible with `mcp-auth`:

- ✅ Export a factory function that accepts `(accessToken, userId, options?)`
- ✅ Factory returns a configured `Server` instance
- ✅ No shared state between server instances
- ✅ Client wrapper accepts credentials in constructor
- ✅ Tools are stateless (receive client as parameter)
- ✅ Proper TypeScript types exported
- ✅ ESM with `.js` extensions in imports
- ✅ Dual build: bundled CLI + preserved modules

## Migration Path

### From Standalone to Multi-Tenant

1. Extract server creation into factory function
2. Move credential loading from env to factory parameters
3. Ensure no shared state between instances
4. Add factory export to package.json
5. Update build to preserve module structure

### From Multi-Tenant to mcp-auth Integration

1. Keep existing factory function
2. Add mcp-auth wrapper in separate entry point
3. Configure auth provider and token resolver
4. Deploy wrapped server for remote access
5. Keep factory for direct usage

## Conclusion

This bootstrap pattern provides a foundation for building MCP servers that are:

- **Modular**: Clear separation of concerns
- **Type-safe**: Strong TypeScript typing
- **Multi-tenant ready**: Isolated instances per user
- **Library-friendly**: Dual build strategy
- **mcp-auth compatible**: Works with authentication framework

The pattern emphasizes **organization and structure** over specific implementations, allowing flexibility in choosing tools and technologies while maintaining consistency and compatibility.
