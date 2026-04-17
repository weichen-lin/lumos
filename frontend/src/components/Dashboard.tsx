import { motion } from "framer-motion";
import {
	AlertTriangle,
	CheckCircle2,
	Clock,
	GitCommit,
	XCircle,
} from "lucide-react";
import type { ReactNode } from "react";
import { Cell, Pie, PieChart, ResponsiveContainer, Tooltip } from "recharts";
import { useDashboardData } from "../hooks/useDashboardData";
import { formatAgo } from "../lib/time";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";

export function Dashboard() {
	const { data, isLoading } = useDashboardData();

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-64">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Initializing Cluster Sync...
				</div>
			</div>
		);
	}

	return (
		<motion.div
			initial={{ opacity: 0, y: 10 }}
			animate={{ opacity: 1, y: 0 }}
			className="h-full flex flex-col gap-8 overflow-y-auto custom-scrollbar pb-8"
		>
			{/* Status Summary */}
			<div className="grid grid-cols-2 md:grid-cols-4 gap-4 shrink-0">
				<StatusCard
					label="Synced"
					value={data.summary.synced}
					icon={<CheckCircle2 className="text-emerald-500" size={20} />}
					subtitle="Total Objects"
				/>
				<StatusCard
					label="Error"
					value={data.summary.error}
					icon={<XCircle className="text-destructive" size={20} />}
					subtitle="Fix Required"
					error
				/>
				<StatusCard
					label="Stale"
					value={data.summary.stale}
					icon={<Clock className="text-amber-500" size={20} />}
					subtitle="Sync Pending"
				/>
				<StatusCard
					label="Health Score"
					value={calculateHealthScore(data.summary)}
					icon={<AlertTriangle className="text-orange-500" size={20} />}
					subtitle="Cluster Status"
					isPercentage
				/>
			</div>

			{/* Main Grid */}
			<div className="grid grid-cols-1 md:grid-cols-12 gap-6 min-h-0">
				<div className="md:col-span-4">
					<ProviderDistributionCard providers={data.providers} />
				</div>
				<div className="md:col-span-8">
					<ActivityFeedCard events={data.recentEvents} />
				</div>
			</div>

			{/* Bottom Grid */}
			<div className="grid grid-cols-1 gap-6">
				<NamespaceHealthCard namespaces={data.namespaceHealth} />
			</div>
		</motion.div>
	);
}

function calculateHealthScore(summary: {
	synced: number;
	error: number;
	stale: number;
}) {
	const total = summary.synced + summary.error + summary.stale;
	if (total === 0) return 100;
	return Math.round((summary.synced / total) * 100);
}

function StatusCard({
	label,
	value,
	icon,
	subtitle,
	error = false,
	isPercentage = false,
}: {
	label: string;
	value: number;
	icon: ReactNode;
	subtitle?: string;
	error?: boolean;
	isPercentage?: boolean;
}) {
	return (
		<Card
			className={`relative group transition-all bg-card/30 hover:bg-card/50 overflow-hidden ${error ? "border-red-500/40 hover:border-red-500/60" : "border-border hover:border-primary/30"}`}
		>
			<CardContent className="p-6 flex items-center justify-between relative z-10">
				<div className="space-y-1">
					<p
						className={`text-[10px] font-bold uppercase tracking-widest ${error ? "text-red-500/70" : "text-muted-foreground"}`}
					>
						{label}
					</p>
					<h3
						className={`text-4xl font-black tracking-tighter italic ${error ? "text-red-500" : ""}`}
					>
						{value}
						{isPercentage ? "%" : ""}
					</h3>
					<p className="text-[9px] text-muted-foreground font-medium uppercase tracking-wider">
						{subtitle}
					</p>
				</div>
				<div
					className={`p-3 rounded-xl group-hover:scale-110 transition-transform ${error ? "bg-red-500/10" : "bg-accent/50"}`}
				>
					{icon}
				</div>
			</CardContent>
		</Card>
	);
}

function ProviderDistributionCard({
	providers,
}: {
	providers: Array<{ name: string; value: number; color: string }>;
}) {
	const total = providers.reduce((sum, p) => sum + p.value, 0);

	return (
		<Card className="h-full border-border bg-card/30 shadow-xl shadow-black/5 flex flex-col">
			<CardHeader className="border-b border-border/40 bg-muted/20 py-3">
				<CardTitle className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest text-center">
					Provider Distribution
				</CardTitle>
			</CardHeader>
			<CardContent className="flex-1 flex flex-col items-center justify-center p-6 gap-6">
				{/* Chart on Top */}
				<div className="w-full h-[160px]">
					<ResponsiveContainer width="100%" height="100%">
						<PieChart>
							<Pie
								data={providers}
								cx="50%"
								cy="50%"
								innerRadius={45}
								outerRadius={75}
								paddingAngle={4}
								dataKey="value"
							>
								{providers.map((p, i) => (
									<Cell key={i} fill={p.color} strokeWidth={0} />
								))}
							</Pie>
							<Tooltip
								wrapperStyle={{ zIndex: 10 }}
								content={({ active, payload }) => {
									if (!active || !payload?.length) return null;
									const { name, value } = payload[0];
									const pct =
										total > 0 ? Math.round((Number(value) / total) * 100) : 0;
									return (
										<div className="bg-card/90 backdrop-blur-md border border-border rounded-lg px-3 py-2 text-xs shadow-2xl">
											<p className="font-bold text-foreground">{name}</p>
											<p className="text-muted-foreground font-mono">
												{value} Objects ({pct}%)
											</p>
										</div>
									);
								}}
							/>
						</PieChart>
					</ResponsiveContainer>
				</div>

				{/* Legend Below, Arranged Horizontally */}
				<div className="flex flex-wrap justify-center gap-x-8 gap-y-4 px-2 w-full">
					{providers.map((p) => {
						const pct = total > 0 ? Math.round((p.value / total) * 100) : 0;
						return (
							<div
								key={p.name}
								className="flex items-center gap-3 text-xs group"
							>
								<div
									className="w-2.5 h-2.5 rounded-full shrink-0 shadow-[0_0_8px_rgba(0,0,0,0.2)]"
									style={{ backgroundColor: p.color }}
								/>
								<div className="flex flex-col">
									<span className="font-black group-hover:text-primary transition-colors text-muted-foreground group-hover:text-foreground uppercase italic tracking-tighter leading-none mb-1">
										{p.name}
									</span>
									<span className="font-mono text-[10px] text-muted-foreground/40 font-bold">
										{p.value}{" "}
										<span className="text-[8px] opacity-60">({pct}%)</span>
									</span>
								</div>
							</div>
						);
					})}
				</div>
			</CardContent>
		</Card>
	);
}

function ActivityFeedCard({
	events,
}: {
	events: Array<{
		id: string;
		status: string;
		time: string | null;
		target: string;
		message: string;
		commitSha?: string;
	}>;
}) {
	return (
		<Card className="h-full border-border bg-card/30 shadow-xl shadow-black/5 flex flex-col overflow-hidden">
			<CardContent className="p-0 overflow-y-auto flex-1 custom-scrollbar">
				<table className="w-full text-left border-collapse table-fixed">
					<thead>
						<tr className="border-b border-border/10 text-[9px] uppercase tracking-widest text-muted-foreground/60 bg-muted/5">
							<th className="px-4 py-3 font-bold w-12 text-center">Stat</th>
							<th className="px-4 py-3 font-bold w-24">Time</th>
							<th className="px-4 py-3 font-bold w-48">Target</th>
							<th className="px-4 py-3 font-bold">Message</th>
							<th className="px-4 py-3 font-bold w-24 text-right pr-6">
								Commit
							</th>
						</tr>
					</thead>
					<tbody className="divide-y divide-border/10">
						{events.length === 0 ? (
							<tr>
								<td
									colSpan={5}
									className="p-12 text-center text-xs text-muted-foreground italic"
								>
									No recent activity detected
								</td>
							</tr>
						) : (
							events.map((event) => (
								<tr key={event.id} className="transition-all group">
									<td className="px-4 py-3 align-middle">
										<div className="flex justify-center">
											{event.status === "error" && (
												<XCircle size={14} className="text-destructive" />
											)}
											{event.status === "warning" && (
												<AlertTriangle size={14} className="text-amber-500" />
											)}
											{event.status === "synced" && (
												<CheckCircle2 size={14} className="text-emerald-500" />
											)}
										</div>
									</td>
									<td className="px-4 py-3 align-middle">
										<span className="text-[10px] font-mono font-bold text-muted-foreground/60">
											{formatAgo(event.time)}
										</span>
									</td>
									<td className="px-4 py-3 align-middle overflow-hidden">
										<span className="text-xs font-black text-foreground truncate uppercase italic tracking-tight block">
											{event.target}
										</span>
									</td>
									<td className="px-4 py-3 align-middle overflow-hidden">
										<span className="text-[11px] text-muted-foreground font-medium truncate block">
											{event.message}
										</span>
									</td>
									<td className="px-4 py-3 align-middle text-right">
										{event.commitSha && (
											<div className="inline-flex items-center gap-1.5 font-mono text-[9px] bg-primary/5 px-2 py-0.5 rounded border border-primary/20">
												<GitCommit size={10} className="text-primary/60" />
												<span className="text-primary/70 font-bold">
													{event.commitSha}
												</span>
											</div>
										)}
									</td>
								</tr>
							))
						)}
					</tbody>
				</table>
			</CardContent>
		</Card>
	);
}

function NamespaceHealthCard({
	namespaces,
}: {
	namespaces: Array<{
		name: string;
		synced: number;
		error: number;
		stale: number;
		coverage: number;
	}>;
}) {
	return (
		<Card className="border-border bg-card/30 overflow-hidden shadow-xl shadow-black/5">
			<CardHeader className="border-b border-border/40 bg-muted/20 py-3">
				<CardTitle className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest">
					Namespace Health Matrix
				</CardTitle>
			</CardHeader>
			<CardContent className="p-0">
				<div className="overflow-x-auto">
					<table className="w-full text-left text-xs table-fixed">
						<thead>
							<tr className="border-b border-border bg-muted/10">
								<th className="px-6 py-4 font-black uppercase tracking-widest text-muted-foreground text-[10px] w-64">
									Namespace
								</th>
								<th className="px-6 py-4 font-black uppercase tracking-widest text-muted-foreground text-[10px] w-32 text-center">
									Synced
								</th>
								<th className="px-6 py-4 font-black uppercase tracking-widest text-destructive text-[10px] w-32 text-center">
									Error
								</th>
								<th className="px-6 py-4 font-black uppercase tracking-widest text-amber-500 text-[10px] w-32 text-center">
									Stale
								</th>
								<th className="px-6 py-4 font-black uppercase tracking-widest text-muted-foreground text-[10px] text-right">
									Coverage
								</th>
							</tr>
						</thead>
						<tbody className="divide-y divide-border/20">
							{namespaces.map((ns) => (
								<tr
									key={ns.name}
									className="group hover:bg-accent/30 transition-colors"
								>
									<td className="px-6 py-4">
										<span className="font-black text-sm uppercase italic tracking-tighter group-hover:text-primary transition-colors">
											{ns.name}
										</span>
									</td>
									<td className="px-6 py-4">
										<div className="flex items-center justify-center gap-2">
											<div className="w-1.5 h-1.5 rounded-full bg-emerald-500 shadow-[0_0_8px_rgba(16,185,129,0.5)]" />
											<span className="font-mono font-bold text-emerald-500 text-sm">
												{ns.synced}
											</span>
										</div>
									</td>
									<td className="px-6 py-4">
										<div className="flex items-center justify-center gap-2">
											<div
												className={`w-1.5 h-1.5 rounded-full ${ns.error > 0 ? "bg-destructive shadow-[0_0_8px_rgba(239,68,68,0.5)]" : "bg-border"}`}
											/>
											<span
												className={`font-mono font-bold text-sm ${ns.error > 0 ? "text-destructive" : "text-muted-foreground/40"}`}
											>
												{ns.error}
											</span>
										</div>
									</td>
									<td className="px-6 py-4">
										<div className="flex items-center justify-center gap-2">
											<div
												className={`w-1.5 h-1.5 rounded-full ${ns.stale > 0 ? "bg-amber-500 shadow-[0_0_8px_rgba(245,158,11,0.5)]" : "bg-border"}`}
											/>
											<span
												className={`font-mono font-bold text-sm ${ns.stale > 0 ? "text-amber-500" : "text-muted-foreground/40"}`}
											>
												{ns.stale}
											</span>
										</div>
									</td>
									<td className="px-6 py-4">
										<div className="flex items-center justify-end gap-4">
											<span className="font-mono font-bold text-sm w-12 text-right">
												{ns.coverage}%
											</span>
											<div className="w-48 h-2 bg-muted rounded-full overflow-hidden border border-border/20">
												<motion.div
													initial={{ width: 0 }}
													animate={{ width: `${ns.coverage}%` }}
													className={`h-full shadow-[0_0_10px_rgba(0,0,0,0.2)] ${
														ns.coverage > 95
															? "bg-emerald-500"
															: ns.coverage > 80
																? "bg-amber-500"
																: "bg-destructive"
													}`}
												/>
											</div>
										</div>
									</td>
								</tr>
							))}
						</tbody>
					</table>
				</div>
			</CardContent>
		</Card>
	);
}
