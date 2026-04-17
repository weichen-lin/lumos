import { createRoute } from "@tanstack/react-router";
import { Dashboard } from "../components/Dashboard";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/",
	component: OverviewPage,
});

function OverviewPage() {
	return (
		<div className="flex flex-col gap-4 h-full min-h-0">
			<div className="space-y-0.5 shrink-0">
				<h2 className="text-xl font-black tracking-tight uppercase italic leading-none">
					Global Overview
				</h2>
				<p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
					Cluster Health & Resource Distribution
				</p>
			</div>

			<div className="flex-1 min-h-0 overflow-hidden">
				<Dashboard />
			</div>
		</div>
	);
}
