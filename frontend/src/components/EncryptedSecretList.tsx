import { useNavigate } from "@tanstack/react-router";
import { motion } from "framer-motion";
import {
	AlertCircle,
	AlertTriangle,
	CheckCircle2,
	Clock,
	KeyRound,
	LockKeyhole,
	Search,
} from "lucide-react";
import { useState } from "react";
import {
	type EncryptedSecret,
	useEncryptedSecretsData,
} from "../hooks/useEncryptedSecretsData";
import { formatAgo } from "../lib/time";
import { Badge } from "./ui/badge";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";

export function EncryptedSecretList() {
	const { data, isLoading } = useEncryptedSecretsData();
	const navigate = useNavigate();
	const [query, setQuery] = useState("");

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Loading Encrypted Secrets...
				</div>
			</div>
		);
	}

	const filtered = data.filter((s) =>
		[s.name, s.namespace, s.store]
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
						className="pl-9 h-10 bg-card/30 border-border/50 text-sm focus-visible:ring-primary/30"
						value={query}
						onChange={(e) => setQuery(e.target.value)}
					/>
				</div>
			</div>

			{/* Table */}
			<Card className="flex-1 border-border/50 bg-card/30 flex flex-col min-h-0 overflow-hidden">
				<CardContent className="flex-1 overflow-y-auto p-0 custom-scrollbar">
					{data.length === 0 ? (
						<div className="flex flex-col items-center justify-center h-full py-20 gap-4">
							<div className="p-4 rounded-full bg-muted/30">
								<LockKeyhole size={24} className="text-muted-foreground/30" />
							</div>
							<div className="flex flex-col items-center gap-1 text-center">
								<p className="text-sm font-medium text-muted-foreground/60">
									No encrypted secrets found
								</p>
								<p className="text-xs text-muted-foreground/40">
									EncryptedSecret resources will appear here once created
								</p>
							</div>
						</div>
					) : (
						<table className="w-full text-left text-sm">
							<thead className="sticky top-0 bg-background/95 backdrop-blur-md z-10 border-b border-border/50">
								<tr className="bg-muted/30">
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Name
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Namespace
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Store
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Target Secret
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Sources
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Status
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Last Sync
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/30">
								{filtered.map((secret) => (
									<tr
										key={secret.uid}
										onClick={() =>
											navigate({
												to: "/encrypted-secrets/$uid",
												params: { uid: secret.uid },
											})
										}
										className="hover:bg-accent/30 transition-colors group cursor-pointer odd:bg-muted/10"
									>
										<td className="px-6 py-5">
											<span className="font-bold text-base tracking-tight">
												{secret.name}
											</span>
										</td>
										<td className="px-6 py-5">
											<span className="text-xs font-mono font-bold px-2 py-1 bg-accent/50 rounded border border-border/30">
												{secret.namespace}
											</span>
										</td>
										<td className="px-6 py-5">
											<span className="font-bold text-xs">{secret.store}</span>
										</td>
										<td className="px-6 py-5">
											<div className="flex items-center gap-1.5 text-muted-foreground">
												<KeyRound size={12} />
												<span className="text-xs font-mono">
													{secret.targetSecret}
												</span>
											</div>
										</td>
										<td className="px-6 py-5">
											<span className="text-xs font-mono text-muted-foreground">
												{secret.sources.length} file
												{secret.sources.length !== 1 ? "s" : ""}
											</span>
										</td>
										<td className="px-6 py-5">
											<StatusBadge
												status={secret.status}
												message={secret.message}
											/>
										</td>
										<td className="px-6 py-5">
											<div className="flex items-center gap-2 text-muted-foreground">
												<Clock size={12} />
												<span className="text-xs font-mono">
													{secret.lastSync
														? formatAgo(secret.lastSync)
														: "never"}
												</span>
											</div>
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

function StatusBadge({
	status,
	message,
}: {
	status: EncryptedSecret["status"];
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
		<Badge className="bg-amber-500/10 text-amber-500 border-amber-500/20 gap-1.5 px-2 py-1 w-fit">
			<AlertTriangle size={12} />
			<span className="text-[10px] font-bold uppercase tracking-widest leading-none">
				Stale
			</span>
		</Badge>
	);
}
