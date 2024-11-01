<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { initializeGame, handleCanvasClick, resetGame } from "./game";

  let canvas: HTMLCanvasElement;
  let playerStatusTag: HTMLElement;

  onMount(() => {
    initializeGame(canvas, playerStatusTag);
    canvas.addEventListener("click", handleCanvasClick);
  });

  onDestroy(() => {
    resetGame();
  });
</script>

<div bind:this={playerStatusTag}></div>
<button onclick={resetGame}>Reset Game</button>
<canvas bind:this={canvas}></canvas>

<style>
  canvas {
    border: 1px solid black;
  }
</style>
