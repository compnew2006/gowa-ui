/**
 * Security Audit Tool
 *
 * Custom OpenCode tool to run security audits on dependencies and code.
 * Combines npm audit, secret scanning, and OWASP checks.
 *
 * NOTE: This tool SCANS for security anti-patterns - it does not introduce them.
 * The regex patterns below are used to DETECT potential issues in user code.
 */
declare const _default: {
    description: string;
    args: {
        type: import("zod").ZodOptional<import("zod").ZodEnum<{
            all: "all";
            dependencies: "dependencies";
            secrets: "secrets";
            code: "code";
        }>>;
        fix: import("zod").ZodOptional<import("zod").ZodBoolean>;
        severity: import("zod").ZodOptional<import("zod").ZodEnum<{
            low: "low";
            moderate: "moderate";
            high: "high";
            critical: "critical";
        }>>;
    };
    execute(args: {
        type?: "all" | "dependencies" | "secrets" | "code" | undefined;
        fix?: boolean | undefined;
        severity?: "low" | "moderate" | "high" | "critical" | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=security-audit.d.ts.map