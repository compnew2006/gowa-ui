/** @type {import('tailwindcss').Config} */
const defaultTheme = require("tailwindcss/defaultTheme");
const colorVar = (name) => `rgb(var(${name}) / <alpha-value>)`;

module.exports = {
  darkMode: ["class"],
  // Add light mode variant for dark-first design
  plugins: [
    require("tailwindcss-animate"),
    function ({ addVariant }) {
      addVariant("light", ".light &");
    },
  ],
  content: ["./index.html", "./src/**/*.{vue,js,ts,jsx,tsx}"],
  theme: {
    container: {
      center: true,
      padding: "2rem",
      screens: {
        "2xl": "1400px",
      },
    },
    extend: {
      fontFamily: {
        sans: ["var(--font-sans)", ...defaultTheme.fontFamily.sans],
        serif: ["var(--font-serif)", ...defaultTheme.fontFamily.serif],
        mono: ["var(--font-mono)", ...defaultTheme.fontFamily.mono],
      },
      colors: {
        border: colorVar("--border"),
        input: colorVar("--input"),
        ring: colorVar("--ring"),
        background: colorVar("--background"),
        foreground: colorVar("--foreground"),
        primary: {
          DEFAULT: colorVar("--primary"),
          foreground: colorVar("--primary-foreground"),
        },
        secondary: {
          DEFAULT: colorVar("--secondary"),
          foreground: colorVar("--secondary-foreground"),
        },
        destructive: {
          DEFAULT: colorVar("--destructive"),
          foreground: colorVar("--destructive-foreground"),
        },
        muted: {
          DEFAULT: colorVar("--muted"),
          foreground: colorVar("--muted-foreground"),
        },
        accent: {
          DEFAULT: colorVar("--accent"),
          foreground: colorVar("--accent-foreground"),
        },
        popover: {
          DEFAULT: colorVar("--popover"),
          foreground: colorVar("--popover-foreground"),
        },
        card: {
          DEFAULT: colorVar("--card"),
          foreground: colorVar("--card-foreground"),
        },
        sidebar: {
          DEFAULT: colorVar("--sidebar"),
          foreground: colorVar("--sidebar-foreground"),
          primary: colorVar("--sidebar-primary"),
          "primary-foreground": colorVar("--sidebar-primary-foreground"),
          accent: colorVar("--sidebar-accent"),
          "accent-foreground": colorVar("--sidebar-accent-foreground"),
          border: colorVar("--sidebar-border"),
          ring: colorVar("--sidebar-ring"),
        },
        chart: {
          1: colorVar("--chart-1"),
          2: colorVar("--chart-2"),
          3: colorVar("--chart-3"),
          4: colorVar("--chart-4"),
          5: colorVar("--chart-5"),
        },
        whatsapp: {
          green: "#25D366",
          teal: "#128C7E",
          "teal-dark": "#075E54",
          light: "#DCF8C6",
        },
        // WhatsApp Web panel surface tokens (theme-independent)
        "wa-panel": {
          canvas: colorVar("--wa-panel-canvas"),
          surface: colorVar("--wa-panel-surface"),
          header: colorVar("--wa-panel-header"),
          toolbar: colorVar("--wa-panel-toolbar"),
          input: colorVar("--wa-panel-input"),
          sheet: colorVar("--wa-panel-sheet"),
          hover: colorVar("--wa-panel-hover"),
          "bubble-in": colorVar("--wa-panel-bubble-in"),
          "bubble-out": colorVar("--wa-panel-bubble-out"),
          tint: colorVar("--wa-panel-tint"),
          accent: "#00a884",
          link: "#53bdeb",
        },
        // Violet color palette (primary accent)
        violet: {
          50: "#f5f3ff",
          100: "#ede9fe",
          200: "#ddd6fe",
          300: "#c4b5fd",
          400: "#a78bfa",
          500: "#8b5cf6",
          600: "#7c3aed",
          700: "#6d28d9",
          800: "#5b21b6",
          900: "#4c1d95",
          950: "#2e1065",
        },
        // Glass effect colors
        glass: {
          bg: "var(--glass-bg)",
          "bg-hover": "var(--glass-bg-hover)",
          border: "var(--glass-border)",
        },
      },
      borderRadius: {
        "2xl": "calc(var(--radius) + 0.4rem)",
        lg: "var(--radius)",
        md: "calc(var(--radius) - 2px)",
        sm: "calc(var(--radius) - 4px)",
      },
      boxShadow: {
        xs: "var(--shadow-xs)",
        sm: "var(--shadow-sm)",
        DEFAULT: "var(--shadow)",
        md: "var(--shadow-md)",
        lg: "var(--shadow-lg)",
        xl: "var(--shadow-xl)",
        "2xl": "var(--shadow-2xl)",
      },
      letterSpacing: {
        tighter: "var(--tracking-tighter)",
        tight: "var(--tracking-tight)",
        normal: "var(--tracking-normal)",
        wide: "var(--tracking-wide)",
        wider: "var(--tracking-wider)",
        widest: "var(--tracking-widest)",
      },
      keyframes: {
        "accordion-down": {
          from: {
            height: "0",
          },
          to: {
            height: "var(--reka-accordion-content-height)",
          },
        },
        "accordion-up": {
          from: {
            height: "var(--reka-accordion-content-height)",
          },
          to: {
            height: "0",
          },
        },
      },
      animation: {
        "accordion-down": "accordion-down 0.2s ease-out",
        "accordion-up": "accordion-up 0.2s ease-out",
      },
      // Glass effect backdrop blur
      backdropBlur: {
        xs: "2px",
      },
    },
  },
};
