import { useNavigate } from "@tanstack/react-router";
import { motion } from "framer-motion";
import {
	AlertCircle,
	AlertTriangle,
	CheckCircle2,
	Clock,
	FileSliders,
	GitCommit,
	Search,
} from "lucide-react";
import { useState } from "react";
import {
	type ExternalConfig,
	useExternalConfigsData,
} from "../hooks/useExternalConfigsData";
import { formatAgo } from "../lib/time";
import { Badge } from "./ui/badge";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";

export function ExternalConfigList() {
	const { data, isLoading } = useExternalConfigsData();
	const navigate = useNavigate();
	const [query, setQuery] = useState("");

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Loading Config Tasks...
				</div>
			</div>
		);
	}

	const filtered = data.filter((c) =>
		[c.name, c.namespace, c.configStore]
			.join(" ")
			.toLowerCase()
			.includes(query.toLowerCase()),
	);

	return (
		<motion.div
			initial={{ opacity: 0, x: 20 }}
			animate={{ opacity: 1, x: 0 }}
			className="h-full flex flex-col gap-6 overflow-hidden"
		>
			{/* Search & Filter Bar */}
			<div className="flex items-center gap-4 shrink-0">
				<div className="relative flex-1 max-w-md">
					<Search
						className="absolute left-3 top-1/2 -translate-y-1/2 text-muted-foreground"
						size={14}
					/>
					<Input
						placeholder="Search by name, store or namespace..."
						className="pl-9 h-10 bg-card/30 border-border text-sm focus-visible:ring-primary/30"
						value={query}
						onChange={(e) => setQuery(e.target.value)}
					/>
				</div>
			</div>

			{/* Table */}
			<Card className="flex-1 border-border bg-card/30 flex flex-col min-h-0 overflow-hidden">
				<CardContent className="flex-1 overflow-y-auto p-0 custom-scrollbar">
					{data.length === 0 ? (
						<div className="flex flex-col items-center justify-center h-full py-20 gap-4">
							<div className="p-4 rounded-full bg-muted/30">
								<FileSliders size={24} className="text-muted-foreground/30" />
							</div>
							<div className="flex flex-col items-center gap-1 text-center">
								<p className="text-sm font-medium text-muted-foreground/60">
									No external configs found
								</p>
								<p className="text-xs text-muted-foreground/40">
									ExternalConfig resources will appear here once created
								</p>
							</div>
						</div>
					) : (
						<table className="w-full text-left text-sm">
							<thead className="sticky top-0 bg-background/95 backdrop-blur-md z-10 border-b border-border">
								<tr className="bg-muted/30">
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Name
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Namespace
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										ConfigStore
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Status
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Last Sync
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Commit
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{filtered.map((config) => (
									<tr
										key={config.id}
										onClick={() =>
											navigate({
												to: "/external-configs/$uid",
												params: { uid: config.uid },
											})
										}
										className="hover:bg-accent/30 transition-colors group cursor-pointer odd:bg-muted/10"
									>
										<td className="px-6 py-5">
											<span className="font-bold text-base tracking-tight">
												{config.name}
											</span>
										</td>
										<td className="px-6 py-5">
											<span className="text-xs font-mono font-bold px-2 py-1 bg-accent/50 rounded border border-border/70">
												{config.namespace}
											</span>
										</td>
										<td className="px-6 py-5">
											<div className="flex flex-col gap-1">
												<span className="font-bold text-xs">
													{config.configStore}
												</span>
												<span className="text-[9px] font-mono text-muted-foreground uppercase">
													{config.storeType}
												</span>
											</div>
										</td>
										<td className="px-6 py-5">
											<StatusBadge
												status={config.status}
												message={config.message}
											/>
										</td>
										<td className="px-6 py-5">
											<div className="flex items-center gap-2 text-muted-foreground">
												<Clock size={12} />
												<span className="text-xs font-mono">
													{formatAgo(config.lastSync)}
												</span>
											</div>
										</td>
										<td className="px-6 py-5">
											<CommitCell sha={config.commitSha} />
										</td>
									</tr>
								))}
							</tbody>
						</table>
					)}
				</CardContent>
			</Card>
		</motion.div>
	);
}

function CommitCell({ sha }: { sha?: string }) {
	if (!sha)
		return (
			<span className="text-[10px] text-muted-foreground/40 font-mono italic">
				—
			</span>
		);

	return (
		<div className="inline-flex items-center gap-1.5 font-mono text-[10px] bg-accent/30 px-2 py-0.5 rounded-sm border border-border/60 group-hover:border-primary/30 transition-colors">
			<GitCommit size={10} className="text-primary/60" />
			<span className="text-primary/80 font-bold">{sha}</span>
		</div>
	);
}

function StatusBadge({
	status,
	message,
}: {
	status: ExternalConfig["status"];
	message?: string;
}) {
	if (status === "Synced") {
		return (
			<Badge className="bg-emerald-500/10 text-emerald-500 border-emerald-500/20 hover:bg-emerald-500/20 gap-1.5 px-2 py-1 w-fit">
				<CheckCircle2 size={12} />
				<span className="text-[10px] font-bold uppercase tracking-widest leading-none">
					Synced
				</span>
			</Badge>
		);
	}
	if (status === "Error") {
		return (
			<div className="flex flex-col items-start gap-1.5">
				<Badge className="bg-red-500/15 text-red-500 border-red-500/30 hover:bg-red-500/20 gap-1.5 px-2 py-1 shadow-[0_0_15px_-5px_rgba(239,68,68,0.4)] w-fit">
					<AlertCircle size={12} />
					<span className="text-[10px] font-bold uppercase tracking-widest leading-none">
						Error
					</span>
				</Badge>
				{message && (
					<span className="text-[10px] text-red-400/80 font-medium italic px-1 truncate max-w-[150px]">
						{message}
					</span>
				)}
			</div>
		);
	}
	return (
		<div className="flex flex-col items-start gap-1.5">
			<Badge className="bg-amber-500/10 text-amber-500 border-amber-500/20 gap-1.5 px-2 py-1 w-fit">
				<AlertTriangle size={12} />
				<span className="text-[10px] font-bold uppercase tracking-widest leading-none">
					Stale
				</span>
			</Badge>
			{message && (
				<span className="text-[10px] text-amber-400/80 font-medium italic px-1 truncate max-w-[150px]">
					{message}
				</span>
			)}
		</div>
	);
}
