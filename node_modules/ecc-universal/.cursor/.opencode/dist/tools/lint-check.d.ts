/**
 * ECC Custom Tool: Lint Check
 *
 * Detects the appropriate linter and returns a runnable lint command.
 */
declare const _default: {
    description: string;
    args: {
        target: import("zod").ZodOptional<import("zod").ZodString>;
        fix: import("zod").ZodOptional<import("zod").ZodBoolean>;
        linter: import("zod").ZodOptional<import("zod").ZodEnum<{
            biome: "biome";
            eslint: "eslint";
            ruff: "ruff";
            pylint: "pylint";
            "golangci-lint": "golangci-lint";
        }>>;
    };
    execute(args: {
        target?: string | undefined;
        fix?: boolean | undefined;
        linter?: "biome" | "eslint" | "ruff" | "pylint" | "golangci-lint" | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=lint-check.d.ts.map