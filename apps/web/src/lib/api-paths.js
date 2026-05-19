export function scheduledPostsPath(filters = {}) {
	const params = new URLSearchParams();
	if (filters.run_id && filters.run_id !== 'all') {
		params.set('run_id', filters.run_id);
	}
	if (filters.artifact_id && filters.artifact_id !== 'all') {
		params.set('artifact_id', filters.artifact_id);
	}
	if (filters.status && filters.status !== 'all') {
		params.set('status', filters.status);
	}
	if (filters.platform && filters.platform !== 'all') {
		params.set('platform', filters.platform);
	}
	if (filters.from) {
		params.set('from', filters.from);
	}
	if (filters.to) {
		params.set('to', filters.to);
	}
	const query = params.toString();
	return query ? `/v1/scheduled-posts?${query}` : '/v1/scheduled-posts';
}
