import { createRoute } from "@tanstack/react-router";
import { EncryptedSecretDetail } from "../components/EncryptedSecretDetail";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/encrypted-secrets/$uid",
	component: EncryptedSecretDetailPage,
});

function EncryptedSecretDetailPage() {
	const { uid } = Route.useParams();
	return <EncryptedSecretDetail uid={uid} />;
}
