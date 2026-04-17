import { createRouter, RouterProvider } from "@tanstack/react-router";

// Import our routes
import { Route as rootRoute } from "./routes/__root";
import { Route as configStoresRoute } from "./routes/config-stores";
import { Route as configStoreDetailRoute } from "./routes/config-stores.$storeId";
import { Route as encryptedSecretsRoute } from "./routes/encrypted-secrets";
import { Route as encryptedSecretDetailRoute } from "./routes/encrypted-secrets.$uid";
import { Route as externalConfigsRoute } from "./routes/external-configs";
import { Route as externalConfigDetailRoute } from "./routes/external-configs.$configId";
import { Route as indexRoute } from "./routes/index";

// Create the route tree
const routeTree = rootRoute.addChildren([
	indexRoute,
	encryptedSecretsRoute,
	encryptedSecretDetailRoute,
	configStoresRoute,
	configStoreDetailRoute,
	externalConfigsRoute,
	externalConfigDetailRoute,
]);

// Create the router instance
const router = createRouter({ routeTree });

// Register the router instance for type safety
declare module "@tanstack/react-router" {
	interface Register {
		router: typeof router;
	}
}

export default function App() {
	return <RouterProvider router={router} />;
}
