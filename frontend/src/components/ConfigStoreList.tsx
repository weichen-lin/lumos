import { useNavigate } from "@tanstack/react-router";
import { motion } from "framer-motion";
import { Database, GitBranch, Search } from "lucide-react";
import { useState } from "react";
import {
	type ConfigStore,
	useConfigStoreData,
} from "../hooks/useConfigStoreData";
import { Badge } from "./ui/badge";
import { Card, CardContent } from "./ui/card";
import { Input } from "./ui/input";

export function ConfigStoreList() {
	const { data, isLoading } = useConfigStoreData();
	const navigate = useNavigate();
	const [query, setQuery] = useState("");

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Mapping Config Infrastructure...
				</div>
			</div>
		);
	}

	const filtered = data.filter((s) =>
		[s.name, s.namespace ?? "", s.provider]
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
						placeholder="Search by name, provider or namespace..."
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
								<Database size={24} className="text-muted-foreground/30" />
							</div>
							<div className="flex flex-col items-center gap-1 text-center">
								<p className="text-sm font-medium text-muted-foreground/60">
									No config stores found
								</p>
								<p className="text-xs text-muted-foreground/40">
									ConfigStore and ClusterConfigStore resources will appear here
								</p>
							</div>
						</div>
					) : (
						<table className="w-full text-left text-sm">
							<thead className="sticky top-0 bg-background/95 backdrop-blur-md z-10 border-b border-border">
								<tr className="bg-muted/30">
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Resource
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Scope
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Provider
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px]">
										Namespace
									</th>
									<th className="px-6 py-4 font-bold uppercase tracking-wider text-muted-foreground text-[10px] text-center">
										Usage
									</th>
								</tr>
							</thead>
							<tbody className="divide-y divide-border/70">
								{filtered.map((store) => (
									<tr
										key={store.id}
										onClick={() =>
											navigate({
												to: "/config-stores/$uid",
												params: { uid: store.uid },
											})
										}
										className="hover:bg-accent/30 transition-colors group cursor-pointer odd:bg-muted/10"
									>
										<td className="px-6 py-5">
											<div className="flex flex-col gap-1">
												<span className="font-bold text-base tracking-tight">
													{store.name}
												</span>
												<span className="text-[10px] font-mono text-muted-foreground uppercase">
													{store.type}
												</span>
											</div>
										</td>
										<td className="px-6 py-5">
											<Badge
												variant="secondary"
												className={`text-[9px] h-5 px-2 font-bold uppercase tracking-widest ${
													store.type === "ClusterConfigStore"
														? "bg-primary/10 text-primary border-primary/20"
														: "bg-muted/50 text-muted-foreground"
												}`}
											>
												{store.type === "ClusterConfigStore"
													? "Global"
													: "Local"}
											</Badge>
										</td>
										<td className="px-6 py-5">
											<div className="flex items-center gap-3">
												<ProviderIcon provider={store.provider} />
												<span className="font-bold text-xs uppercase tracking-wider">
													{store.provider}
												</span>
											</div>
										</td>
										<td className="px-6 py-5">
											{store.namespace ? (
												<span className="text-xs font-mono font-bold px-2 py-1 bg-accent/50 rounded border border-border/70">
													{store.namespace}
												</span>
											) : (
												<span className="text-[10px] uppercase font-bold text-muted-foreground italic">
													All Namespaces
												</span>
											)}
										</td>
										<td className="px-6 py-5 text-center font-mono text-lg font-black text-foreground">
											{store.usageCount}
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

function ProviderIcon({ provider }: { provider: ConfigStore["provider"] }) {
	if (provider === "Git")
		return (
			<div className="p-1.5 bg-white/10 rounded-md shrink-0 flex items-center justify-center w-8 h-8">
				<GitBranch size={16} className="text-foreground/70" />
			</div>
		);
	return (
		<div className="p-1.5 bg-white/10 rounded-md shrink-0 flex items-center justify-center w-8 h-8">
			<GitBranch size={16} className="text-foreground/70" />
		</div>
	);
}
