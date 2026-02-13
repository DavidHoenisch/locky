<script>
  import { currentRoute } from './router.js'
  
  export let path
  export let component
</script>

{#if $currentRoute === path}
  <svelte:component this={component} />
{/if}
