import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import { scheduledPostsPath } from '../src/lib/api-paths.js';
import {
	artifactDisplayName,
	artifactGroups,
	artifactKindLabel,
	artifactKindOptions,
	artifactPreviewType,
	artifactsForKind,
	auditIssueRows,
	auditStatusLabel,
	createRunPayload,
	defaultScheduleDateTime,
	formatShortDate,
	latestBrief,
	runSummary,
	filteredSchedulePosts,
	scheduleAgendaEmptyMessage,
	scheduleAgendaFilterOptions,
	scheduleAgendaGroups,
	scheduleAgendaQueryFilters,
	scheduleAgendaRows,
	scheduleArtifactOptions,
	schedulePostPayload,
	scheduleValidationSummary,
	templateContentTypeLabel,
	templateForType,
	templateLabel,
	templatesForType,
	workflowForType,
	workflowLabel,
	workflowStepLabel,
	workflowTimeline,
	workflowsForType
} from '../src/lib/view-models.js';

describe('web view model helpers', () => {
	it('selects the highest brief version', () => {
		const brief = latestBrief([
			{ id: 'brief-1', version: 1 },
			{ id: 'brief-3', version: 3 },
			{ id: 'brief-2', version: 2 }
		]);

		assert.equal(brief.id, 'brief-3');
	});

	it('groups artifacts by kind without losing order', () => {
		const groups = artifactGroups([
			{ kind: 'slide_png', path: 'slide-01.png' },
			{ kind: 'caption', path: 'caption.txt' },
			{ kind: 'slide_png', path: 'slide-02.png' }
		]);

		assert.deepEqual(
			[...groups.entries()].map(([kind, artifacts]) => [kind, artifacts.map((artifact) => artifact.path)]),
			[
				['slide_png', ['slide-01.png', 'slide-02.png']],
				['caption', ['caption.txt']]
			]
		);
	});

	it('detects previewable artifact media types', () => {
		assert.equal(artifactPreviewType({ kind: 'carousel_png', path: 'runs/run-1/output/slide-01.png' }), 'image');
		assert.equal(artifactPreviewType({ kind: 'video_mp4', path: 'runs/run-1/output/video.mp4' }), 'video');
		assert.equal(artifactPreviewType({ kind: 'content_json', path: 'runs/run-1/content.json' }), null);
	});

	it('builds artifact filter options and filtered lists', () => {
		const artifacts = [
			{ id: 'artifact-1', kind: 'caption_text', path: 'runs/run-1/output/caption.txt' },
			{ id: 'artifact-2', kind: 'carousel_png', path: 'runs/run-1/output/slide-01.png' },
			{ id: 'artifact-3', kind: 'caption_text', path: 'runs/run-1/output/caption-2.txt' }
		];

		assert.deepEqual(artifactKindOptions(artifacts), ['caption_text', 'carousel_png']);
		assert.deepEqual(
			artifactsForKind(artifacts, 'caption_text').map((artifact) => artifact.id),
			['artifact-1', 'artifact-3']
		);
		assert.equal(artifactsForKind(artifacts, 'all').length, 3);
		assert.equal(artifactKindLabel('content_json'), 'content json');
		assert.equal(artifactDisplayName(artifacts[1]), 'slide-01.png');
		assert.equal(artifactDisplayName({ id: 'artifact-4', kind: 'content_json' }), 'artifact-4');
	});

	it('formats artifact audit status and issue rows', () => {
		assert.equal(auditStatusLabel(null), 'Not checked');
		assert.equal(auditStatusLabel({ healthy: true, issues: [] }), 'Healthy');
		assert.equal(auditStatusLabel({ healthy: false, issues: [{ kind: 'missing_file' }, { kind: 'checksum_mismatch' }] }), '2 issues');
		assert.deepEqual(
			auditIssueRows({
				healthy: false,
				issues: [
					{
						kind: 'missing_file',
						artifact_id: 'artifact-1',
						path: 'runs/run-1/output/slide.png',
						message: 'artifact file is missing'
					},
					{
						kind: 'untracked_file',
						path: 'runs/run-1/output/extra.png',
						message: ''
					}
				]
			}),
			[
				{
					kind: 'missing file',
					artifactID: 'artifact-1',
					path: 'runs/run-1/output/slide.png',
					message: 'artifact file is missing'
				},
				{
					kind: 'untracked file',
					artifactID: 'n/a',
					path: 'runs/run-1/output/extra.png',
					message: 'No message returned'
				}
			]
		);
	});

	it('builds schedule artifact options without research artifacts', () => {
		assert.deepEqual(
			scheduleArtifactOptions([
				{ id: 'research-1', kind: 'research_json', path: 'runs/run-1/research.json' },
				{ id: 'slide-1', kind: 'carousel_png', path: 'runs/run-1/output/carousel/slide-01.png' },
				{ id: 'caption-1', kind: 'caption_text', path: 'runs/run-1/output/caption.txt' },
				{ kind: 'content_json', path: 'runs/run-1/output/content.json' }
			]),
			[
				{ id: 'slide-1', label: 'carousel png / slide-01.png' },
				{ id: 'caption-1', label: 'caption text / caption.txt' }
			]
		);
	});

	it('builds schedule payloads with RFC3339 timestamps', () => {
		assert.deepEqual(
			schedulePostPayload({
				artifact_id: 'artifact-1',
				platform: 'instagram',
				scheduled_at: '2026-05-18T09:30'
			}),
			{
				artifact_id: 'artifact-1',
				platform: 'instagram',
				scheduled_at: '2026-05-18T09:30:00.000Z'
			}
		);
		assert.equal(defaultScheduleDateTime(new Date('2026-05-18T08:30:00Z')).length, 16);
	});

	it('builds sorted scheduled-post agenda rows with run topics', () => {
		assert.deepEqual(
			scheduleAgendaRows(
				[
					{
						id: 'schedule-2',
						run_id: 'run-2',
						artifact_id: '',
						platform: 'linkedin',
						status: 'scheduled',
						scheduled_at: '2026-05-18T10:00:00Z',
						validation: {
							credential_storage: 'external_only',
							credentials_stored_in_db: false,
							requires_platform_connector: true,
							artifact_selected: false,
							warning: 'No rendered artifact was selected for the scheduled post.'
						}
					},
					{
						id: 'schedule-1',
						run_id: 'run-1',
						artifact_id: 'artifact-1',
						platform: 'instagram',
						status: 'scheduled',
						scheduled_at: '2026-05-18T09:00:00Z'
					}
				],
				[{ id: 'run-1', topic: 'AI launch' }]
			),
			[
				{
					id: 'schedule-1',
					runID: 'run-1',
					topic: 'AI launch',
					platform: 'instagram',
					status: 'scheduled',
					artifactID: 'artifact-1',
					scheduledAt: '2026-05-18T09:00:00Z',
					validation: {
						hasMetadata: false,
						warning: '',
						details: []
					},
					cancellable: true
				},
				{
					id: 'schedule-2',
					runID: 'run-2',
					topic: 'run-2',
					platform: 'linkedin',
					status: 'scheduled',
					artifactID: 'none',
					scheduledAt: '2026-05-18T10:00:00Z',
					validation: {
						hasMetadata: true,
						warning: 'No rendered artifact was selected for the scheduled post.',
						details: ['external only', 'no credentials in DB', 'connector required', 'no artifact selected']
					},
					cancellable: true
				}
			]
		);
	});

	it('summarizes scheduled-post validation metadata defensively', () => {
		assert.deepEqual(scheduleValidationSummary(null), { hasMetadata: false, warning: '', details: [] });
		assert.deepEqual(scheduleValidationSummary('not-json-object'), { hasMetadata: false, warning: '', details: [] });
		assert.deepEqual(
			scheduleValidationSummary({
				credential_storage: 'external_only',
				credentials_stored_in_db: false,
				requires_platform_connector: true,
				artifact_selected: true
			}),
			{
				hasMetadata: true,
				warning: '',
				details: ['external only', 'no credentials in DB', 'connector required', 'artifact selected']
			}
		);
	});

	it('only marks scheduled agenda rows as cancellable', () => {
		assert.deepEqual(
			scheduleAgendaRows([
				{ id: 'schedule-1', run_id: 'run-1', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' },
				{ id: 'schedule-2', run_id: 'run-1', status: 'cancelled', scheduled_at: '2026-05-18T10:00:00Z' }
			]).map((row) => [row.id, row.cancellable]),
			[
				['schedule-1', true],
				['schedule-2', false]
			]
		);
	});

	it('filters scheduled-post agenda rows by status and platform', () => {
		const posts = [
			{ id: 'schedule-3', run_id: 'run-1', platform: 'linkedin', status: 'cancelled', scheduled_at: '2026-05-18T11:00:00Z' },
			{ id: 'schedule-1', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' },
			{ id: 'schedule-2', run_id: 'run-2', platform: 'linkedin', status: 'scheduled', scheduled_at: '2026-05-18T10:00:00Z' }
		];

		assert.deepEqual(scheduleAgendaFilterOptions(posts, 'status'), ['cancelled', 'scheduled']);
		assert.deepEqual(scheduleAgendaFilterOptions(posts, 'platform'), ['instagram', 'linkedin']);
		assert.deepEqual(
			filteredSchedulePosts(posts, { status: 'scheduled', platform: 'linkedin' }).map((post) => post.id),
			['schedule-2']
		);
		assert.deepEqual(
			scheduleAgendaRows(posts, [], { status: 'scheduled', platform: 'all' }).map((row) => row.id),
			['schedule-1', 'schedule-2']
		);
		assert.deepEqual(
			scheduleAgendaRows(posts, [], { runID: 'run-1', status: 'all', platform: 'all' }).map((row) => row.id),
			['schedule-1', 'schedule-3']
		);
		assert.equal(
			scheduleAgendaEmptyMessage(
				{ runID: 'run-2', status: 'scheduled', platform: 'instagram' },
				[{ id: 'run-2', topic: 'Launch calendar' }]
			),
			'No scheduled posts for Launch calendar match these filters.'
		);
		assert.equal(
			scheduleAgendaEmptyMessage({ status: 'cancelled', platform: 'instagram' }),
			'No cancelled instagram posts match these filters.'
		);
	});

	it('filters scheduled-post agenda rows by inclusive date range', () => {
		const posts = [
			{ id: 'schedule-1', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' },
			{ id: 'schedule-2', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T10:00:00Z' },
			{ id: 'schedule-3', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T11:00:00Z' }
		];

		assert.deepEqual(
			filteredSchedulePosts(posts, {
				status: 'scheduled',
				platform: 'instagram',
				from: '2026-05-18T10:00:00Z',
				to: '2026-05-18T11:00:00Z'
			}).map((post) => post.id),
			['schedule-2', 'schedule-3']
		);
		assert.equal(
			scheduleAgendaEmptyMessage({ status: 'all', platform: 'all', from: '2026-05-18T10:00' }),
			'No posts match this filter.'
		);
	});

	it('groups scheduled-post agenda rows by local calendar day', () => {
		const groups = scheduleAgendaGroups(
			[
				{ id: 'schedule-3', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-19T08:00:00Z' },
				{ id: 'schedule-2', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T11:00:00Z' },
				{ id: 'schedule-1', run_id: 'run-1', platform: 'instagram', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' }
			],
			[{ id: 'run-1', topic: 'AI launch' }],
			{ status: 'scheduled', platform: 'all' }
		);

		assert.deepEqual(groups.map((group) => [group.key, group.count, group.rows.map((row) => row.id)]), [
			['2026-05-18', 2, ['schedule-1', 'schedule-2']],
			['2026-05-19', 1, ['schedule-3']]
		]);
		assert.match(groups[0].label, /May 18, 2026/);
		assert.notEqual(groups[0].earliestTime, 'n/a');
	});

	it('keeps same-time scheduled-post agenda ordering deterministic', () => {
		assert.deepEqual(
			scheduleAgendaRows([
				{ id: 'schedule-b', run_id: 'run-1', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' },
				{ id: 'schedule-a', run_id: 'run-1', status: 'scheduled', scheduled_at: '2026-05-18T09:00:00Z' }
			]).map((row) => row.id),
			['schedule-a', 'schedule-b']
		);
	});

	it('returns no scheduled-post agenda groups for empty or filtered-out rows', () => {
		assert.deepEqual(scheduleAgendaGroups([], [], { status: 'scheduled', platform: 'all' }), []);
		assert.deepEqual(
			scheduleAgendaGroups(
				[{ id: 'schedule-1', platform: 'linkedin', status: 'cancelled', scheduled_at: '2026-05-18T09:00:00Z' }],
				[],
				{ status: 'scheduled', platform: 'linkedin' }
			),
			[]
		);
	});

	it('builds scheduled-post API paths from agenda filters', () => {
		assert.equal(scheduledPostsPath(), '/v1/scheduled-posts');
		assert.equal(scheduledPostsPath({ run_id: 'all', status: 'all', platform: 'all' }), '/v1/scheduled-posts');
		assert.equal(scheduledPostsPath({ run_id: 'run-1', status: 'all', platform: 'all' }), '/v1/scheduled-posts?run_id=run-1');
		assert.equal(scheduledPostsPath({ status: 'scheduled', platform: 'all' }), '/v1/scheduled-posts?status=scheduled');
		assert.equal(scheduledPostsPath({ status: 'all', platform: 'instagram' }), '/v1/scheduled-posts?platform=instagram');
		assert.equal(
			scheduledPostsPath({ status: 'cancelled', platform: 'linkedin' }),
			'/v1/scheduled-posts?status=cancelled&platform=linkedin'
		);
		assert.equal(
			scheduledPostsPath({
				run_id: 'run-1',
				status: 'scheduled',
				platform: 'instagram',
				from: '2026-05-18T09:00:00.000Z',
				to: '2026-05-19T09:00:00.000Z'
			}),
			'/v1/scheduled-posts?run_id=run-1&status=scheduled&platform=instagram&from=2026-05-18T09%3A00%3A00.000Z&to=2026-05-19T09%3A00%3A00.000Z'
		);
	});

	it('normalizes agenda date range filters for scheduled-post API requests', () => {
		assert.deepEqual(scheduleAgendaQueryFilters({ runID: 'all', status: 'all', platform: 'all', from: '', to: '' }), {
			filters: {},
			error: ''
		});
		assert.deepEqual(
			scheduleAgendaQueryFilters({
				status: 'scheduled',
				platform: 'linkedin',
				runID: 'run-1',
				from: '2026-05-18T09:00',
				to: '2026-05-18T10:00'
			}),
			{
				filters: {
					run_id: 'run-1',
					status: 'scheduled',
					platform: 'linkedin',
					from: '2026-05-18T09:00:00.000Z',
					to: '2026-05-18T10:00:00.000Z'
				},
				error: ''
			}
		);
		assert.equal(
			scheduleAgendaQueryFilters({ from: '2026-05-18T11:00', to: '2026-05-18T10:00' }).error,
			'Agenda range start must not be later than range end.'
		);
		assert.equal(
			scheduleAgendaQueryFilters({ from: 'not-a-date' }).error,
			'Agenda date range must use valid local date and time values.'
		);
	});

	it('builds dashboard summary counters from loaded details', () => {
		const summary = runSummary(
			{ id: 'run-1', topic: 'AI agents', content_type: 'carousel', template_id: 'carousel/ai-news-clean' },
			{
				briefs: [{ version: 1 }, { version: 2 }],
				revisions: [{ id: 'rev-1' }],
				artifacts: [{ id: 'artifact-1' }, { id: 'artifact-2' }]
			}
		);

		assert.equal(summary.latestVersion, 2);
		assert.equal(summary.revisionCount, 1);
		assert.equal(summary.artifactCount, 2);
	});

	it('keeps template selection aligned to content type', () => {
		const templates = [
			{ id: 'video/ai-news-short', name: 'AI news short video', content_type: 'short_video' },
			{ id: 'carousel/ai-news-clean', name: 'AI news carousel', content_type: 'carousel' }
		];

		assert.equal(templateForType(templates, 'short_video'), 'video/ai-news-short');
		assert.equal(templateForType(templates, 'carousel'), 'carousel/ai-news-clean');
		assert.equal(templateForType(templates, 'thread'), '');
	});

	it('filters and sorts template catalog entries by content type', () => {
		const templates = [
			{ id: 'carousel/z', name: 'Zebra', content_type: 'carousel' },
			{ id: 'video/ai-news-short', name: 'AI news short video', content_type: 'short_video' },
			{ id: 'carousel/a', name: 'Alpha', content_type: 'carousel' },
			{ id: 'carousel/fallback', name: '', content_type: 'carousel' }
		];

		assert.deepEqual(
			templatesForType(templates, 'carousel').map((template) => template.id),
			['carousel/a', 'carousel/fallback', 'carousel/z']
		);
	});

	it('formats template display labels', () => {
		assert.equal(templateLabel({ id: 'carousel/ai-news-clean', name: 'AI News Clean Carousel' }), 'AI News Clean Carousel');
		assert.equal(templateLabel({ id: 'carousel/custom', name: '' }), 'carousel/custom');
		assert.equal(templateContentTypeLabel('short_video'), 'Short video');
		assert.equal(templateContentTypeLabel('carousel'), 'Carousel');
	});

	it('formats invalid dates without throwing', () => {
		assert.equal(formatShortDate('not-a-date'), 'n/a');
	});

	it('filters workflow presets by content type and labels steps', () => {
		const workflows = [
			{ id: 'video-flow', name: 'Video Flow', content_type: 'short_video' },
			{ id: 'simple-carousel', name: 'Simple Carousel', content_type: 'carousel' },
			{ id: 'fallback-carousel', name: '', content_type: 'carousel' }
		];

		assert.deepEqual(
			workflowsForType(workflows, 'carousel').map((workflow) => workflow.id),
			['fallback-carousel', 'simple-carousel']
		);
		assert.equal(workflowForType(workflows, 'short_video'), 'video-flow');
		assert.equal(workflowLabel({ id: 'fallback-carousel', name: '' }), 'fallback-carousel');
		assert.equal(workflowStepLabel({ type: 'research_plan' }), 'Research plan');
	});

	it('builds a workflow-backed create-run payload', () => {
		assert.deepEqual(
			createRunPayload({
				workflow_id: 'simple-carousel',
				topic: 'AI agents in marketing',
				content_type: 'carousel',
				template_id: 'carousel/ai-news-clean',
				tone: 'clear and practical',
				platform: 'instagram',
				target_audience: 'content operators'
			}),
			{
				workflow_id: 'simple-carousel',
				topic: 'AI agents in marketing',
				tone: 'clear and practical',
				platform: 'instagram',
				target_audience: 'content operators'
			}
		);
	});

	it('builds a manual create-run payload without workflow metadata', () => {
		assert.deepEqual(
			createRunPayload({
				workflow_id: '',
				topic: 'AI agents in marketing',
				content_type: 'carousel',
				template_id: 'carousel/ai-news-clean',
				tone: 'clear and practical',
				platform: 'instagram',
				target_audience: 'content operators'
			}),
			{
				topic: 'AI agents in marketing',
				content_type: 'carousel',
				template_id: 'carousel/ai-news-clean',
				tone: 'clear and practical',
				platform: 'instagram',
				target_audience: 'content operators'
			}
		);
	});

	it('derives an empty run workflow timeline', () => {
		const timeline = workflowTimeline({ briefs: [], revisions: [], artifacts: [], scheduled_posts: [] });

		assert.deepEqual(
			timeline.map((step) => [step.key, step.status]),
			[
				['research', 'pending'],
				['brief', 'pending'],
				['revision', 'pending'],
				['render', 'pending'],
				['schedule', 'pending']
			]
		);
	});

	it('marks rendered runs in the workflow timeline', () => {
		const timeline = workflowTimeline({
			briefs: [{ version: 1, created_at: '2026-05-11T00:00:00Z' }],
			revisions: [],
			artifacts: [
				{ kind: 'content_json', created_at: '2026-05-11T00:01:00Z' },
				{ kind: 'carousel_png', created_at: '2026-05-11T00:02:00Z' }
			],
			scheduled_posts: []
		});

		assert.equal(timeline.find((step) => step.key === 'brief').status, 'done');
		assert.equal(timeline.find((step) => step.key === 'render').status, 'done');
	});

	it('marks revised runs in the workflow timeline', () => {
		const timeline = workflowTimeline({
			briefs: [{ version: 1 }, { version: 2 }],
			revisions: [{ id: 'rev-1', created_at: '2026-05-11T00:03:00Z' }],
			artifacts: [],
			scheduled_posts: []
		});

		const revision = timeline.find((step) => step.key === 'revision');
		assert.equal(revision.status, 'done');
		assert.equal(revision.detail, '1 revision saved');
	});

	it('marks scheduled runs in the workflow timeline', () => {
		const timeline = workflowTimeline({
			briefs: [{ version: 1 }],
			revisions: [],
			artifacts: [{ kind: 'carousel_png' }],
			scheduled_posts: [{ id: 'post-1', scheduled_at: '2026-05-12T00:00:00Z' }]
		});

		const schedule = timeline.find((step) => step.key === 'schedule');
		assert.equal(schedule.status, 'done');
		assert.equal(schedule.detail, '1 scheduled post');
	});

	it('uses latest timeline timestamps from unsorted API responses', () => {
		const timeline = workflowTimeline({
			briefs: [
				{ version: 2, created_at: '2026-05-11T00:05:00Z' },
				{ version: 1, created_at: '2026-05-11T00:00:00Z' }
			],
			revisions: [
				{ id: 'rev-2', created_at: '2026-05-11T00:06:00Z' },
				{ id: 'rev-1', created_at: '2026-05-11T00:03:00Z' }
			],
			artifacts: [
				{ kind: 'carousel_png', created_at: '2026-05-11T00:02:00Z' },
				{ kind: 'caption_text', created_at: '2026-05-11T00:07:00Z' }
			],
			scheduled_posts: [
				{ id: 'post-1', scheduled_at: '2026-05-12T00:00:00Z' },
				{ id: 'post-2', scheduled_at: '2026-05-13T00:00:00Z' }
			]
		});

		assert.equal(timeline.find((step) => step.key === 'brief').at, '2026-05-11T00:05:00Z');
		assert.equal(timeline.find((step) => step.key === 'revision').at, '2026-05-11T00:06:00Z');
		assert.equal(timeline.find((step) => step.key === 'render').at, '2026-05-11T00:07:00Z');
		assert.equal(timeline.find((step) => step.key === 'schedule').at, '2026-05-13T00:00:00Z');
	});
});
