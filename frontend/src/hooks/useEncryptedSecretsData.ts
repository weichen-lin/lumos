import { useQuery } from "@tanstack/react-query";

export interface EncryptedSecret {
	uid: string;
	name: string;
	namespace: string;
	store: string;
	ageKeyRef: string;
	targetSecret: string;
	status: "Synced" | "Error" | "Stale";
	message?: string;
	lastSync: string | null;
	commitSha?: string;
	sources: string[];
	refreshInterval: string;
}

const fetchEncryptedSecrets = async (): Promise<EncryptedSecret[]> => {
	const res = await fetch("/api/encrypted-secrets");
	if (!res.ok) throw new Error(`Failed: ${res.status}`);
	return res.json();
};

export function useEncryptedSecretsData() {
	return useQuery({
		queryKey: ["encrypted-secrets"],
		queryFn: fetchEncryptedSecrets,
		refetchInterval: 15000,
	});
}
