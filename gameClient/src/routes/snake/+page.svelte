<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { initializeGame, disconnect } from "./game";
  import { messages, numOfPlayers } from "./store";

  let canvas: HTMLCanvasElement;

  onMount(() => {
    initializeGame(canvas);
  });

  onDestroy(() => {
    disconnect();
  });
</script>

<main class="container">
  <div class="messages">
    {#each $messages as message}
      <div>{message}</div>
    {/each}
    <div class="num-players">{$numOfPlayers} players</div>
  </div>

  <canvas bind:this={canvas} class="gameCanvas"></canvas>
</main>

<style>
  .container {
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    width: 100%;
  }

  .messages {
    color: green;
    font-weight: bold;
  }

  .num-players {
    color: blue;
  }
</style>
