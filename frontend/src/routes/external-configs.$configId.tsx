import { createRoute } from "@tanstack/react-router";
import { ExternalConfigDetail } from "../components/ExternalConfigDetail";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/external-configs/$uid",
	component: ExternalConfigDetailPage,
});

function ExternalConfigDetailPage() {
	const { uid } = Route.useParams();
	return <ExternalConfigDetail configId={uid} />;
}
