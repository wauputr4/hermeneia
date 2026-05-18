export function scheduledPostsPath(filters = {}) {
	const params = new URLSearchParams();
	if (filters.status && filters.status !== 'all') {
		params.set('status', filters.status);
	}
	if (filters.platform && filters.platform !== 'all') {
		params.set('platform', filters.platform);
	}
	const query = params.toString();
	return query ? `/v1/scheduled-posts?${query}` : '/v1/scheduled-posts';
}
