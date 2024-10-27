<script lang="ts">
  import { onMount } from "svelte";
  import { initializeGame } from "./game";

  let canvas: HTMLCanvasElement;

  onMount(() => {
    initializeGame(canvas);
  });
</script>

<main>
  <canvas bind:this={canvas} class="gameCanvas"></canvas>
</main>
