import { useQuery } from "@tanstack/react-query";

export interface ConfigDetail {
	id: string;
	name: string;
	namespace: string;
	status: "Synced" | "Error" | "Stale";
	message?: string;
	lastSync: string | null;
	nextSync: string | null;
	refreshInterval: string;
	store: {
		name: string;
		type: "ConfigStore" | "ClusterConfigStore";
		provider: "Git" | "Unknown";
	};
	data: Array<{
		source: string;
		key: string;
		value: string;
		format: string;
		commitSha?: string;
		lastChanged: string | null;
	}>;
	events: Array<{
		id: string;
		type: "Normal" | "Warning";
		reason: string;
		message: string;
		time: string;
	}>;
	labels: Record<string, string>;
}

const fetchConfigDetail = async (uid: string): Promise<ConfigDetail> => {
	const res = await fetch(`/api/external-configs/${uid}`);
	if (!res.ok) throw new Error(`Failed to fetch config detail: ${res.status}`);
	return res.json();
};

export function useExternalConfigDetailData(uid: string) {
	return useQuery({
		queryKey: ["external-config-detail", uid],
		queryFn: () => fetchConfigDetail(uid),
		refetchInterval: 15000,
		enabled: !!uid,
	});
}
