import { useQuery } from "@tanstack/react-query";

export interface ConfigStore {
	uid: string;
	id: string;
	name: string;
	type: "ConfigStore" | "ClusterConfigStore";
	provider: "Git" | "Consul";
	namespace?: string;
	usageCount: number;
}

const fetchConfigStores = async (): Promise<ConfigStore[]> => {
	const res = await fetch("/api/config-stores");
	if (!res.ok) throw new Error(`Failed to fetch config stores: ${res.status}`);
	return res.json();
};

export function useConfigStoreData() {
	return useQuery({
		queryKey: ["config-stores"],
		queryFn: fetchConfigStores,
		refetchInterval: 15000,
	});
}
