export function formatAgo(iso: string | null | undefined): string {
	if (!iso) return "never";
	const d = (Date.now() - new Date(iso).getTime()) / 1000;
	if (d < 60) return "just now";
	if (d < 3600) return `${Math.floor(d / 60)}m ago`;
	if (d < 86400) return `${Math.floor(d / 3600)}h ago`;
	return `${Math.floor(d / 86400)}d ago`;
}

export function formatUntil(iso: string | null | undefined): string {
	if (!iso) return "pending";
	const d = (new Date(iso).getTime() - Date.now()) / 1000;
	if (d <= 0) return "syncing soon";
	if (d < 60) return `in ${Math.floor(d)}s`;
	if (d < 3600) return `in ${Math.floor(d / 60)}m`;
	return `in ${Math.floor(d / 3600)}h`;
}
