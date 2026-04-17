import { useQuery } from "@tanstack/react-query";

export interface GitInfo {
	url: string;
	branch: string;
}

export interface ConsulInfo {
	address: string;
	prefix: string;
}

export interface EcRef {
	uid: string;
	id: string;
	name: string;
	namespace: string;
	status: "Synced" | "Error" | "Stale";
	lastSync: string | null;
}

export interface ConfigStoreDetail {
	id: string;
	name: string;
	type: "ConfigStore" | "ClusterConfigStore";
	provider: "Git" | "Consul";
	namespace?: string;
	git?: GitInfo;
	consul?: ConsulInfo;
	externalConfigs: EcRef[];
}

const fetchConfigStoreDetail = async (
	uid: string,
): Promise<ConfigStoreDetail> => {
	const res = await fetch(`/api/config-stores/${uid}`);
	if (!res.ok)
		throw new Error(`Failed to fetch config store detail: ${res.status}`);
	return res.json();
};

export function useConfigStoreDetailData(uid: string) {
	return useQuery({
		queryKey: ["config-store-detail", uid],
		queryFn: () => fetchConfigStoreDetail(uid),
		refetchInterval: 15000,
	});
}
