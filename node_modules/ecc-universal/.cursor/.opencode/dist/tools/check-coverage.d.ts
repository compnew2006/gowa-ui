/**
 * Check Coverage Tool
 *
 * Custom OpenCode tool to analyze test coverage and report on gaps.
 * Supports common coverage report formats.
 */
declare const _default: {
    description: string;
    args: {
        threshold: import("zod").ZodOptional<import("zod").ZodNumber>;
        showUncovered: import("zod").ZodOptional<import("zod").ZodBoolean>;
        format: import("zod").ZodOptional<import("zod").ZodEnum<{
            summary: "summary";
            detailed: "detailed";
            json: "json";
        }>>;
    };
    execute(args: {
        threshold?: number | undefined;
        showUncovered?: boolean | undefined;
        format?: "summary" | "detailed" | "json" | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=check-coverage.d.ts.map