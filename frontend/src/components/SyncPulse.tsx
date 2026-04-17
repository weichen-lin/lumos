import { Badge } from "./ui/badge";

type Status = "Synced" | "Error" | "Syncing";

export const SyncPulse = ({ status }: { status: Status }) => {
	const getVariant = () => {
		if (status === "Synced") return "success";
		if (status === "Error") return "destructive";
		return "secondary";
	};

	return (
		<Badge
			variant={getVariant()}
			className="gap-2 px-2 py-0.5 uppercase tracking-wider text-[10px] font-bold"
		>
			<span
				className={`w-1.5 h-1.5 rounded-full ${status === "Syncing" ? "animate-pulse bg-primary" : "bg-current"}`}
			/>
			{status}
		</Badge>
	);
};
