import { useQuery } from "@tanstack/react-query";

export interface ExternalConfig {
	uid: string;
	id: string;
	name: string;
	namespace: string;
	configStore: string;
	storeType: "ConfigStore" | "ClusterConfigStore";
	status: "Synced" | "Error" | "Stale";
	message?: string;
	lastSync: string | null;
	commitSha?: string;
}

const fetchExternalConfigs = async (): Promise<ExternalConfig[]> => {
	const res = await fetch("/api/external-configs");
	if (!res.ok)
		throw new Error(`Failed to fetch external configs: ${res.status}`);
	return res.json();
};

export function useExternalConfigsData() {
	return useQuery({
		queryKey: ["external-configs"],
		queryFn: fetchExternalConfigs,
		refetchInterval: 15000,
	});
}
