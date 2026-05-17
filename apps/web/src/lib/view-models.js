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

export function artifactPreviewType(artifact) {
	const path = artifact.path?.toLowerCase() ?? '';
	if (artifact.kind === 'carousel_png' || path.endsWith('.png')) {
		return 'image';
	}
	if (artifact.kind === 'video_mp4' || path.endsWith('.mp4')) {
		return 'video';
	}
	return null;
}

export function artifactKindLabel(kind) {
	return kind.replaceAll('_', ' ');
}

export function artifactKindOptions(artifacts) {
	return [...new Set(artifacts.map((artifact) => artifact.kind))].sort();
}

export function artifactsForKind(artifacts, kind) {
	if (!kind || kind === 'all') {
		return artifacts;
	}
	return artifacts.filter((artifact) => artifact.kind === kind);
}

export function artifactDisplayName(artifact) {
	return artifact.path?.split(/[\\/]/)?.at(-1) || artifact.id;
}

export function auditStatusLabel(audit) {
	if (!audit) {
		return 'Not checked';
	}
	return audit.healthy ? 'Healthy' : `${audit.issues?.length ?? 0} issue${audit.issues?.length === 1 ? '' : 's'}`;
}

export function auditIssueRows(audit) {
	return (audit?.issues ?? []).map((issue) => ({
		kind: artifactKindLabel(issue.kind || 'unknown'),
		artifactID: issue.artifact_id || 'n/a',
		path: issue.path || 'n/a',
		message: issue.message || 'No message returned'
	}));
}

export function scheduleArtifactOptions(artifacts) {
	return (artifacts ?? [])
		.filter((artifact) => artifact.id && artifact.kind !== 'research_json')
		.map((artifact) => ({
			id: artifact.id,
			label: `${artifactKindLabel(artifact.kind)} / ${artifactDisplayName(artifact)}`
		}));
}

export function schedulePostPayload(form) {
	return {
		artifact_id: form.artifact_id,
		platform: form.platform,
		scheduled_at: new Date(form.scheduled_at).toISOString()
	};
}

export function defaultScheduleDateTime(now = new Date()) {
	const nextHour = new Date(now.getTime() + 60 * 60 * 1000);
	const local = new Date(nextHour.getTime() - nextHour.getTimezoneOffset() * 60 * 1000);
	return local.toISOString().slice(0, 16);
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

export function templatesForType(templates, contentType) {
	return templates
		.filter((template) => template.content_type === contentType)
		.sort((a, b) => templateLabel(a).localeCompare(templateLabel(b)) || a.id.localeCompare(b.id));
}

export function templateForType(templates, contentType) {
	return templatesForType(templates, contentType)[0]?.id ?? '';
}

export function templateLabel(template) {
	return template.name || template.id;
}

export function templateContentTypeLabel(contentType) {
	switch (contentType) {
		case 'carousel':
			return 'Carousel';
		case 'short_video':
			return 'Short video';
		default:
			return contentType;
	}
}

export function workflowLabel(workflow) {
	return workflow.name || workflow.id;
}

export function workflowsForType(workflows, contentType) {
	return workflows
		.filter((workflow) => workflow.content_type === contentType)
		.sort((a, b) => workflowLabel(a).localeCompare(workflowLabel(b)) || a.id.localeCompare(b.id));
}

export function workflowForType(workflows, contentType) {
	return workflowsForType(workflows, contentType)[0]?.id ?? '';
}

export function workflowStepLabel(step) {
	if (step.name) {
		return step.name;
	}
	switch (step.type) {
		case 'research_plan':
			return 'Research plan';
		case 'create_brief':
			return 'Create brief';
		case 'revise_brief':
			return 'Revise brief';
		case 'render':
			return 'Render';
		case 'schedule_record':
			return 'Schedule';
		default:
			return step.type;
	}
}

export function createRunPayload(form) {
	const base = {
		topic: form.topic,
		tone: form.tone,
		platform: form.platform,
		target_audience: form.target_audience
	};
	if (form.workflow_id) {
		return {
			...base,
			workflow_id: form.workflow_id
		};
	}
	return {
		...base,
		content_type: form.content_type,
		template_id: form.template_id
	};
}

function timestampValue(value) {
	const timestamp = new Date(value ?? '').getTime();
	return Number.isNaN(timestamp) ? 0 : timestamp;
}

function compareTimestamp(field) {
	return (a, b) => timestampValue(a?.[field]) - timestampValue(b?.[field]);
}

export function workflowTimeline(details) {
	if (!details) {
		return [];
	}
	const artifacts = details.artifacts ?? [];
	const revisions = [...(details.revisions ?? [])].sort(compareTimestamp('created_at'));
	const schedules = [...(details.scheduled_posts ?? [])].sort(compareTimestamp('scheduled_at'));
	const hasResearch = artifacts.some((artifact) => artifact.kind === 'research_json');
	const renderedArtifacts = artifacts
		.filter((artifact) => artifact.kind !== 'research_json')
		.sort(compareTimestamp('created_at'));
	const sortedBriefs = [...(details.briefs ?? [])].sort((a, b) => a.version - b.version);
	const latest = sortedBriefs.at(-1) ?? null;
	return [
		{
			key: 'research',
			label: 'Research',
			status: hasResearch ? 'done' : 'pending',
			detail: hasResearch ? 'Research artifact recorded' : 'No research artifact yet'
		},
		{
			key: 'brief',
			label: 'Brief',
			status: latest ? 'done' : 'pending',
			detail: latest ? `Latest brief v${latest.version}` : 'No brief version yet',
			at: latest?.created_at
		},
		{
			key: 'revision',
			label: 'Revision',
			status: revisions.length > 0 ? 'done' : 'pending',
			detail: revisions.length > 0 ? `${revisions.length} revision${revisions.length === 1 ? '' : 's'} saved` : 'No revision saved',
			at: revisions.at(-1)?.created_at
		},
		{
			key: 'render',
			label: 'Render',
			status: renderedArtifacts.length > 0 ? 'done' : 'pending',
			detail:
				renderedArtifacts.length > 0
					? `${renderedArtifacts.length} export artifact${renderedArtifacts.length === 1 ? '' : 's'} recorded`
					: 'No render artifacts yet',
			at: renderedArtifacts.at(-1)?.created_at
		},
		{
			key: 'schedule',
			label: 'Schedule',
			status: schedules.length > 0 ? 'done' : 'pending',
			detail: schedules.length > 0 ? `${schedules.length} scheduled post${schedules.length === 1 ? '' : 's'}` : 'Not scheduled',
			at: schedules.at(-1)?.scheduled_at
		}
	];
}

const shortDateFormatter = new Intl.DateTimeFormat('en', {
	month: 'short',
	day: '2-digit',
	hour: '2-digit',
	minute: '2-digit'
});

export function formatShortDate(value) {
	const date = new Date(value);
	if (Number.isNaN(date.getTime())) {
		return 'n/a';
	}
	return shortDateFormatter.format(date);
}
