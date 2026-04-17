import { useQuery } from "@tanstack/react-query";

export interface DashboardStats {
	summary: {
		synced: number;
		error: number;
		stale: number;
		expiring: number;
	};
	providers: Array<{
		name: string;
		value: number;
		color: string;
	}>;
	recentEvents: Array<{
		id: string;
		status: string;
		time: string | null;
		target: string;
		message: string;
		commitSha?: string;
	}>;
	namespaceHealth: Array<{
		name: string;
		synced: number;
		error: number;
		stale: number;
		coverage: number;
	}>;
}

const fetchDashboardStats = async (): Promise<DashboardStats> => {
	const res = await fetch("/api/dashboard/config-stats");
	if (!res.ok)
		throw new Error(`Failed to fetch dashboard stats: ${res.status}`);
	return res.json();
};

export function useDashboardData() {
	return useQuery({
		queryKey: ["dashboard-stats"],
		queryFn: fetchDashboardStats,
		refetchInterval: 30000,
	});
}
