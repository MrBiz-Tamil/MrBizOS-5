<script lang="ts">
	import { goto } from '$app/navigation';
	import { onMount } from 'svelte';

	onMount(() => {
		goto('/customer/services');
	});
</script>

<div style="min-height: 100vh; background: #090d16; display: flex; align-items: center; justify-content: center; color: #94a3b8;">
	<span>Redirecting to Customer Travel Services...</span>
</div>
