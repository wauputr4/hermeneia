import assert from 'node:assert/strict';
import { describe, it } from 'node:test';

import {
	artifactGroups,
	artifactPreviewType,
	formatShortDate,
	latestBrief,
	runSummary,
	templateForType
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
		assert.equal(templateForType('video'), 'video/ai-news-short');
		assert.equal(templateForType('carousel'), 'carousel/ai-news-clean');
	});

	it('formats invalid dates without throwing', () => {
		assert.equal(formatShortDate('not-a-date'), 'n/a');
	});
});
