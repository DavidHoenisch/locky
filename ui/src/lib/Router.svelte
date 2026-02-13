<script>
  import { currentRoute } from './router.js'
</script>

{#if $currentRoute}
  <slot />
{/if}
