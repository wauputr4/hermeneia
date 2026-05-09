export const TEMPLATES = [
	{ id: 'carousel/ai-news-clean', label: 'AI news carousel', type: 'carousel' },
	{ id: 'video/ai-news-short', label: 'AI news short video', type: 'video' }
];

export function latestBrief(briefs) {
	return [...briefs].sort((a, b) => b.version - a.version)[0] ?? null;
}

export function artifactGroups(artifacts) {
	return artifacts.reduce((groups, artifact) => {
		const list = groups.get(artifact.kind) ?? [];
		list.push(artifact);
		groups.set(artifact.kind, list);
		return groups;
	}, new Map());
}

export function runSummary(run, details) {
	const latest = details ? latestBrief(details.briefs) : null;
	const artifactCount = details ? details.artifacts.length : 0;
	const revisionCount = details ? details.revisions.length : 0;
	return {
		id: run.id,
		topic: run.topic,
		type: run.content_type,
		template: run.template_id,
		latestVersion: latest?.version ?? 0,
		artifactCount,
		revisionCount
	};
}

export function templateForType(contentType) {
	return TEMPLATES.find((template) => template.type === contentType)?.id ?? TEMPLATES[0].id;
}

export function formatShortDate(value) {
	return new Intl.DateTimeFormat('en', {
		month: 'short',
		day: '2-digit',
		hour: '2-digit',
		minute: '2-digit'
	}).format(new Date(value));
}
