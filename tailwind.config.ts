import type { Config } from "tailwindcss";

const config: Config = {
	content: ["./app/**/*.{js,ts,jsx,tsx,mdx}"],
	theme: {
		extend: {
			fontFamily: {
				display: ["var(--font-display)", "sans-serif"],
				mono: ["var(--font-mono)", "monospace"],
			},
		},
	},
	plugins: [],
};

export default config;
