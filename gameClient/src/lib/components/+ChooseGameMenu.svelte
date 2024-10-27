<script>
  import { goto } from "$app/navigation";
</script>

<main class="container">
  <button onclick={() => goto("/snake")}> Snake </button>
  <button onclick={() => goto("/caro")}> Caro </button>
  <button onclick={() => goto("/graph")}> Graph Game </button>
</main>

<style>
  .container {
    display: flex;
    gap: 10px;
  }
</style>
