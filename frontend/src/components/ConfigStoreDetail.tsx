import { Link, useNavigate } from "@tanstack/react-router";
import { motion } from "framer-motion";
import {
	AlertCircle,
	AlertTriangle,
	CheckCircle2,
	ChevronLeft,
	Cloud,
	GitBranch,
	Server,
} from "lucide-react";
import { useConfigStoreDetailData } from "../hooks/useConfigStoreDetailData";
import { formatAgo } from "../lib/time";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Separator } from "./ui/separator";

export function ConfigStoreDetail({ storeId }: { storeId: string }) {
	const { data, isLoading } = useConfigStoreDetailData(storeId);
	const navigate = useNavigate();

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Loading Store...
				</div>
			</div>
		);
	}

	return (
		<motion.div
			initial={{ opacity: 0, y: 10 }}
			animate={{ opacity: 1, y: 0 }}
			className="h-full flex flex-col gap-6 overflow-hidden"
		>
			{/* Header */}
			<div className="flex items-start gap-4 shrink-0">
				<Link
					to="/config-stores"
					className="mt-1 p-1.5 hover:bg-accent rounded-md transition-colors text-muted-foreground hover:text-foreground"
				>
					<ChevronLeft size={18} />
				</Link>
				<div className="space-y-2">
					<div className="flex items-center gap-3 flex-wrap">
						<h2 className="text-2xl font-black tracking-tighter uppercase italic">
							{data.name}
						</h2>
						<Badge
							variant="secondary"
							className={`text-[9px] h-5 px-2 font-bold uppercase tracking-widest ${
								data.type === "ClusterConfigStore"
									? "bg-primary/10 text-primary border-primary/20"
									: "bg-muted/50 text-muted-foreground"
							}`}
						>
							{data.type === "ClusterConfigStore" ? "Global" : "Local"}
						</Badge>
					</div>
					<div className="flex items-center gap-4 text-[10px] font-mono font-bold uppercase tracking-tight">
						<div className="flex items-center gap-2 bg-primary/5 border border-primary/20 px-2 py-1 rounded-md">
							<ProviderIcon provider={data.provider} />
							<span className="text-foreground tracking-normal">
								{data.provider}
							</span>
						</div>
						{data.namespace && (
							<>
								<Separator
									orientation="vertical"
									className="h-3 bg-border/50"
								/>
								<span className="px-1.5 py-0.5 bg-accent/30 rounded border border-border/60 text-foreground/70">
									NS: {data.namespace}
								</span>
							</>
						)}
						{!data.namespace && (
							<>
								<Separator
									orientation="vertical"
									className="h-3 bg-border/50"
								/>
								<span className="text-muted-foreground italic">
									All Namespaces
								</span>
							</>
						)}
					</div>
				</div>
			</div>

			{/* Main Grid */}
			<div className="flex-1 grid grid-cols-12 gap-6 min-h-0 overflow-hidden">
				{/* Left: ExternalConfigs */}
				<div className="col-span-8 flex flex-col min-h-0 overflow-hidden">
					<Card className="flex-1 border-border bg-card/30 flex flex-col min-h-0 overflow-hidden shadow-2xl shadow-black/20">
						<CardHeader className="py-3 px-6 border-b border-border/70 bg-muted/20 shrink-0">
							<CardTitle className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">
								ExternalConfigs using this store ({data.externalConfigs.length})
							</CardTitle>
						</CardHeader>
						<CardContent className="p-0 overflow-y-auto custom-scrollbar flex-1">
							{data.externalConfigs.length === 0 ? (
								<div className="flex items-center justify-center h-32 text-xs text-muted-foreground italic">
									No ExternalConfigs reference this store
								</div>
							) : (
								<table className="w-full text-left text-sm">
									<thead className="sticky top-0 bg-background/95 backdrop-blur-sm z-10 border-b border-border">
										<tr className="text-[10px] uppercase text-muted-foreground tracking-widest">
											<th className="px-6 py-3 font-bold">Name</th>
											<th className="px-6 py-3 font-bold">Namespace</th>
											<th className="px-6 py-3 font-bold">Status</th>
											<th className="px-6 py-3 font-bold text-right">
												Last Sync
											</th>
										</tr>
									</thead>
									<tbody className="divide-y divide-border/60">
										{data.externalConfigs.map((ec) => {
											const statusColor =
												ec.status === "Synced"
													? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
													: ec.status === "Error"
														? "bg-red-500/10 text-red-500 border-red-500/20"
														: "bg-amber-500/10 text-amber-500 border-amber-500/20";
											const StatusIcon =
												ec.status === "Synced"
													? CheckCircle2
													: ec.status === "Error"
														? AlertCircle
														: AlertTriangle;
											return (
												<tr
													key={ec.id}
													onClick={() =>
														navigate({
															to: "/external-configs/$uid",
															params: { uid: ec.uid },
														})
													}
													className="hover:bg-accent/20 transition-colors cursor-pointer odd:bg-muted/10"
												>
													<td className="px-6 py-4 font-bold text-sm">
														{ec.name}
													</td>
													<td className="px-6 py-4">
														<span className="text-xs font-mono font-bold px-2 py-1 bg-accent/50 rounded border border-border/70">
															{ec.namespace}
														</span>
													</td>
													<td className="px-6 py-4">
														<Badge
															className={`px-2 py-0.5 text-[10px] uppercase font-bold tracking-widest gap-1 ${statusColor}`}
														>
															<StatusIcon size={10} />
															{ec.status}
														</Badge>
													</td>
													<td className="px-6 py-4 text-right text-xs font-mono text-muted-foreground">
														{formatAgo(ec.lastSync)}
													</td>
												</tr>
											);
										})}
									</tbody>
								</table>
							)}
						</CardContent>
					</Card>
				</div>

				{/* Right: Connection Info */}
				<div className="col-span-4 flex flex-col gap-4 min-h-0">
					<Card className="border-border bg-card/30 shadow-2xl shadow-black/20">
						<CardHeader className="py-3 px-6 border-b border-border/70 bg-muted/20">
							<CardTitle className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">
								Connection
							</CardTitle>
						</CardHeader>
						<CardContent className="p-6 space-y-4">
							{data.git && (
								<>
									<InfoRow label="URL" value={data.git.url} mono />
									<InfoRow label="Branch" value={data.git.branch} mono />
								</>
							)}
							{data.consul && (
								<>
									<InfoRow label="Address" value={data.consul.address} mono />
									<InfoRow label="Prefix" value={data.consul.prefix} mono />
								</>
							)}
						</CardContent>
					</Card>
				</div>
			</div>
		</motion.div>
	);
}

function InfoRow({
	label,
	value,
	mono = false,
}: {
	label: string;
	value: string;
	mono?: boolean;
}) {
	return (
		<div className="space-y-1">
			<span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
				{label}
			</span>
			<p
				className={`text-xs break-all ${mono ? "font-mono text-foreground/80" : "text-foreground"}`}
			>
				{value}
			</p>
		</div>
	);
}

function ProviderIcon({ provider }: { provider: "Git" | "Consul" }) {
	if (provider === "Git")
		return <GitBranch size={14} className="text-primary/70" />;
	if (provider === "Consul")
		return <Server size={14} className="text-primary/70" />;
	return <Cloud size={14} className="text-primary/70" />;
}
