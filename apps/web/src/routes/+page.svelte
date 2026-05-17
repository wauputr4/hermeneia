<script lang="ts">
	import {
		artifactFileURL,
		createRun,
		listTemplates,
		listRuns,
		listWorkflows,
		renderRun,
		reviseRun,
		showRun,
		type BriefVersion,
		type ContentRun,
		type RunDetails,
		type Template,
		type WorkflowPreset
	} from '$lib/api';
	import {
		artifactDisplayName,
		artifactGroups,
		artifactKindLabel,
		artifactKindOptions,
		artifactPreviewType,
		artifactsForKind,
		createRunPayload,
		formatShortDate,
		latestBrief,
		runSummary,
		templateContentTypeLabel,
		templateForType,
		templateLabel,
		templatesForType,
		workflowLabel,
		workflowStepLabel,
		workflowTimeline,
		workflowsForType
	} from '$lib/view-models.js';
	import { onMount } from 'svelte';

	let runs = $state<ContentRun[]>([]);
	let templates = $state<Template[]>([]);
	let workflows = $state<WorkflowPreset[]>([]);
	let selectedRunID = $state('');
	let selectedDetails = $state<RunDetails | null>(null);
	let selectedBrief = $state<BriefVersion | null>(null);
	let loading = $state(true);
	let loadingTemplates = $state(true);
	let loadingWorkflows = $state(true);
	let busy = $state(false);
	let error = $state('');
	let templateError = $state('');
	let workflowError = $state('');
	let notice = $state('');
	let revisionInstruction = $state('');
	let artifactKindFilter = $state('all');
	let createForm = $state({
		workflow_id: '',
		topic: 'AI agents in marketing',
		content_type: 'carousel',
		template_id: '',
		tone: 'clear and practical',
		platform: 'instagram',
		target_audience: 'content operators'
	});

	$effect(() => {
		selectedBrief = selectedDetails ? latestBrief(selectedDetails.briefs) : null;
	});
	const artifactKindFilters = $derived(selectedDetails ? artifactKindOptions(selectedDetails.artifacts) : []);
	const visibleArtifacts = $derived(
		selectedDetails ? artifactsForKind(selectedDetails.artifacts, artifactKindFilter) : []
	);
	const groupedArtifacts = $derived(visibleArtifacts ? [...artifactGroups(visibleArtifacts).entries()] : []);
	const selectedTemplateOptions = $derived(
		templatesForType(templates, createForm.content_type)
	);
	const selectedWorkflowOptions = $derived(
		workflowsForType(workflows, createForm.content_type)
	);
	const selectedTemplate = $derived(
		templates.find((template) => template.id === createForm.template_id) ?? null
	);
	const selectedWorkflow = $derived(
		workflows.find((workflow) => workflow.id === createForm.workflow_id) ?? null
	);
	const activeSummary = $derived(
		selectedDetails && selectedDetails.run ? runSummary(selectedDetails.run, selectedDetails) : null
	);
	const selectedRunTimeline = $derived(workflowTimeline(selectedDetails));

	onMount(async () => {
		await Promise.all([loadTemplates(), loadWorkflows(), loadRuns()]);
	});

	async function loadTemplates() {
		loadingTemplates = true;
		templateError = '';
		try {
			templates = await listTemplates();
			const compatible = templatesForType(templates, createForm.content_type);
			if (!createForm.template_id || !compatible.some((template) => template.id === createForm.template_id)) {
				createForm.template_id = compatible[0]?.id ?? '';
			}
		} catch (err) {
			templates = [];
			createForm.template_id = '';
			templateError = err instanceof Error ? err.message : 'Unable to load template catalog';
		} finally {
			loadingTemplates = false;
		}
	}

	async function loadWorkflows() {
		loadingWorkflows = true;
		workflowError = '';
		try {
			workflows = await listWorkflows();
			const compatible = workflowsForType(workflows, createForm.content_type);
			if (!createForm.workflow_id || !compatible.some((workflow) => workflow.id === createForm.workflow_id)) {
				applyWorkflow(compatible[0] ?? null);
			}
		} catch (err) {
			workflows = [];
			createForm.workflow_id = '';
			workflowError = err instanceof Error ? err.message : 'Unable to load workflow catalog';
		} finally {
			loadingWorkflows = false;
		}
	}

	async function loadRuns() {
		loading = true;
		error = '';
		try {
			runs = await listRuns();
			if (runs.length > 0) {
				await selectRun(runs[0].id);
			}
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to load runs';
		} finally {
			loading = false;
		}
	}

	async function selectRun(runID: string) {
		selectedRunID = runID;
		error = '';
		try {
			selectedDetails = await showRun(runID);
			if (artifactKindFilter !== 'all' && !artifactKindOptions(selectedDetails.artifacts).includes(artifactKindFilter)) {
				artifactKindFilter = 'all';
			}
		} catch (err) {
			selectedDetails = null;
			error = err instanceof Error ? err.message : 'Unable to load run details';
		}
	}

	async function submitCreateRun() {
		if (!createForm.template_id) {
			templateError = 'Select a compatible template before creating a run.';
			return;
		}
		busy = true;
		error = '';
		notice = '';
		try {
			const result = await createRun(createRunPayload(createForm));
			notice = 'Run created';
			runs = await listRuns();
			await selectRun(result.run.id);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to create run';
		} finally {
			busy = false;
		}
	}

	async function submitRevision() {
		if (!selectedRunID || !revisionInstruction.trim()) return;
		busy = true;
		error = '';
		notice = '';
		try {
			await reviseRun(selectedRunID, revisionInstruction.trim());
			revisionInstruction = '';
			notice = 'Revision saved';
			await selectRun(selectedRunID);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to revise run';
		} finally {
			busy = false;
		}
	}

	async function submitRender() {
		if (!selectedRunID) return;
		busy = true;
		error = '';
		notice = '';
		try {
			await renderRun(selectedRunID);
			notice = 'Render job completed';
			await selectRun(selectedRunID);
		} catch (err) {
			error = err instanceof Error ? err.message : 'Unable to render run';
		} finally {
			busy = false;
		}
	}

	function changeContentType(value: string) {
		createForm.content_type = value;
		const workflow = workflowsForType(workflows, value)[0] ?? null;
		if (workflow) {
			applyWorkflow(workflow);
			return;
		}
		createForm.workflow_id = '';
		createForm.template_id = templateForType(templates, value);
	}

	function changeWorkflow(workflowID: string) {
		applyWorkflow(workflows.find((workflow) => workflow.id === workflowID) ?? null);
	}

	function applyWorkflow(workflow: WorkflowPreset | null) {
		createForm.workflow_id = workflow?.id ?? '';
		if (workflow) {
			createForm.content_type = workflow.content_type;
			createForm.template_id = workflow.default_template_id;
			return;
		}
		createForm.template_id = templateForType(templates, createForm.content_type);
	}
</script>

<svelte:head>
	<title>Hermeneia Studio</title>
	<meta
		name="description"
		content="Local Hermeneia web UI for reviewing content runs, briefs, revisions, and artifacts."
	/>
</svelte:head>

<main class="shell">
	<section class="topbar">
		<div>
			<p class="eyebrow">Hermeneia Studio</p>
			<h1>Content runs, briefs, and exports in one local control room.</h1>
		</div>
		<div class="signal">
			<span>{runs.length}</span>
			<small>runs</small>
		</div>
	</section>

	{#if error}
		<p class="banner error">{error}</p>
	{/if}
	{#if notice}
		<p class="banner notice">{notice}</p>
	{/if}

	<section class="workspace">
		<aside class="sidebar" aria-label="Content runs">
			<div class="panel-head">
				<h2>Runs</h2>
				<button type="button" class="ghost" onclick={loadRuns} disabled={busy || loading}>Refresh</button>
			</div>
			{#if loading}
				<p class="muted">Loading local API data...</p>
			{:else if runs.length === 0}
				<p class="muted">No content runs yet. Create one to start the review loop.</p>
			{:else}
				<div class="run-list">
					{#each runs as run}
						<button
							type="button"
							class:active={run.id === selectedRunID}
							onclick={() => selectRun(run.id)}
						>
							<strong>{run.topic}</strong>
							<span>{run.content_type} / {formatShortDate(run.created_at)}</span>
						</button>
					{/each}
				</div>
			{/if}
		</aside>

		<section class="detail">
			{#if selectedDetails && activeSummary}
				<div class="detail-head">
					<div>
						<p class="eyebrow">{activeSummary.type} / {activeSummary.template}</p>
						<h2>{activeSummary.topic}</h2>
						<p class="muted mono">{activeSummary.id}</p>
					</div>
					<div class="stats">
						<div><strong>v{activeSummary.latestVersion}</strong><span>brief</span></div>
						<div><strong>{activeSummary.revisionCount}</strong><span>revisions</span></div>
						<div><strong>{activeSummary.artifactCount}</strong><span>artifacts</span></div>
					</div>
				</div>

				<div class="detail-grid">
					<section class="brief">
						<div class="panel-head">
							<h3>Brief Editor</h3>
							<select bind:value={selectedBrief}>
								{#each selectedDetails.briefs as brief}
									<option value={brief}>Version {brief.version}</option>
								{/each}
							</select>
						</div>
						{#if selectedBrief}
							<div class="brief-body">
								<label>
									Hook
									<textarea readonly value={selectedBrief.body.hook ?? ''}></textarea>
								</label>
								<label>
									Angle
									<textarea readonly value={selectedBrief.body.angle ?? ''}></textarea>
								</label>
								<label>
									Caption
									<textarea readonly value={selectedBrief.body.caption_draft ?? ''}></textarea>
								</label>
								<div class="chips">
									{#each selectedBrief.body.hashtags ?? [] as hashtag}
										<span>{hashtag}</span>
									{/each}
								</div>
							</div>
						{/if}
					</section>

					<section class="actions">
						<h3>Operations</h3>
						<form
							onsubmit={(event) => {
								event.preventDefault();
								submitRevision();
							}}
						>
							<label>
								Revision instruction
								<textarea bind:value={revisionInstruction} placeholder="Make the hook sharper"></textarea>
							</label>
							<button type="submit" disabled={busy || !revisionInstruction.trim()}>Save revision</button>
						</form>
						<button type="button" class="primary" onclick={submitRender} disabled={busy}>Render export</button>
					</section>
				</div>

					<section class="lower-grid">
						<div>
							<div class="panel-head">
								<h3>Artifacts</h3>
								<select class="compact-select" bind:value={artifactKindFilter} aria-label="Filter artifacts by kind">
									<option value="all">All kinds</option>
									{#each artifactKindFilters as kind}
										<option value={kind}>{artifactKindLabel(kind)}</option>
									{/each}
								</select>
							</div>
							{#if selectedDetails.artifacts.length === 0}
								<p class="muted">No artifacts recorded yet.</p>
							{:else if groupedArtifacts.length === 0}
								<p class="muted">No artifacts match this filter.</p>
							{:else}
								{#each groupedArtifacts as [kind, artifacts]}
									<div class="artifact-group">
										<h4>{artifactKindLabel(kind)} <span>{artifacts.length}</span></h4>
										{#each artifacts as artifact}
											{@const previewType = artifactPreviewType(artifact)}
											{@const fileURL = artifactFileURL(artifact)}
											<div class:media-artifact={previewType} class="artifact-card">
												{#if previewType === 'image'}
													<img src={fileURL} alt={artifact.path} loading="lazy" />
												{:else if previewType === 'video'}
													<video src={fileURL} controls muted playsinline></video>
												{/if}
												<div class="artifact-meta">
													<strong>{artifactDisplayName(artifact)}</strong>
													<span>{artifact.path}</span>
													<small>{formatShortDate(artifact.created_at)} / {artifact.checksum || 'no checksum'}</small>
													<div class="artifact-links">
														<a href={fileURL} target="_blank" rel="noreferrer">Open</a>
														<a href={fileURL} download={artifactDisplayName(artifact)}>Download</a>
													</div>
												</div>
											</div>
										{/each}
									</div>
								{/each}
							{/if}
						</div>
					<div>
						<h3>Step Timeline</h3>
						{#if selectedRunTimeline.length === 0}
							<p class="muted">No timeline events yet.</p>
						{:else}
							<ol class="step-timeline">
								{#each selectedRunTimeline as step}
									<li class={step.status}>
										<span>{step.label}</span>
										<strong>{step.detail}</strong>
										{#if step.at}
											<small>{formatShortDate(step.at)}</small>
										{/if}
									</li>
								{/each}
							</ol>
						{/if}
					</div>
				</section>
			{:else}
				<div class="empty">
					<h2>Connect to the local Hermeneia API.</h2>
					<p>Start the Go server with <code>hermeneia serve --addr 127.0.0.1:19318</code>.</p>
				</div>
			{/if}
		</section>

		<aside class="creator" aria-label="Create run">
			<h2>New Run</h2>
			<form
				onsubmit={(event) => {
					event.preventDefault();
					submitCreateRun();
				}}
			>
				<label>
					Workflow
					<select value={createForm.workflow_id} onchange={(event) => changeWorkflow(event.currentTarget.value)} disabled={loadingWorkflows || selectedWorkflowOptions.length === 0}>
						{#if loadingWorkflows}
							<option value="">Loading workflows...</option>
						{:else if selectedWorkflowOptions.length === 0}
							<option value="">No compatible workflows</option>
						{:else}
							<option value="">Manual run</option>
							{#each selectedWorkflowOptions as workflow}
								<option value={workflow.id}>{workflowLabel(workflow)}</option>
							{/each}
						{/if}
					</select>
				</label>
				{#if workflowError}
					<p class="field-note error-text">{workflowError}</p>
				{:else if selectedWorkflow}
					<div class="workflow-card">
						<strong>{workflowLabel(selectedWorkflow)}</strong>
						<p>{selectedWorkflow.description}</p>
						<div class="workflow-steps">
							{#each selectedWorkflow.steps as step, index}
								<span>{index + 1}. {workflowStepLabel(step)}</span>
							{/each}
						</div>
						<small>Needs: {selectedWorkflow.required_inputs.join(', ')}</small>
					</div>
				{/if}
				<label>
					Topic
					<input bind:value={createForm.topic} required />
				</label>
				<label>
					Content type
					<select value={createForm.content_type} onchange={(event) => changeContentType(event.currentTarget.value)}>
						<option value="carousel">Carousel</option>
						<option value="short_video">Short video</option>
					</select>
				</label>
				<label>
					Template
					<select bind:value={createForm.template_id} disabled={loadingTemplates || selectedTemplateOptions.length === 0}>
						{#if loadingTemplates}
							<option value="">Loading templates...</option>
						{:else if selectedTemplateOptions.length === 0}
							<option value="">No compatible templates</option>
						{:else}
							{#each selectedTemplateOptions as template}
								<option value={template.id}>{templateLabel(template)}</option>
							{/each}
						{/if}
					</select>
				</label>
				{#if templateError}
					<p class="field-note error-text">{templateError}</p>
				{:else if selectedTemplate}
					<div class="template-card">
						<strong>{selectedTemplate.name}</strong>
						<span>{templateContentTypeLabel(selectedTemplate.content_type)} / {selectedTemplate.aspect_ratio} / {selectedTemplate.renderer}</span>
						<p>{selectedTemplate.description}</p>
						<small>{selectedTemplate.output_kinds?.join(', ') ?? ''}</small>
					</div>
				{:else if !loadingTemplates}
					<p class="field-note">No template is available for {templateContentTypeLabel(createForm.content_type)}.</p>
				{/if}
				<label>
					Platform
					<input bind:value={createForm.platform} />
				</label>
				<label>
					Audience
					<input bind:value={createForm.target_audience} />
				</label>
				<label>
					Tone
					<input bind:value={createForm.tone} />
				</label>
				<button type="submit" disabled={busy || !createForm.topic.trim() || !createForm.template_id}>Create run</button>
			</form>
		</aside>
	</section>
</main>

<style>
	:global(body) {
		margin: 0;
		background: #f2efe7;
		color: #1d241f;
		font-family: Georgia, 'Times New Roman', serif;
	}

	:global(*) {
		box-sizing: border-box;
	}

	button,
	input,
	select,
	textarea {
		font: inherit;
	}

	.shell {
		min-height: 100vh;
		padding: 24px;
		background:
			linear-gradient(90deg, rgba(28, 35, 31, 0.06) 1px, transparent 1px),
			linear-gradient(180deg, rgba(28, 35, 31, 0.05) 1px, transparent 1px),
			#f2efe7;
		background-size: 28px 28px;
	}

	.topbar,
	.workspace,
	.detail-grid,
	.lower-grid {
		display: grid;
		gap: 16px;
	}

	.topbar {
		grid-template-columns: 1fr auto;
		align-items: end;
		margin-bottom: 18px;
	}

	h1,
	h2,
	h3,
	h4,
	p {
		margin: 0;
	}

	h1 {
		max-width: 860px;
		font-size: clamp(2rem, 4vw, 4.8rem);
		line-height: 0.95;
		font-weight: 700;
	}

	h2 {
		font-size: 1.45rem;
	}

	h3 {
		font-size: 1rem;
		text-transform: uppercase;
	}

	.eyebrow,
	.muted,
	small,
	.mono {
		font-family: 'Courier New', monospace;
	}

	.eyebrow {
		margin-bottom: 8px;
		color: #8b2d1e;
		font-size: 0.78rem;
		text-transform: uppercase;
	}

	.muted {
		color: #657166;
		font-size: 0.88rem;
	}

	.signal {
		width: 120px;
		aspect-ratio: 1;
		display: grid;
		place-items: center;
		border: 2px solid #1d241f;
		background: #d9e078;
		box-shadow: 8px 8px 0 #1d241f;
	}

	.signal span {
		font-size: 2.5rem;
		font-weight: 700;
	}

	.signal small {
		margin-top: -26px;
		text-transform: uppercase;
	}

	.workspace {
		grid-template-columns: minmax(220px, 280px) minmax(0, 1fr) minmax(240px, 320px);
		align-items: start;
	}

	.sidebar,
	.detail,
	.creator,
	.brief,
	.actions,
	.lower-grid > div,
	.empty {
		border: 2px solid #1d241f;
		background: rgba(255, 252, 241, 0.92);
		box-shadow: 5px 5px 0 rgba(29, 36, 31, 0.9);
	}

	.sidebar,
	.creator,
	.detail,
	.brief,
	.actions,
	.lower-grid > div,
	.empty {
		padding: 18px;
	}

	.panel-head,
	.detail-head,
	.stats {
		display: flex;
		gap: 12px;
		align-items: center;
		justify-content: space-between;
	}

	.run-list {
		display: grid;
		gap: 10px;
		margin-top: 14px;
	}

	.run-list button {
		width: 100%;
		padding: 12px;
		border: 1px solid #1d241f;
		background: #fffaf0;
		text-align: left;
		cursor: pointer;
	}

	.run-list button.active,
	.run-list button:hover {
		background: #203d35;
		color: #fffaf0;
	}

	.run-list span {
		display: block;
		margin-top: 5px;
		font-family: 'Courier New', monospace;
		font-size: 0.72rem;
	}

	.detail {
		min-height: 620px;
	}

	.detail-head {
		align-items: start;
		border-bottom: 2px solid #1d241f;
		padding-bottom: 18px;
	}

	.stats div {
		min-width: 82px;
		padding: 10px;
		border-left: 2px solid #1d241f;
		text-align: center;
	}

	.stats strong {
		display: block;
		font-size: 1.6rem;
	}

	.stats span {
		font-family: 'Courier New', monospace;
		font-size: 0.72rem;
		text-transform: uppercase;
	}

	.detail-grid {
		grid-template-columns: minmax(0, 1fr) minmax(220px, 280px);
		margin-top: 18px;
	}

	.lower-grid {
		grid-template-columns: 1fr 1fr;
		margin-top: 16px;
	}

	form,
	.brief-body {
		display: grid;
		gap: 12px;
	}

	label {
		display: grid;
		gap: 6px;
		font-family: 'Courier New', monospace;
		font-size: 0.76rem;
		text-transform: uppercase;
	}

	input,
	select,
	textarea {
		width: 100%;
		border: 1px solid #1d241f;
		border-radius: 0;
		background: #fffaf0;
		color: #1d241f;
		padding: 10px;
	}

	textarea {
		min-height: 82px;
		resize: vertical;
		text-transform: none;
	}

	button {
		border: 2px solid #1d241f;
		background: #d9e078;
		color: #1d241f;
		padding: 10px 14px;
		cursor: pointer;
		font-family: 'Courier New', monospace;
		text-transform: uppercase;
	}

	button.primary {
		width: 100%;
		margin-top: 12px;
		background: #f06d3f;
	}

	button.ghost {
		background: transparent;
	}

	button:disabled {
		cursor: not-allowed;
		opacity: 0.5;
	}

	.chips {
		display: flex;
		flex-wrap: wrap;
		gap: 8px;
	}

	.chips span {
		border: 1px solid #1d241f;
		background: #d9e078;
		padding: 5px 8px;
		font-family: 'Courier New', monospace;
		font-size: 0.75rem;
	}

		.artifact-group {
			margin-top: 14px;
			border-top: 1px solid #1d241f;
			padding-top: 10px;
		}

		.artifact-group h4 {
			display: flex;
			align-items: center;
			justify-content: space-between;
			gap: 10px;
			font-family: 'Courier New', monospace;
			font-size: 0.78rem;
			text-transform: uppercase;
		}

		.artifact-group h4 span {
			border: 1px solid #1d241f;
			background: #d9e078;
			padding: 2px 6px;
		}

		.compact-select {
			max-width: 170px;
			padding: 7px 9px;
			font-family: 'Courier New', monospace;
			font-size: 0.75rem;
			text-transform: uppercase;
		}

		.artifact-card {
			display: grid;
			gap: 8px;
			margin-top: 10px;
			border: 1px solid #1d241f;
			background: #fffaf0;
			padding: 10px;
		}

		.artifact-card.media-artifact {
			grid-template-columns: minmax(92px, 160px) minmax(0, 1fr);
			align-items: start;
		}

		.artifact-card img,
		.artifact-card video {
			width: 100%;
			border: 1px solid #1d241f;
			background: #1d241f;
		}

	.artifact-card img {
		aspect-ratio: 4 / 5;
		object-fit: cover;
	}

		.artifact-card video {
			aspect-ratio: 9 / 16;
		}

		.artifact-meta {
			display: grid;
			gap: 6px;
			font-family: 'Courier New', monospace;
			font-size: 0.75rem;
			word-break: break-word;
		}

		.artifact-meta strong {
			font-family: Georgia, 'Times New Roman', serif;
			font-size: 0.98rem;
			word-break: break-word;
		}

		.artifact-meta span,
		.artifact-meta small {
			color: #657166;
		}

		.artifact-links {
			display: flex;
			flex-wrap: wrap;
			gap: 8px;
		}

		.artifact-links a {
			border: 1px solid #1d241f;
			background: #d9e078;
			padding: 5px 8px;
			color: #1d241f;
			text-decoration: none;
			text-transform: uppercase;
		}

		.artifact-group small {
			color: #8b2d1e;
		}

	.field-note,
	.template-card,
	.workflow-card {
		font-family: 'Courier New', monospace;
		font-size: 0.76rem;
	}

	.error-text {
		color: #8b2d1e;
	}

	.template-card,
	.workflow-card {
		display: grid;
		gap: 6px;
		border: 1px solid #1d241f;
		background: #fffaf0;
		padding: 10px;
	}

	.workflow-card {
		border-width: 2px;
		background: #e8f0c6;
	}

	.template-card strong,
	.template-card span,
	.template-card small,
	.workflow-card strong,
	.workflow-card small {
		display: block;
	}

	.template-card span,
	.template-card small,
	.workflow-card small {
		color: #657166;
	}

	.workflow-steps {
		display: grid;
		gap: 5px;
	}

	.workflow-steps span {
		border-left: 3px solid #1d241f;
		padding-left: 8px;
	}

	.step-timeline {
		display: grid;
		gap: 10px;
		margin: 12px 0 0;
		padding: 0;
		list-style: none;
	}

	.step-timeline li {
		position: relative;
		display: grid;
		gap: 4px;
		border: 1px solid #1d241f;
		background: #fffaf0;
		padding: 10px 10px 10px 42px;
		font-family: 'Courier New', monospace;
		font-size: 0.78rem;
	}

	.step-timeline li::before {
		content: '';
		position: absolute;
		left: 12px;
		top: 12px;
		width: 14px;
		aspect-ratio: 1;
		border: 2px solid #1d241f;
		background: #fffaf0;
	}

	.step-timeline li.done::before {
		background: #d9e078;
	}

	.step-timeline li.pending {
		color: #657166;
	}

	.step-timeline span {
		text-transform: uppercase;
	}

	.banner {
		margin: 0 0 14px;
		border: 2px solid #1d241f;
		padding: 10px 14px;
		font-family: 'Courier New', monospace;
	}

	.banner.error {
		background: #ffd7c8;
	}

	.banner.notice {
		background: #d9e078;
	}

	.empty {
		display: grid;
		min-height: 420px;
		place-content: center;
		gap: 12px;
		text-align: center;
	}

	code {
		background: #1d241f;
		color: #fffaf0;
		padding: 2px 5px;
	}

	@media (max-width: 1080px) {
		.workspace,
		.detail-grid,
		.lower-grid {
			grid-template-columns: 1fr;
		}

		.sidebar,
		.creator {
			position: static;
		}
	}

	@media (max-width: 680px) {
		.shell {
			padding: 14px;
		}

		.topbar,
		.detail-head,
		.stats {
			grid-template-columns: 1fr;
			display: grid;
		}

		.signal {
			width: 88px;
		}
	}
</style>
