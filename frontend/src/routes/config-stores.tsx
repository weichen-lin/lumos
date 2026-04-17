import { createRoute } from "@tanstack/react-router";
import { ConfigStoreList } from "../components/ConfigStoreList";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/config-stores",
	component: ConfigStoresPage,
});

function ConfigStoresPage() {
	return (
		<div className="flex flex-col gap-6 h-full min-h-0">
			<div className="space-y-0.5 shrink-0">
				<h2 className="text-xl font-black tracking-tight uppercase italic leading-none">
					ConfigStores
				</h2>
				<p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
					Config Source Infrastructure
				</p>
			</div>
			<div className="flex-1 min-h-0">
				<ConfigStoreList />
			</div>
		</div>
	);
}
