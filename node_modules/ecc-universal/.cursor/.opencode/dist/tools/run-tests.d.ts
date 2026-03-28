/**
 * Run Tests Tool
 *
 * Custom OpenCode tool to run test suites with various options.
 * Automatically detects the package manager and test framework.
 */
declare const _default: {
    description: string;
    args: {
        pattern: import("zod").ZodOptional<import("zod").ZodString>;
        coverage: import("zod").ZodOptional<import("zod").ZodBoolean>;
        watch: import("zod").ZodOptional<import("zod").ZodBoolean>;
        updateSnapshots: import("zod").ZodOptional<import("zod").ZodBoolean>;
    };
    execute(args: {
        pattern?: string | undefined;
        coverage?: boolean | undefined;
        watch?: boolean | undefined;
        updateSnapshots?: boolean | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=run-tests.d.ts.map