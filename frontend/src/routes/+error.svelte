<script lang="ts">
	import { page } from '$app/stores';
</script>

<div style="min-height: 100vh; display: flex; align-items: center; justify-content: center; padding: 1.5rem; text-align: center;">
	<div>
		<h1 style="font-size: 3rem; margin-bottom: 0.5rem;">{$page.status}</h1>
		<p style="margin-bottom: 1.25rem;">
			{$page.status === 404 ? 'This page could not be found.' : 'Something went wrong. Please try again.'}
		</p>
		<a href="/">Back to dashboard</a>
	</div>
</div>
