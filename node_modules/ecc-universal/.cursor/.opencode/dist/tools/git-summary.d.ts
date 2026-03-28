/**
 * ECC Custom Tool: Git Summary
 *
 * Returns branch/status/log/diff details for the active repository.
 */
declare const _default: {
    description: string;
    args: {
        depth: import("zod").ZodOptional<import("zod").ZodNumber>;
        includeDiff: import("zod").ZodOptional<import("zod").ZodBoolean>;
        baseBranch: import("zod").ZodOptional<import("zod").ZodString>;
    };
    execute(args: {
        depth?: number | undefined;
        includeDiff?: boolean | undefined;
        baseBranch?: string | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=git-summary.d.ts.map