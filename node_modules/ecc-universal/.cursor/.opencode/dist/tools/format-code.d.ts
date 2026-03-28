/**
 * ECC Custom Tool: Format Code
 *
 * Returns the formatter command that should be run for a given file.
 * This avoids shell execution assumptions while still giving precise guidance.
 */
declare const _default: {
    description: string;
    args: {
        filePath: import("zod").ZodString;
        formatter: import("zod").ZodOptional<import("zod").ZodEnum<{
            biome: "biome";
            prettier: "prettier";
            black: "black";
            gofmt: "gofmt";
            rustfmt: "rustfmt";
        }>>;
    };
    execute(args: {
        filePath: string;
        formatter?: "biome" | "prettier" | "black" | "gofmt" | "rustfmt" | undefined;
    }, context: import("@opencode-ai/plugin/tool").ToolContext): Promise<string>;
};
export default _default;
//# sourceMappingURL=format-code.d.ts.map