import { Link } from "@tanstack/react-router";
import { AnimatePresence, motion } from "framer-motion";
import {
	AlertCircle,
	ArrowRight,
	Check,
	ChevronDown,
	ChevronLeft,
	ChevronRight,
	Clock,
	Cloud,
	Copy,
	Eye,
	FileCode2,
	GitBranch,
	GitCommit,
	Maximize2,
	Server,
	X,
} from "lucide-react";
import { useState } from "react";
import {
	type ConfigDetail,
	useExternalConfigDetailData,
} from "../hooks/useExternalConfigDetailData";
import { formatAgo } from "../lib/time";
import { Badge } from "./ui/badge";
import { Button } from "./ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "./ui/card";
import { Separator } from "./ui/separator";

export function ExternalConfigDetail({ configId }: { configId: string }) {
	const { data, isLoading } = useExternalConfigDetailData(configId);
	const [selectedItem, setSelectedItem] = useState<{
		key: string;
		value: string;
		source: string;
	} | null>(null);
	const [collapsedSources, setCollapsedSources] = useState<Set<string>>(
		new Set(),
	);

	if (isLoading || !data) {
		return (
			<div className="flex items-center justify-center h-full">
				<div className="animate-pulse text-muted-foreground font-mono text-xs tracking-widest uppercase">
					Fetching Config State...
				</div>
			</div>
		);
	}

	const toggleSource = (source: string) => {
		const next = new Set(collapsedSources);
		if (next.has(source)) {
			next.delete(source);
		} else {
			next.add(source);
		}
		setCollapsedSources(next);
	};

	const statusColor =
		data.status === "Synced"
			? "bg-emerald-500/10 text-emerald-500 border-emerald-500/20"
			: data.status === "Error"
				? "bg-red-500/10 text-red-500 border-red-500/20"
				: "bg-amber-500/10 text-amber-500 border-amber-500/20";

	// Group data by source
	const groupedData = data.data.reduce(
		(acc, item) => {
			if (!acc[item.source]) {
				acc[item.source] = [];
			}
			acc[item.source].push(item);
			return acc;
		},
		{} as Record<string, typeof data.data>,
	);

	return (
		<motion.div
			initial={{ opacity: 0, y: 10 }}
			animate={{ opacity: 1, y: 0 }}
			className="h-full flex flex-col gap-6 overflow-hidden relative"
		>
			{/* Header */}
			<div className="flex items-start gap-4 shrink-0">
				<Link
					to="/external-configs"
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
							className={`px-2 py-0.5 text-[10px] uppercase font-bold tracking-widest ${statusColor}`}
						>
							{data.status}
						</Badge>
						<div className="flex gap-1.5">
							{Object.entries(data.labels).map(([k, v]) => (
								<Badge
									key={k}
									variant="outline"
									className="text-[9px] h-5 uppercase font-bold text-muted-foreground/70 border-border/70 bg-muted/20"
								>
									{v}
								</Badge>
							))}
						</div>
					</div>

					<div className="flex items-center gap-4 text-[10px] font-mono font-bold uppercase tracking-tight">
						<div className="flex items-center gap-2 text-muted-foreground">
							<span className="px-1.5 py-0.5 bg-accent/30 rounded border border-border/60 text-foreground/70">
								NS: {data.namespace}
							</span>
						</div>

						<Separator orientation="vertical" className="h-3 bg-border/50" />

						<div className="flex items-center gap-2 bg-primary/5 border border-primary/20 px-2 py-1 rounded-md shadow-sm">
							<ProviderIconCompact provider={data.store.provider} />
							<span className="text-foreground tracking-normal">
								{data.store.name}
							</span>
							<span className="text-primary/60 text-[9px] font-black">
								({data.store.type})
							</span>
						</div>

						<Separator orientation="vertical" className="h-3 bg-border/50" />

						<div className="flex items-center gap-3 text-muted-foreground/60">
							<div className="flex items-center gap-1.5">
								<Clock size={12} />
								<span>{formatAgo(data.lastSync)}</span>
								<span className="opacity-40 italic">
									@{data.refreshInterval}
								</span>
							</div>
						</div>
					</div>
				</div>
			</div>

			{/* Main Grid */}
			<div className="flex-1 grid grid-cols-12 gap-6 min-h-0 overflow-hidden">
				{/* Content Area */}
				<div className="col-span-9 flex flex-col gap-6 min-h-0 overflow-hidden">
					{data.status === "Error" ? (
						<ErrorView message={data.message} />
					) : (
						<Card className="border-border bg-card/30 flex flex-col min-h-0 overflow-hidden shadow-2xl shadow-black/20">
							<CardContent className="p-0 overflow-y-auto custom-scrollbar [scrollbar-gutter:stable] bg-muted/30 dark:bg-card/10">
								<table className="w-full text-left text-sm border-collapse table-fixed">
									<colgroup>
										<col />
										<col />
										<col className="w-[180px]" />
									</colgroup>
									<thead className="sticky top-0 bg-background/95 backdrop-blur-sm z-10 border-b border-border text-[10px] uppercase text-muted-foreground tracking-widest">
										<tr>
											<th className="px-6 py-3 font-bold">ConfigMap Key</th>
											<th className="px-6 py-3 font-bold">Value</th>
											<th className="px-6 py-3 font-bold text-right">
												Last Changed
											</th>
										</tr>
									</thead>
									<tbody className="">
										{Object.keys(groupedData).length === 0 ? (
											<tr>
												<td
													colSpan={3}
													className="px-6 py-10 text-center text-xs text-muted-foreground italic"
												>
													No keys produced from remote source
												</td>
											</tr>
										) : (
											Object.entries(groupedData).map(([source, items]) => (
												<SourceGroup
													key={source}
													source={source}
													items={items}
													isCollapsed={collapsedSources.has(source)}
													onToggle={() => toggleSource(source)}
													onSelect={(item) =>
														setSelectedItem({
															key: item.key,
															value: item.value,
															source: item.source,
														})
													}
												/>
											))
										)}
									</tbody>
								</table>
							</CardContent>
						</Card>
					)}
				</div>

				{/* Right: Events */}
				<div className="col-span-3 flex flex-col min-h-0 overflow-hidden">
					<Card className="flex-1 border-border bg-card/50 flex flex-col min-h-0 overflow-hidden shadow-lg shadow-black/5">
						<CardHeader className="py-3 px-6 border-b border-border/70 bg-muted/40 dark:bg-muted/20">
							<CardTitle className="text-[10px] uppercase font-bold text-muted-foreground tracking-widest flex items-center gap-2">
								<Clock size={14} /> Events
							</CardTitle>
						</CardHeader>
						<CardContent className="flex-1 overflow-y-auto p-4 custom-scrollbar">
							<div className="relative space-y-5 before:absolute before:left-1 before:top-2 before:bottom-2 before:w-px before:bg-border/50">
								{data.events.map((event) => (
									<div key={event.id} className="relative pl-6 group">
										<div
											className={`absolute left-0 top-1 w-2 h-2 rounded-full border bg-background z-10 ${event.type === "Warning" ? "border-destructive" : "border-emerald-500"}`}
										/>
										<div className="space-y-0.5">
											<div className="flex items-center justify-between">
												<span
													className={`text-[9px] font-bold uppercase tracking-widest ${event.type === "Warning" ? "text-destructive" : "text-emerald-500"}`}
												>
													{event.reason}
												</span>
												<span className="text-[8px] font-mono text-muted-foreground">
													{formatAgo(event.time)}
												</span>
											</div>
											<p className="text-[11px] text-muted-foreground leading-tight">
												{event.message}
											</p>
										</div>
									</div>
								))}
							</div>
						</CardContent>
					</Card>
				</div>
			</div>

			{/* Modal */}
			<AnimatePresence>
				{selectedItem && (
					<ValueModal
						item={selectedItem}
						onClose={() => setSelectedItem(null)}
					/>
				)}
			</AnimatePresence>
		</motion.div>
	);
}

function SourceGroup({
	source,
	items,
	isCollapsed,
	onToggle,
	onSelect,
}: {
	source: string;
	items: ConfigDetail["data"];
	isCollapsed: boolean;
	onToggle: () => void;
	onSelect: (item: ConfigDetail["data"][0]) => void;
}) {
	// Accurate check using the 'format' field from back-end
	const isRawMode = items[0]?.format === "Raw";

	return (
		<>
			<tr
				className="bg-muted/40 border-y border-border/40 cursor-pointer hover:bg-muted/60 transition-colors"
				onClick={onToggle}
			>
				<td colSpan={3} className="px-6 py-2">
					<div className="flex items-center gap-3">
						<div className="text-muted-foreground/60 transition-transform duration-200">
							{isCollapsed ? (
								<ChevronRight size={14} />
							) : (
								<ChevronDown size={14} />
							)}
						</div>
						<div className="flex items-center gap-2 overflow-hidden">
							<FileCode2 size={12} className="text-primary/60 shrink-0" />
							<span className="text-[10px] font-mono font-bold text-primary tracking-tight truncate">
								SOURCE: {source}
							</span>
							<Badge
								variant="outline"
								className="h-4 text-[8px] px-1 font-mono uppercase bg-background/50 shrink-0"
							>
								{items.length} {items.length === 1 ? "Key" : "Keys"}
							</Badge>
							<Badge className="ml-1 h-3.5 bg-muted/80 text-[7px] uppercase tracking-tighter border-none text-muted-foreground shrink-0">
								Format: {items[0]?.format || "Raw"}
							</Badge>
						</div>
					</div>
				</td>
			</tr>
			{!isCollapsed &&
				items.map((item, idx) => (
					<motion.tr
						initial={{ opacity: 0, y: -4 }}
						animate={{ opacity: 1, y: 0 }}
						key={`${source}-${item.key}-${idx}`}
						onClick={isRawMode ? () => onSelect(item) : undefined}
						className={`odd:bg-muted/10 ${isRawMode ? "hover:bg-accent/30 cursor-pointer active:scale-[0.995]" : ""} transition-all group border-b border-border/20 last:border-b-0`}
					>
						<td className="px-6 py-4 overflow-hidden">
							<div className="flex items-center gap-3 min-w-0">
								<div className="w-1.5 h-1.5 rounded-full bg-primary/20 group-hover:bg-primary transition-colors shrink-0" />
								<span className="font-bold text-xs font-mono tracking-wider group-hover:text-primary transition-colors truncate">
									{item.key}
								</span>
							</div>
						</td>
						<td className="px-6 py-4 overflow-hidden text-ellipsis">
							{isRawMode ? (
								<div className="flex items-center gap-2 text-muted-foreground group-hover:text-foreground/70 transition-colors">
									<Eye size={12} />
									<span className="text-[10px] uppercase font-bold tracking-widest">
										Inspect Value
									</span>
								</div>
							) : (
								<div className="min-w-0">
									<code className="text-[11px] font-mono text-foreground/80 bg-muted/30 px-2 py-0.5 rounded border border-border/50 truncate block">
										{item.value}
									</code>
								</div>
							)}
						</td>
						<td className="px-6 py-4 text-right shrink-0">
							<div className="flex flex-col items-end gap-1">
								{item.commitSha && (
									<div className="inline-flex items-center gap-1 font-mono text-[9px] text-primary/60">
										<GitCommit size={10} />
										<span>{item.commitSha}</span>
									</div>
								)}
								<span className="text-[10px] font-mono text-muted-foreground italic">
									{formatAgo(item.lastChanged)}
								</span>
							</div>
						</td>
					</motion.tr>
				))}
		</>
	);
}

function ErrorView({ message }: { message?: string }) {
	return (
		<motion.div
			initial={{ opacity: 0, scale: 0.98 }}
			animate={{ opacity: 1, scale: 1 }}
			className="h-full flex flex-col items-center justify-center p-12 text-center bg-red-500/5 border border-red-500/20 rounded-xl overflow-hidden relative"
		>
			<div className="absolute inset-0 bg-[radial-gradient(circle_at_50%_50%,rgba(239,68,68,0.1),transparent_70%)]" />

			<div className="relative space-y-6 max-w-lg">
				<div className="inline-flex items-center justify-center w-20 h-20 rounded-full bg-red-500/10 border border-red-500/30 text-red-500 shadow-[0_0_30px_-5px_rgba(239,68,68,0.4)]">
					<AlertCircle size={40} strokeWidth={2.5} />
				</div>

				<div className="space-y-2">
					<h3 className="text-2xl font-black uppercase italic tracking-tighter text-red-500">
						Synchronization Failure
					</h3>
					<p className="text-sm font-medium text-red-400/80 leading-relaxed font-mono">
						{message ||
							"The operator encountered an unrecoverable error while fetching remote data."}
					</p>
				</div>

				<div className="pt-4 flex flex-col items-center gap-4">
					<div className="w-full h-px bg-gradient-to-r from-transparent via-red-500/30 to-transparent" />
					<p className="text-[10px] uppercase font-bold tracking-[0.2em] text-muted-foreground/60">
						Please check controller logs for more details
					</p>
				</div>
			</div>
		</motion.div>
	);
}

function ValueModal({
	item,
	onClose,
}: {
	item: { key: string; value: string; source: string };
	onClose: () => void;
}) {
	const [copied, setCopied] = useState(false);
	const isJson = (() => {
		try {
			JSON.parse(item.value);
			return true;
		} catch {
			return false;
		}
	})();

	const formattedValue = isJson
		? JSON.stringify(JSON.parse(item.value), null, 2)
		: item.value;

	const handleCopy = () => {
		navigator.clipboard.writeText(formattedValue);
		setCopied(true);
		setTimeout(() => setCopied(false), 2000);
	};

	return (
		<motion.div
			initial={{ opacity: 0 }}
			animate={{ opacity: 1 }}
			exit={{ opacity: 0 }}
			className="fixed inset-0 z-[100] flex items-center justify-center p-8 bg-background/80 backdrop-blur-md"
			onClick={onClose}
		>
			<motion.div
				initial={{ opacity: 0, scale: 0.95, y: 20 }}
				animate={{ opacity: 1, scale: 1, y: 0 }}
				exit={{ opacity: 0, scale: 0.95, y: 20 }}
				className="w-full max-w-4xl max-h-[85vh] bg-card border border-border shadow-2xl rounded-xl flex flex-col overflow-hidden"
				onClick={(e) => e.stopPropagation()}
			>
				{/* Modal Header */}
				<div className="px-6 py-4 border-b border-border bg-card flex items-center justify-between shrink-0">
					<div className="flex flex-col text-left">
						<div className="flex items-center gap-3">
							<Maximize2 size={16} className="text-primary" />
							<h3 className="text-lg font-black uppercase italic tracking-tighter">
								Value Detail
							</h3>
							<Badge className="bg-primary/10 text-primary border-primary/20 text-[9px] uppercase font-bold">
								{isJson ? "JSON" : "Raw Text"}
							</Badge>
						</div>
						<div className="flex items-center gap-2 mt-1 text-[10px] font-mono text-muted-foreground">
							<span className="font-bold text-foreground/60">SOURCE:</span>
							<span>{item.source}</span>
							<ArrowRight size={10} />
							<span className="font-bold text-foreground/60">KEY:</span>
							<span className="text-primary/80">{item.key}</span>
						</div>
					</div>
					<div className="flex items-center gap-2">
						<Button
							variant="outline"
							size="icon"
							className={`h-8 w-8 transition-colors ${copied ? "text-emerald-500 border-emerald-500/50" : "text-muted-foreground hover:text-foreground"}`}
							onClick={handleCopy}
						>
							{copied ? <Check size={14} /> : <Copy size={14} />}
						</Button>
						<Button
							variant="ghost"
							size="icon"
							className="h-8 w-8 text-muted-foreground hover:text-foreground"
							onClick={onClose}
						>
							<X size={18} />
						</Button>
					</div>
				</div>

				{/* Modal Content */}
				<div className="flex-1 overflow-auto p-6 bg-muted/30 custom-scrollbar text-left border-y border-border/20">
					<pre className="font-mono text-xs text-foreground leading-relaxed break-all whitespace-pre-wrap">
						{formattedValue}
					</pre>
				</div>

				{/* Footer */}
				<div className="px-6 py-3 border-t border-border bg-card text-[10px] uppercase font-bold tracking-widest text-muted-foreground/60">
					<span>Total Characters: {item.value.length}</span>
				</div>
			</motion.div>
		</motion.div>
	);
}

function ProviderIconCompact({
	provider,
}: {
	provider: "Git" | "Consul" | "Unknown";
}) {
	if (provider === "Git")
		return <GitBranch size={14} className="text-primary/70" />;
	if (provider === "Consul")
		return <Server size={14} className="text-primary/70" />;
	return <Cloud size={14} className="text-primary/70" />;
}
