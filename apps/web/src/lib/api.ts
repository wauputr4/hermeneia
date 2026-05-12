import { env } from '$env/dynamic/public';

const DEFAULT_API_BASE = 'http://127.0.0.1:19318';

export type ContentRun = {
	id: string;
	topic: string;
	content_type: string;
	template_id: string;
	created_at: string;
};

export type BriefVersion = {
	id: string;
	run_id: string;
	version: number;
	body: BriefBody;
	created_at: string;
};

export type BriefBody = {
	topic?: string;
	angle?: string;
	hook?: string;
	audience?: string;
	platform?: string;
	content_type?: string;
	tone?: string;
	key_points?: string[];
	visual_direction?: string;
	cta?: string;
	caption_draft?: string;
	hashtags?: string[];
	[key: string]: unknown;
};

export type RevisionEvent = {
	id: string;
	run_id: string;
	from_brief_version_id: string;
	to_brief_version_id: string;
	instruction: string;
	created_at: string;
};

export type Artifact = {
	id: string;
	run_id: string;
	brief_version_id?: string;
	kind: string;
	path: string;
	checksum?: string;
	created_at: string;
};

export type ScheduledPost = {
	id: string;
	run_id: string;
	artifact_id: string;
	platform: string;
	status: string;
	scheduled_at: string;
	created_at: string;
};

export type Template = {
	id: string;
	name: string;
	content_type: string;
	description: string;
	version: string;
	aspect_ratio: string;
	renderer: string;
	output_kinds: string[];
	input_schema: Record<string, unknown>;
	preview_asset?: string;
	assets?: string[];
};

export type WorkflowStep = {
	type: string;
	name?: string;
};

export type WorkflowPreset = {
	id: string;
	name: string;
	description: string;
	content_type: string;
	default_template_id: string;
	steps: WorkflowStep[];
	required_inputs: string[];
	revision_policy?: {
		mode?: string;
		max_revisions?: number;
	};
};

export type RunDetails = {
	run: ContentRun;
	briefs: BriefVersion[];
	revisions: RevisionEvent[];
	artifacts: Artifact[];
	scheduled_posts: ScheduledPost[];
};

export type CreateRunInput = {
	topic: string;
	content_type: string;
	template_id: string;
	tone: string;
	platform: string;
	target_audience: string;
};

function apiBase(): string {
	return (env.PUBLIC_HERMENEIA_API_BASE || DEFAULT_API_BASE).replace(/\/$/, '');
}

async function request<T>(path: string, init?: RequestInit): Promise<T> {
	const headers = new Headers(init?.headers);
	if (!headers.has('content-type')) {
		headers.set('content-type', 'application/json');
	}

	const response = await fetch(`${apiBase()}${path}`, {
		...init,
		headers
	});
	if (!response.ok) {
		let message = `${response.status} ${response.statusText}`;
		try {
			const body = (await response.json()) as { error?: string };
			message = body.error || message;
		} catch {
			// Keep the HTTP status text when the API returns no JSON body.
		}
		throw new Error(message);
	}
	if (response.status === 204) {
		throw new Error('API returned no JSON content');
	}
	return (await response.json()) as T;
}

export async function listRuns(): Promise<ContentRun[]> {
	const response = await request<{ runs: ContentRun[] }>('/v1/runs');
	return response.runs;
}

export async function listTemplates(): Promise<Template[]> {
	const response = await request<{ templates: Template[] }>('/v1/templates');
	return response.templates;
}

export async function listWorkflows(): Promise<WorkflowPreset[]> {
	const response = await request<{ workflows: WorkflowPreset[] }>('/v1/workflows');
	return response.workflows;
}

export function showRun(runID: string): Promise<RunDetails> {
	return request<RunDetails>(`/v1/runs/${encodeURIComponent(runID)}`);
}

export function createRun(input: CreateRunInput): Promise<{ run: ContentRun; brief: BriefVersion }> {
	return request('/v1/runs', {
		method: 'POST',
		body: JSON.stringify(input)
	});
}

export function reviseRun(runID: string, instruction: string): Promise<{ brief: BriefVersion }> {
	return request(`/v1/runs/${encodeURIComponent(runID)}/revisions`, {
		method: 'POST',
		body: JSON.stringify({ instruction })
	});
}

export function renderRun(runID: string): Promise<{ artifacts: Artifact[] }> {
	return request(`/v1/runs/${encodeURIComponent(runID)}/render`, {
		method: 'POST'
	});
}

export function artifactFileURL(artifact: Artifact): string {
	return `${apiBase()}/v1/runs/${encodeURIComponent(artifact.run_id)}/artifacts/${encodeURIComponent(artifact.id)}/file`;
}
