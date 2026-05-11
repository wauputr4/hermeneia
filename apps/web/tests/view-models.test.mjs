import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
	artifactGroups,
	artifactPreviewType,
	formatShortDate,
	latestBrief,
	runSummary,
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
});
