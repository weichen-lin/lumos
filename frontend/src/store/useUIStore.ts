import { create } from "zustand";
import { persist } from "zustand/middleware";

export type ViewType = "overview" | "secretstores";
export type Theme = "light" | "dark" | "system";

interface UIState {
	currentNamespace: string;
	isSidebarOpen: boolean;
	activeMappingId: string | null;
	activeView: ViewType;
	theme: Theme;
	setNamespace: (ns: string) => void;
	toggleSidebar: () => void;
	setActiveMapping: (id: string | null) => void;
	setActiveView: (view: ViewType) => void;
	cycleTheme: () => void;
}

export const useUIStore = create<UIState>()(
	persist(
		(set) => ({
			currentNamespace: "default",
			isSidebarOpen: true,
			activeMappingId: null,
			activeView: "overview",
			theme: "dark",
			setNamespace: (ns) => set({ currentNamespace: ns }),
			toggleSidebar: () => set((s) => ({ isSidebarOpen: !s.isSidebarOpen })),
			setActiveMapping: (id) => set({ activeMappingId: id }),
			setActiveView: (view) => set({ activeView: view }),
			cycleTheme: () =>
				set((s) => {
					const order: Theme[] = ["dark", "light", "system"];
					const next = order[(order.indexOf(s.theme) + 1) % order.length];
					return { theme: next };
				}),
		}),
		{
			name: "lumos-ui",
			partialize: (state) => ({ theme: state.theme }),
		},
	),
);
