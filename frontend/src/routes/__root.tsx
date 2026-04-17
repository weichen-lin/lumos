import { createRootRoute, Link, Outlet } from "@tanstack/react-router";
import {
	LayoutDashboard as IconDashboard,
	FileCode as IconFileCode,
	LockKeyhole as IconLock,
	Monitor as IconMonitor,
	Moon as IconMoon,
	Server as IconServer,
	Sun as IconSun,
} from "lucide-react";
import type { ReactNode } from "react";
import { useEffect } from "react";
import { SyncPulse as UISyncPulse } from "../components/SyncPulse";
import {
	Avatar as UIAvatar,
	AvatarFallback as UIAvatarFallback,
	AvatarImage as UIAvatarImage,
} from "../components/ui/avatar";
import { Separator as UISeparator } from "../components/ui/separator";
import { useDashboardData } from "../hooks/useDashboardData";
import { useUIStore } from "../store/useUIStore";

export const Route = createRootRoute({
	component: RootComponent,
});

const THEME_ICON: Record<string, ReactNode> = {
	dark: <IconMoon size={14} />,
	light: <IconSun size={14} />,
	system: <IconMonitor size={14} />,
};

const THEME_LABEL = { dark: "Dark", light: "Light", system: "System" } as const;

function RootComponent() {
	const { theme, cycleTheme } = useUIStore();
	const { data: stats } = useDashboardData();

	const syncStatus = stats
		? stats.summary.error > 0
			? "Error"
			: stats.summary.stale > 0
				? "Syncing"
				: "Synced"
		: null;

	useEffect(() => {
		const root = document.documentElement;
		const isDark =
			theme === "dark" ||
			(theme === "system" &&
				window.matchMedia("(prefers-color-scheme: dark)").matches);
		root.classList.toggle("dark", isDark);
	}, [theme]);

	// Keep system theme in sync when OS preference changes
	useEffect(() => {
		if (theme !== "system") return;
		const mq = window.matchMedia("(prefers-color-scheme: dark)");
		const handler = (e: MediaQueryListEvent) =>
			document.documentElement.classList.toggle("dark", e.matches);
		mq.addEventListener("change", handler);
		return () => mq.removeEventListener("change", handler);
	}, [theme]);

	return (
		<div className="h-screen bg-background text-foreground flex flex-col md:flex-row font-sans overflow-hidden">
			{/* Sidebar Navigation */}
			<aside className="w-full md:w-64 h-full border-r border-border bg-card flex flex-col p-4 gap-6 shrink-0">
				<div className="flex items-center gap-3 px-2 text-primary">
					<img src="/favicon.svg" alt="Lumos" className="w-8 h-8" />
					<span className="font-bold tracking-tight uppercase text-foreground">
						Lumos
					</span>
				</div>

				<nav className="flex flex-col gap-1 px-1 overflow-y-auto">
					<NavLink
						to="/"
						icon={<IconDashboard size={16} />}
						label="Overview"
						exact
					/>

					<NavSection label="Secrets" />
					<NavLink
						to="/encrypted-secrets"
						icon={<IconLock size={16} />}
						label="EncryptedSecrets"
					/>

					<NavSection label="Config" />
					<NavLink
						to="/config-stores"
						icon={<IconServer size={16} />}
						label="ConfigStores"
					/>
					<NavLink
						to="/external-configs"
						icon={<IconFileCode size={16} />}
						label="ExternalConfigs"
					/>
				</nav>
			</aside>

			{/* Main Content Area */}
			<main className="flex-1 flex flex-col h-full min-w-0 overflow-hidden">
				{/* Header */}
				<header className="h-14 border-b border-border flex items-center justify-end px-8 bg-card/50 backdrop-blur-md shrink-0 z-20">
					<div className="flex items-center gap-6">
						{syncStatus && <UISyncPulse status={syncStatus} />}
						{syncStatus && (
							<UISeparator orientation="vertical" className="h-4" />
						)}
						<button
							type="button"
							onClick={cycleTheme}
							className="flex items-center justify-center w-7 h-7 rounded-md cursor-pointer text-muted-foreground bg-muted/40 border border-border/70 hover:bg-accent hover:text-foreground hover:border-border/60 active:scale-95 transition-all outline-none shrink-0"
							title={`Theme: ${THEME_LABEL[theme]}`}
						>
							{THEME_ICON[theme]}
						</button>
						<UISeparator orientation="vertical" className="h-4" />
						<UIAvatar className="w-7 h-7 border border-border">
							<UIAvatarImage src="https://api.dicebear.com/7.x/avataaars/svg?seed=Lumos" />
							<UIAvatarFallback>LM</UIAvatarFallback>
						</UIAvatar>
					</div>
				</header>

				{/* Dynamic Content */}
				<div className="flex-1 overflow-hidden p-6 max-w-full mx-auto w-full flex flex-col gap-6">
					<Outlet />
				</div>
			</main>
		</div>
	);
}

function NavSection({ label }: { label: string }) {
	return (
		<div className="px-3 pt-4 pb-1">
			<span className="text-[9px] font-bold uppercase tracking-widest text-muted-foreground/50">
				{label}
			</span>
		</div>
	);
}

function NavLink({
	to,
	icon,
	label,
	exact = false,
}: {
	to: string;
	icon: ReactNode;
	label: string;
	exact?: boolean;
}) {
	return (
		<Link
			to={to}
			activeProps={{ className: "bg-accent text-accent-foreground shadow-sm" }}
			activeOptions={exact ? { exact: true } : undefined}
			className="w-full flex items-center gap-3 px-3 py-2 rounded-md transition-all text-xs font-semibold text-muted-foreground hover:bg-accent/50 hover:text-accent-foreground"
		>
			{icon}
			<span>{label}</span>
		</Link>
	);
}
