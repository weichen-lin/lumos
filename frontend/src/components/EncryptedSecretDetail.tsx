import { Link } from "@tanstack/react-router";
import { motion } from "framer-motion";
import {
	AlertCircle,
	AlertTriangle,
	CheckCircle2,
	ChevronLeft,
	Clock,
	FileCode,
	KeyRound,
	LockKeyhole,
	ShieldCheck,
} from "lucide-react";
import { useEncryptedSecretDetailData } from "../hooks/useEncryptedSecretDetailData";
import { formatAgo } from "../lib/time";
import { Badge } from "./ui/badge";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Separator } from "./ui/separator";

export function EncryptedSecretDetail({ uid }: { uid: string }) {
	const { data, isLoading } = useEncryptedSecretDetailData(uid);

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Fetching Encrypted Secret...
				</div>
			</div>
		);
	}

	const statusColor =
		data.status === "Synced"
			? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
			: data.status === "Error"
				? "bg-red-500/10 text-red-500 border-red-500/20"
				: "bg-amber-500/10 text-amber-500 border-amber-500/20";

	const StatusIcon =
		data.status === "Synced"
			? CheckCircle2
			: data.status === "Error"
				? AlertCircle
				: AlertTriangle;

	return (
		<motion.div
			initial={{ opacity: 0, y: 10 }}
			animate={{ opacity: 1, y: 0 }}
			className="h-full flex flex-col gap-6 overflow-hidden"
		>
			{/* Header */}
			<div className="flex items-start justify-between shrink-0">
				<div className="flex items-start gap-4">
					<Link
						to="/encrypted-secrets"
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
								className={`${statusColor} gap-1.5 px-2 py-1 text-[10px] font-bold uppercase tracking-widest`}
							>
								<StatusIcon size={11} />
								{data.status}
							</Badge>
						</div>

						<div className="flex items-center gap-4 text-[10px] font-mono font-bold uppercase tracking-tight">
							<div className="flex items-center gap-2 text-muted-foreground">
								<span className="px-1.5 py-0.5 bg-accent/30 rounded border border-border/60 text-foreground/70">
									NS: {data.namespace}
								</span>
							</div>

							<Separator orientation="vertical" className="h-3 bg-border/50" />

							<div className="flex items-center gap-2 bg-primary/5 border border-primary/20 px-2 py-1 rounded-md shadow-sm">
								<LockKeyhole size={14} className="text-primary/70" />
								<span className="text-foreground tracking-normal">
									{data.targetSecret}
								</span>
							</div>

							<Separator orientation="vertical" className="h-3 bg-border/50" />

							<div className="flex items-center gap-3 text-muted-foreground/60">
								<div className="flex items-center gap-1.5">
									<Clock size={12} />
									<span>
										{data.lastSync ? formatAgo(data.lastSync) : "never"}
									</span>
									<span className="opacity-40 italic">
										@{data.refreshInterval}
									</span>
								</div>
							</div>
						</div>
					</div>
				</div>
			</div>

			{data.status === "Error" && data.message && (
				<div className="flex items-center gap-2 px-3 py-2 rounded-lg bg-red-500/10 border border-red-500/20 text-red-400 text-xs max-w-xl shrink-0">
					<AlertCircle size={13} className="shrink-0" />
					<span>{data.message}</span>
				</div>
			)}

			<Separator className="shrink-0" />

			{/* Body */}
			<div className="flex-1 min-h-0 grid grid-cols-12 gap-6 overflow-hidden">
				{/* Left: details */}
				<div className="col-span-9 flex flex-col gap-6 overflow-y-auto custom-scrollbar pr-1 pb-6">
					{/* Secret Info Card */}
					<Card className="border-border bg-card/30 shadow-2xl shadow-black/20">
						<CardHeader className="px-6 py-4 border-b border-border/40 bg-muted/20">
							<CardTitle className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
								<ShieldCheck size={14} />
								Secret Info
							</CardTitle>
						</CardHeader>
						<CardContent className="p-6 grid grid-cols-2 gap-x-12 gap-y-6">
							<InfoRow label="Git Store">
								<span className="font-bold text-sm">{data.store}</span>
							</InfoRow>
							<InfoRow label="Target K8s Secret">
								<div className="flex items-center gap-2">
									<KeyRound
										size={13}
										className="text-muted-foreground shrink-0"
									/>
									<span className="font-mono font-bold text-sm">
										{data.targetSecret}
									</span>
								</div>
							</InfoRow>
							<InfoRow label="Age Key Secret">
								<span className="font-mono text-sm">{data.ageKeyRef}</span>
							</InfoRow>
							<InfoRow label="Refresh Interval">
								<span className="font-mono text-sm">
									{data.refreshInterval}
								</span>
							</InfoRow>
							<InfoRow label="Last Sync">
								<div className="flex items-center gap-2 text-muted-foreground">
									<Clock size={13} />
									<span className="font-mono text-sm">
										{data.lastSync ? formatAgo(data.lastSync) : "never"}
									</span>
								</div>
							</InfoRow>
							{data.commitSha && (
								<InfoRow label="Commit SHA">
									<span className="font-mono text-xs text-muted-foreground">
										{data.commitSha}
									</span>
								</InfoRow>
							)}
						</CardContent>
					</Card>

					{/* Encrypted Sources Card */}
					<Card className="border-border bg-card/30 shadow-2xl shadow-black/20">
						<CardHeader className="px-6 py-4 border-b border-border/40 bg-muted/20">
							<CardTitle className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
								<FileCode size={14} />
								Encrypted Sources
							</CardTitle>
						</CardHeader>
						<CardContent className="p-4 flex flex-col gap-2">
							{data.sources.map((src) => (
								<div
									key={src}
									className="flex items-center gap-3 px-4 py-3 rounded-lg bg-muted/20 border border-border/30 hover:bg-muted/30 transition-colors"
								>
									<FileCode size={14} className="text-primary/60 shrink-0" />
									<span className="font-mono text-xs font-bold tracking-tight">
										{src}
									</span>
									<Badge className="ml-auto bg-violet-500/10 text-violet-400 border-violet-500/20 text-[9px] font-mono px-2 py-0.5 uppercase">
										SOPS age
									</Badge>
								</div>
							))}
						</CardContent>
					</Card>
				</div>

				{/* Right: Events */}
				<div className="col-span-3 flex flex-col gap-3 overflow-hidden">
					<Card className="flex-1 border-border bg-card/30 flex flex-col min-h-0 overflow-hidden shadow-2xl shadow-black/20">
						<CardHeader className="px-4 py-3 border-b border-border/40 bg-muted/20 shrink-0">
							<CardTitle className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground flex items-center gap-2">
								<Clock size={14} /> Events
							</CardTitle>
						</CardHeader>
						<CardContent className="flex-1 overflow-y-auto p-4 custom-scrollbar">
							{data.events.length === 0 ? (
								<div className="flex items-center justify-center h-full text-muted-foreground/50 text-xs py-8 italic">
									No events recorded
								</div>
							) : (
								<div className="relative space-y-5 before:absolute before:left-1 before:top-2 before:bottom-2 before:w-px before:bg-border/50">
									{data.events.map((ev) => (
										<div key={ev.id} className="relative pl-6 group">
											<div
												className={`absolute left-0 top-1 w-2 h-2 rounded-full border bg-background z-10 ${
													ev.type === "Warning"
														? "border-amber-500"
														: "border-emerald-500"
												}`}
											/>
											<div className="space-y-1">
												<div className="flex items-center justify-between">
													<span
														className={`text-[9px] font-bold uppercase tracking-widest ${
															ev.type === "Warning"
																? "text-amber-500"
																: "text-emerald-500"
														}`}
													>
														{ev.reason}
													</span>
													{ev.time && (
														<span className="text-[8px] font-mono text-muted-foreground/50">
															{formatAgo(ev.time)}
														</span>
													)}
												</div>
												<p className="text-[11px] text-muted-foreground leading-snug break-words">
													{ev.message}
												</p>
											</div>
										</div>
									))}
								</div>
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
	children,
}: {
	label: string;
	children: React.ReactNode;
}) {
	return (
		<div className="flex flex-col gap-1">
			<span className="text-[10px] font-bold uppercase tracking-widest text-muted-foreground">
				{label}
			</span>
			<div>{children}</div>
		</div>
	);
}
