/**
 * Everything Claude Code (ECC) Plugin Hooks for OpenCode
 *
 * This plugin translates Claude Code hooks to OpenCode's plugin system.
 * OpenCode's plugin system is MORE sophisticated than Claude Code with 20+ events
 * compared to Claude Code's 3 phases (PreToolUse, PostToolUse, Stop).
 *
 * Hook Event Mapping:
 * - PreToolUse → tool.execute.before
 * - PostToolUse → tool.execute.after
 * - Stop → session.idle / session.status
 * - SessionStart → session.created
 * - SessionEnd → session.deleted
 */
import type { PluginInput } from "@opencode-ai/plugin";
export declare const ECCHooksPlugin: ({ client, $, directory, worktree, }: PluginInput) => Promise<{
    /**
     * Prettier Auto-Format Hook
     * Equivalent to Claude Code PostToolUse hook for prettier
     *
     * Triggers: After any JS/TS/JSX/TSX file is edited
     * Action: Runs prettier --write on the file
     */
    "file.edited": (event: {
        path: string;
    }) => Promise<void>;
    /**
     * TypeScript Check Hook
     * Equivalent to Claude Code PostToolUse hook for tsc
     *
     * Triggers: After edit tool completes on .ts/.tsx files
     * Action: Runs tsc --noEmit to check for type errors
     */
    "tool.execute.after": (input: {
        tool: string;
        args?: {
            filePath?: string;
        };
    }, output: unknown) => Promise<void>;
    /**
     * Pre-Tool Security Check
     * Equivalent to Claude Code PreToolUse hook
     *
     * Triggers: Before tool execution
     * Action: Warns about potential security issues
     */
    "tool.execute.before": (input: {
        tool: string;
        args?: Record<string, unknown>;
    }) => Promise<void>;
    /**
     * Session Created Hook
     * Equivalent to Claude Code SessionStart hook
     *
     * Triggers: When a new session starts
     * Action: Loads context and displays welcome message
     */
    "session.created": () => Promise<void>;
    /**
     * Session Idle Hook
     * Equivalent to Claude Code Stop hook
     *
     * Triggers: When session becomes idle (task completed)
     * Action: Runs console.log audit on all edited files
     */
    "session.idle": () => Promise<void>;
    /**
     * Session Deleted Hook
     * Equivalent to Claude Code SessionEnd hook
     *
     * Triggers: When session ends
     * Action: Final cleanup and state saving
     */
    "session.deleted": () => Promise<void>;
    /**
     * File Watcher Hook
     * OpenCode-only feature
     *
     * Triggers: When file system changes are detected
     * Action: Updates tracking
     */
    "file.watcher.updated": (event: {
        path: string;
        type: string;
    }) => Promise<void>;
    /**
     * Todo Updated Hook
     * OpenCode-only feature
     *
     * Triggers: When todo list is updated
     * Action: Logs progress
     */
    "todo.updated": (event: {
        todos: Array<{
            text: string;
            done: boolean;
        }>;
    }) => Promise<void>;
    /**
     * Shell Environment Hook
     * OpenCode-specific: Inject environment variables into shell commands
     *
     * Triggers: Before shell command execution
     * Action: Sets PROJECT_ROOT, PACKAGE_MANAGER, DETECTED_LANGUAGES, ECC_VERSION
     */
    "shell.env": () => Promise<Record<string, string>>;
    /**
     * Session Compacting Hook
     * OpenCode-specific: Control context compaction behavior
     *
     * Triggers: Before context compaction
     * Action: Push ECC context block and custom compaction prompt
     */
    "experimental.session.compacting": () => Promise<{
        context: string;
        compaction_prompt: string;
    }>;
    /**
     * Permission Auto-Approve Hook
     * OpenCode-specific: Auto-approve safe operations
     *
     * Triggers: When permission is requested
     * Action: Auto-approve reads, formatters, and test commands; log all for audit
     */
    "permission.ask": (event: {
        tool: string;
        args: unknown;
    }) => Promise<{
        approved: boolean;
        reason: string;
    } | {
        approved: undefined;
        reason?: undefined;
    }>;
}>;
export default ECCHooksPlugin;
//# sourceMappingURL=ecc-hooks.d.ts.map