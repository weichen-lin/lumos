import { createRoute } from "@tanstack/react-router";
import { EncryptedSecretList } from "../components/EncryptedSecretList";
import { Route as rootRoute } from "./__root";

export const Route = createRoute({
	getParentRoute: () => rootRoute,
	path: "/encrypted-secrets",
	component: EncryptedSecretsPage,
});

function EncryptedSecretsPage() {
	return (
		<div className="flex flex-col gap-6 h-full min-h-0">
			<div className="space-y-0.5 shrink-0">
				<h2 className="text-xl font-black tracking-tight uppercase italic leading-none">
					EncryptedSecrets
				</h2>
				<p className="text-[10px] text-muted-foreground font-medium uppercase tracking-widest">
					SOPS Age-Encrypted Secrets from Git
				</p>
			</div>
			<div className="flex-1 min-h-0">
				<EncryptedSecretList />
			</div>
		</div>
	);
}
