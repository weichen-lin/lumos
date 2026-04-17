import path from "node:path";
import tailwindcss from "@tailwindcss/vite";
import react from "@vitejs/plugin-react";
import { defineConfig } from "vite";

// https://vite.dev/config/
export default defineConfig({
	plugins: [react(), tailwindcss()],
	resolve: {
		alias: {
			"@": path.resolve(__dirname, "./src"),
		},
	},
	server: {
		proxy: {
			"/api": "http://localhost:8090",
		},
	},
	build: {
		rolldownOptions: {
			output: {
				codeSplitting: {
					groups: [
						{ name: "vendor-recharts", test: /node_modules[\\/]recharts/ },
						{ name: "vendor-motion", test: /node_modules[\\/]framer-motion/ },
						{ name: "vendor-tanstack", test: /node_modules[\\/]@tanstack/ },
						{ name: "vendor", test: /node_modules/ },
					],
				},
			},
		},
	},
});
