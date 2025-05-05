<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { initializeGame, disconnect } from "./game";
  // Import các store cần thiết
  import { messages, playersStore } from "./store";
  import { currentUser } from "$lib/stores/currentUser"; // Để lấy userID

  let canvas: HTMLCanvasElement | null = null;
  let messageLogDiv: HTMLDivElement | null = null; // For auto-scrolling log

  // Lấy userID để highlight người chơi hiện tại
  const userId = $currentUser?.id;

  // Biến derived để lấy danh sách người chơi từ store object
  let playerList = $derived(
    Object.values($playersStore).sort((a, b) => (b.score ?? 0) - (a.score ?? 0))
  ); // Sắp xếp theo điểm giảm dần

  onMount(() => {
    if (canvas) {
      initializeGame(canvas);
    } else {
      console.error("Canvas element not found on mount.");
      // Có thể hiển thị lỗi cho người dùng
    }
  });

  onDestroy(() => {
    disconnect(); // Gọi hàm disconnect khi component bị hủy
  });

  // --- Auto-scroll message log ---
  function scrollToBottom() {
    if (messageLogDiv) {
      messageLogDiv.scrollTop = messageLogDiv.scrollHeight;
    }
  }

  $effect(() => {
    // Auto-scroll message log when new messages are added
    if (messageLogDiv && $messages) {
      // Need to wait a tick for the DOM to update with new messages
      Promise.resolve().then(scrollToBottom);
    }
  });
</script>

<div class="pageContainerSnake">
  <h1>Multiplayer Snake</h1>
  <div class="mainContentSnake">
    <!-- Left Panel: Info & Messages -->
    <div class="leftPanelSnake">
      <div class="scoreboard">
        <h3>Scoreboard ({playerList.length})</h3>
        <ul>
          {#if playerList.length === 0}
            <li>Waiting for players...</li>
          {/if}
          {#each playerList as player (player.id)}
            <li class={player.id === userId ? "you" : ""}>
              <span class="playerName"
                >{player.id === userId
                  ? "You"
                  : `Player ${player.id.substring(0, 4)}`}</span
              >
              <span class="playerScore">{player.score ?? 0} pts</span>
            </li>
          {/each}
        </ul>
      </div>

      <div class="messagesPanelSnake">
        <h3>Game Log</h3>
        <div bind:this={messageLogDiv} class="logSnake">
          {#if $messages.length === 0}
            <div class="noMessagesSnake">Connecting...</div>
          {/if}
          {#each $messages as message, i (i)}
            <div>{message}</div>
          {/each}
        </div>
      </div>
    </div>

    <!-- Right Panel: Game Canvas -->
    <div class="rightPanelSnake">
      <canvas bind:this={canvas} class="gameCanvasSnake"></canvas>
      <div class="controls-info">Use WASD or Arrow Keys to move</div>
    </div>
  </div>
</div>

<style>
  /* Make page container fill available height and prevent its own scroll */
  .pageContainerSnake {
    max-width: 1100px;
    margin: 0 auto; /* Remove vertical margin */
    padding: 15px;
    font-family: sans-serif;
    background-color: #f8f9fa;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.08);

    /* Height management */
    /* Adjust 40px based on actual layout padding/margins outside this component */
    /* Consider the sidebar width if it affects available height */
    height: calc(100vh - 40px);
    display: flex;
    flex-direction: column;
    overflow: hidden; /* Prevent this container from scrolling */
  }

  h1 {
    text-align: center;
    color: #2e7d32;
    margin-top: 0; /* Remove top margin */
    margin-bottom: 15px; /* Reduce bottom margin */
    flex-shrink: 0; /* Prevent h1 from shrinking */
  }

  /* Make main content fill remaining height */
  .mainContentSnake {
    flex-grow: 1; /* Take remaining vertical space */
    display: flex;
    gap: 30px;
    overflow: hidden; /* Prevent this container from scrolling */
  }

  /* Make panels fill height */
  .leftPanelSnake {
    flex: 1;
    min-width: 280px;
    display: flex;
    flex-direction: column;
    gap: 15px; /* Reduce gap */
    overflow: hidden; /* Prevent panel overflow */
  }

  .rightPanelSnake {
    flex: 2;
    display: flex;
    flex-direction: column;
    justify-content: center; /* Center content vertically */
    align-items: center; /* Center content horizontally */
    overflow: hidden; /* Prevent panel overflow */
  }

  .scoreboard {
    flex-shrink: 0; /* Prevent scoreboard from shrinking */
  }

  .scoreboard h3,
  .messagesPanelSnake h3 {
    margin-top: 0;
    margin-bottom: 8px; /* Reduce margin */
    color: #555;
    border-bottom: 1px solid #ddd;
    padding-bottom: 4px; /* Reduce padding */
    font-size: 1.1em; /* Slightly smaller */
  }

  .scoreboard ul {
    list-style: none;
    padding: 0;
    margin: 0;
    max-height: 180px; /* Add max-height and scroll if too many players */
    overflow-y: auto; /* Allow internal scroll for player list */
  }
  .scoreboard li {
    display: flex;
    justify-content: space-between;
    padding: 5px 8px; /* Reduce padding */
    margin-bottom: 4px; /* Reduce margin */
    border-radius: 4px;
    background-color: #fff;
    border: 1px solid #eee;
    font-size: 0.9em; /* Slightly smaller */
    transition: background-color 0.2s ease;
  }
  .scoreboard li.you {
    font-weight: bold;
    background-color: #e8f5e9;
    border-left: 4px solid #4caf50;
    padding-left: 10px;
  }
  .playerName {
    color: #333;
  }
  .playerScore {
    color: #1e88e5;
    font-weight: bold;
  }

  /* Make message panel fill remaining space in left panel */
  .messagesPanelSnake {
    flex-grow: 1; /* Take remaining vertical space */
    display: flex;
    flex-direction: column;
    overflow: hidden; /* Important: prevent pushing other elements */
  }

  /* Make log fill message panel and scroll internally */
  .logSnake {
    flex-grow: 1; /* Take available space */
    overflow-y: auto; /* Allow internal scrolling */
    font-size: 0.9em;
    color: #444;
    border: 1px solid #ddd;
    padding: 10px; /* Reduce padding */
    background-color: #fff;
    border-radius: 4px;
    line-height: 1.5; /* Adjust line height */
    /* Remove max-height if previously set */
  }
  .noMessagesSnake {
    color: #999;
    font-style: italic;
  }

  .gameCanvasSnake {
    border: 3px solid #444;
    display: block;
    background-color: #ffffff;
    box-shadow: 0 4px 8px rgba(0, 0, 0, 0.1);
    /* Kích thước canvas được đặt trong JS (CANVAS_SIZE) */
    flex-shrink: 0; /* Prevent canvas from shrinking */
  }

  .controls-info {
    margin-top: 10px; /* Reduce margin */
    font-size: 0.9em;
    color: #666;
    text-align: center;
    flex-shrink: 0; /* Prevent shrinking */
  }
</style>
