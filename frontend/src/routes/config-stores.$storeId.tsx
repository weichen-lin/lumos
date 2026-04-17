import { createRoute } from "@tanstack/react-router";
import { ConfigStoreDetail } from "../components/ConfigStoreDetail";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/config-stores/$uid",
	component: ConfigStoreDetailPage,
});

function ConfigStoreDetailPage() {
	const { uid } = Route.useParams();
	return <ConfigStoreDetail storeId={uid} />;
}
