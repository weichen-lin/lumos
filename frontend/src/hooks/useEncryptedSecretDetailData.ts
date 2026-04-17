import { useQuery } from "@tanstack/react-query";

export interface EncryptedSecretDetail {
	uid: string;
	name: string;
	namespace: string;
	store: string;
	ageKeyRef: string;
	targetSecret: string;
	status: "Synced" | "Error" | "Stale";
	message?: string;
	lastSync: string | null;
	nextSync: string | null;
	commitSha?: string;
	sources: string[];
	refreshInterval: string;
	events: Array<{
		id: string;
		type: "Normal" | "Warning";
		reason: string;
		message: string;
		time: string | null;
	}>;
}

const fetchEncryptedSecretDetail = async (
	uid: string,
): Promise<EncryptedSecretDetail> => {
	const res = await fetch(`/api/encrypted-secrets/${uid}`);
	if (!res.ok) throw new Error(`Failed: ${res.status}`);
	return res.json();
};

export function useEncryptedSecretDetailData(uid: string) {
	return useQuery({
		queryKey: ["encrypted-secrets", uid],
		queryFn: () => fetchEncryptedSecretDetail(uid),
		refetchInterval: 15000,
	});
}
