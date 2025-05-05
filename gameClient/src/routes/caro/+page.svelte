<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { currentUser } from "$lib/stores/currentUser";
  import type { User } from "$lib/stores/currentUser"; // Import User type if not already
  import { get } from "svelte/store";

  // --- Constants ---
  const BOARD_SIZE = 15;
  const CELL_SIZE = 40; // Canvas pixels per cell

  // --- DOM Elements (Use bind:this) ---
  let canvasElement: HTMLCanvasElement | null = null;
  let playerStatusTag: HTMLElement | null = null;
  let messageLogDiv: HTMLDivElement | null = null; // For auto-scrolling

  // --- Reactive Game State ---
  let messages = $state<string[]>([]);
  let gameBoard = $state<string[][]>(
    Array(BOARD_SIZE)
      .fill(null)
      .map(() => Array(BOARD_SIZE).fill(""))
  ); // Initialize empty board
  let playerList = $state<any[]>([]); // Consider defining a Player type matching backend (minus Conn)
  let currentPlayerId = $state<string>("");
  let winnerId = $state<string>("");
  let socket: WebSocket | null = $state(null);
  let connectionStatus = $state<
    "connecting" | "connected" | "disconnected" | "error"
  >("connecting");

  // --- Derived State ---
  const numOfPlayers = $derived(playerList.length);
  const user = get(currentUser); // Get user data once
  const userId = user?.id; // Store user ID

  // --- WebSocket Logic ---
  function connectWebSocket() {
    if (!user) {
      console.error("Cannot connect: User not available.");
      connectionStatus = "error";
      messages = ["Error: User data not found. Please log in."];
      return;
    }

    console.log("Attempting to connect WebSocket...");
    // Ensure the address is correct for your deployment environment
    const ws = new WebSocket("ws://localhost:8080/caro");
    socket = ws; // Assign to state variable

    ws.onopen = () => {
      console.log("WebSocket connected. Sending init...");
      connectionStatus = "connected";
      messages = ["Connected to server."]; // Add initial connection message
      if (user) {
        // Send init message
        ws.send(
          JSON.stringify({
            type: "init",
            player: {
              id: user.id,
              name: user.username || `Anon_${user.id.substring(0, 4)}`, // Match backend default name logic
            },
          })
        );
      }
    };

    ws.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log("Received:", data);

        switch (data.type) {
          case "gameState":
            gameBoard = data.board || []; // Update board state
            playerList = data.players || []; // Update player list
            currentPlayerId = data.currentTurn || ""; // Update current turn
            winnerId = data.winner || ""; // Update winner
            break;
          case "playerJoinedOrLeave":
            // Append new notification messages
            messages = [...messages, ...(data.message || [])];
            // Note: playerList is updated via gameState, no need to use data.totalPlayer here
            break;
          case "error":
            console.error("Server error:", data.message);
            messages = [...messages, `Error: ${data.message}`];
            // Optionally change connectionStatus or close socket based on error type
            if (data.message === "Player ID already connected") {
              connectionStatus = "error";
              if (socket) socket.close(); // Close this duplicate connection
            }
            break;
          default:
            console.warn("Unknown message type:", data.type);
            messages = [
              ...messages,
              `Received unknown message type: ${data.type}`,
            ];
        }
      } catch (error) {
        console.error("Failed to parse message or handle update:", error);
        messages = [...messages, "Error processing server message."];
      }
    };

    ws.onerror = (error) => {
      console.error("WebSocket error:", error);
      messages = [...messages, "Connection error. Please refresh."];
      connectionStatus = "error";
      socket = null; // Clear socket on error
    };

    ws.onclose = (event) => {
      console.log(
        "WebSocket closed. Code:",
        event.code,
        "Reason:",
        event.reason || "No reason given" // Provide default if reason is empty
      );
      // Avoid adding duplicate "Disconnected" message if already in error state
      if (connectionStatus !== "error") {
        messages = [...messages, "Disconnected from server."];
        connectionStatus = "disconnected";
      }
      // Clear game state on disconnect for a cleaner experience if they refresh/reconnect
      gameBoard = Array(BOARD_SIZE)
        .fill(null)
        .map(() => Array(BOARD_SIZE).fill(""));
      playerList = [];
      currentPlayerId = "";
      winnerId = "";
      socket = null; // Clear socket on close
      // Optionally attempt to reconnect here using a timer and backoff strategy
    };
  }

  // --- Game Logic ---
  function handleCanvasClick(event: MouseEvent) {
    // Check connection, turn, game state etc.
    if (
      !canvasElement ||
      !socket ||
      socket.readyState !== WebSocket.OPEN ||
      !userId ||
      winnerId ||
      currentPlayerId !== userId
    ) {
      // Log detailed reason for ignoring click
      console.log("Cannot make move:", {
        canvas: !!canvasElement,
        socketState: socket?.readyState,
        userId,
        winnerId,
        isMyTurn: currentPlayerId === userId,
        isConnected: socket?.readyState === WebSocket.OPEN,
      });
      return;
    }

    const rect = canvasElement.getBoundingClientRect();
    const x = Math.floor((event.clientX - rect.left) / CELL_SIZE);
    const y = Math.floor((event.clientY - rect.top) / CELL_SIZE);

    // Validate coordinates and if cell is empty
    if (
      x < 0 ||
      x >= BOARD_SIZE ||
      y < 0 ||
      y >= BOARD_SIZE ||
      gameBoard[y]?.[x] !== ""
    ) {
      console.log("Invalid move (out of bounds or cell taken):", {
        x,
        y,
        cell: gameBoard[y]?.[x],
      });
      return;
    }

    console.log(`Sending move: { x: ${x}, y: ${y} }`);
    socket.send(JSON.stringify({ type: "move", move: { x, y } }));
  }

  function handleResetClick() {
    if (socket && socket.readyState === WebSocket.OPEN) {
      console.log("Sending reset request...");
      messages = [...messages, "Requesting game reset..."]; // Give user feedback
      socket.send(JSON.stringify({ type: "reset" }));
    } else {
      console.error("Cannot reset: WebSocket not connected.");
      messages = [...messages, "Cannot reset: Not connected."];
    }
  }

  // --- Drawing Logic ---
  function drawMark(
    ctx: CanvasRenderingContext2D,
    x: number,
    y: number,
    mark: string
  ) {
    const halfSize = CELL_SIZE / 2;
    const centerX = x * CELL_SIZE + halfSize;
    const centerY = y * CELL_SIZE + halfSize;
    const markPadding = CELL_SIZE * 0.15; // Padding from cell edge
    const markSize = halfSize - markPadding; // Size based on padding

    ctx.lineWidth = 3; // Mark line width
    ctx.strokeStyle = mark === "X" ? "#D32F2F" : "#1976D2"; // Use distinct red/blue

    if (mark === "X") {
      ctx.beginPath();
      ctx.moveTo(centerX - markSize, centerY - markSize);
      ctx.lineTo(centerX + markSize, centerY + markSize);
      ctx.moveTo(centerX + markSize, centerY - markSize);
      ctx.lineTo(centerX - markSize, centerY + markSize);
      ctx.stroke();
    } else if (mark === "O") {
      ctx.beginPath();
      ctx.arc(centerX, centerY, markSize, 0, Math.PI * 2);
      ctx.stroke();
    }
  }

  function drawBoard() {
    if (!canvasElement) return;
    const ctx = canvasElement.getContext("2d");
    if (!ctx) return;

    const canvasSize = BOARD_SIZE * CELL_SIZE;
    // Ensure canvas size is set correctly
    if (canvasElement.width !== canvasSize) canvasElement.width = canvasSize;
    if (canvasElement.height !== canvasSize) canvasElement.height = canvasSize;

    // Clear previous drawing
    ctx.clearRect(0, 0, canvasElement.width, canvasElement.height);

    // --- Draw Grid ---
    ctx.strokeStyle = "#BDBDBD"; // Lighter grey for grid
    ctx.lineWidth = 1;
    for (let i = 0; i <= BOARD_SIZE; i++) {
      // Vertical lines
      ctx.beginPath();
      ctx.moveTo(i * CELL_SIZE + 0.5, 0); // Offset by 0.5 for sharper lines
      ctx.lineTo(i * CELL_SIZE + 0.5, canvasElement.height);
      ctx.stroke();
      // Horizontal lines
      ctx.beginPath();
      ctx.moveTo(0, i * CELL_SIZE + 0.5);
      ctx.lineTo(canvasElement.width, i * CELL_SIZE + 0.5);
      ctx.stroke();
    }

    // --- Draw Marks ---
    // Check if gameBoard is valid before iterating
    if (gameBoard && gameBoard.length === BOARD_SIZE) {
      for (let y = 0; y < BOARD_SIZE; y++) {
        if (gameBoard[y] && gameBoard[y].length === BOARD_SIZE) {
          for (let x = 0; x < BOARD_SIZE; x++) {
            if (gameBoard[y][x]) {
              drawMark(ctx, x, y, gameBoard[y][x]);
            }
          }
        } else {
          console.warn(`Invalid gameBoard row at index ${y}`); // Warn if a row is invalid
        }
      }
    } else {
      console.warn("gameBoard is not valid for drawing"); // Warn if the board itself is invalid
    }

    // --- Update Status Tag ---
    if (playerStatusTag) {
      let statusText = "Connecting...";
      let statusClass = "status-connecting"; // CSS class for styling

      if (connectionStatus === "connected") {
        if (winnerId) {
          const winnerPlayer = playerList.find((p) => p.id === winnerId);
          statusText = `${winnerPlayer?.name || `Player ${winnerId.substring(0, 4)}`} won! 🎉`;
          statusClass = "status-winner";
        } else if (currentPlayerId) {
          const currentPlayer = playerList.find(
            (p) => p.id === currentPlayerId
          );
          const playerMark = currentPlayer?.mark
            ? ` (${currentPlayer.mark})`
            : "";
          if (currentPlayer?.id === userId) {
            statusText = `Your Turn${playerMark}`;
            statusClass = "status-my-turn";
          } else {
            statusText = `Turn: ${currentPlayer?.name || `Player ${currentPlayerId.substring(0, 4)}`}${playerMark}`;
            statusClass = "status-opponent-turn";
          }
        } else if (numOfPlayers < 2) {
          statusText = "Waiting for another player...";
          statusClass = "status-waiting";
        } else {
          statusText = "Game ready. X starts."; // More informative ready state
          statusClass = "status-ready";
        }
      } else if (connectionStatus === "disconnected") {
        statusText = "Disconnected. Refresh?";
        statusClass = "status-disconnected";
      } else if (connectionStatus === "error") {
        statusText = "Connection Error!";
        statusClass = "status-error";
      }
      playerStatusTag.textContent = statusText;
      // Apply CSS class for styling
      playerStatusTag.className = `statusTag ${statusClass}`;
    }
  }

  // --- Auto-scroll message log ---
  function scrollToBottom() {
    if (messageLogDiv) {
      messageLogDiv.scrollTop = messageLogDiv.scrollHeight;
    }
  }

  // --- Lifecycle ---
  onMount(() => {
    // Set canvas size initially
    if (canvasElement) {
      const canvasSize = BOARD_SIZE * CELL_SIZE;
      canvasElement.width = canvasSize;
      canvasElement.height = canvasSize;
    }
    connectWebSocket(); // Start connection attempt
  });

  onDestroy(() => {
    if (socket) {
      console.log("Closing WebSocket connection on component destroy.");
      socket.close();
      socket = null;
    }
  });

  // --- Reactive Effects ---
  $effect(() => {
    // Redraw board whenever relevant state changes
    if (canvasElement) {
      drawBoard();
    }
  });

  $effect(() => {
    // Auto-scroll message log when new messages are added
    if (messageLogDiv && messages) {
      // Need to wait a tick for the DOM to update with new messages
      Promise.resolve().then(scrollToBottom);
    }
  });
</script>

<div class="pageContainer">
  <h1>Caro Game</h1>
  <div class="mainContent">
    <!-- Left Panel: Info & Messages -->
    <div class="leftPanel">
      {#if user}
        <div class="userInfo">
          Logged in as: <strong>{user.username}</strong> ({user.id.substring(
            0,
            6
          )}...)
        </div>
      {:else}
        <div class="userInfo error">User not loaded. Please log in.</div>
      {/if}

      <div class="playerInfo">
        <h3>Players ({numOfPlayers})</h3>
        <ul>
          {#if playerList.length === 0}
            <li>No players connected.</li>
          {/if}
          {#each playerList as player (player.id)}
            <li class={player.id === userId ? "you" : ""}>
              {player.name || `Anon_${player.id.substring(0, 4)}`}
              {player.mark ? ` (${player.mark})` : ""}
              {#if player.id === userId}
                (You){/if}
              {#if player.id === currentPlayerId && !winnerId}
                <span class="turnIndicator">← Turn</span>{/if}
              {#if player.id === winnerId}
                <span class="winnerIndicator">🏆 Winner!</span>{/if}
            </li>
          {/each}
        </ul>
      </div>

      <div class="messagesPanel">
        <h3>Game Log</h3>
        <div bind:this={messageLogDiv} class="log">
          {#if messages.length === 0}
            <div class="noMessages">Waiting for connection...</div>
          {/if}
          {#each messages as message, i (i)}
            <div>{message}</div>
          {/each}
        </div>
      </div>
    </div>

    <!-- Right Panel: Game Area -->
    <div class="rightPanel">
      <div class="gameArea">
        <!-- Status Tag above Canvas -->
        <div bind:this={playerStatusTag} class="statusTag status-connecting">
          Initializing...
        </div>
        <!-- Canvas for the game board -->
        <canvas
          bind:this={canvasElement}
          on:click={handleCanvasClick}
          class={currentPlayerId === userId &&
          !winnerId &&
          connectionStatus === "connected"
            ? "my-turn-canvas"
            : ""}
        ></canvas>
        <!-- Reset Button below Canvas -->
        <button
          on:click={handleResetClick}
          class="resetButton"
          disabled={!socket || socket.readyState !== WebSocket.OPEN}
          >Reset Game</button
        >
      </div>
    </div>
  </div>
</div>

<style>
  .pageContainer {
    max-width: 1200px;
    margin: 20px auto;
    padding: 15px;
    font-family: sans-serif;
    background-color: #f4f4f4;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
  }

  h1 {
    text-align: center;
    color: #333;
    margin-bottom: 25px;
  }

  .mainContent {
    display: flex;
    gap: 30px; /* Space between left and right panels */
  }

  .leftPanel {
    flex: 1; /* Takes up available space */
    min-width: 250px; /* Minimum width */
    display: flex;
    flex-direction: column;
    gap: 20px;
  }

  .rightPanel {
    flex: 2; /* Takes up more space */
    display: flex;
    justify-content: center; /* Center game area horizontally */
    align-items: flex-start; /* Align game area to the top */
  }

  .userInfo {
    background-color: #e3f2fd;
    padding: 10px 15px;
    border-radius: 4px;
    border: 1px solid #bbdefb;
    font-size: 0.95em;
  }
  .userInfo.error {
    background-color: #ffebee;
    border-color: #ffcdd2;
    color: #c62828;
    font-weight: bold;
  }

  .playerInfo h3,
  .messagesPanel h3 {
    margin-top: 0;
    margin-bottom: 10px;
    color: #555;
    border-bottom: 1px solid #ddd;
    padding-bottom: 5px;
  }

  .playerInfo ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .playerInfo li {
    margin-bottom: 6px;
    padding: 4px 0;
    font-size: 0.95em;
  }
  .playerInfo li.you {
    font-weight: bold;
    color: #1976d2; /* Blue for 'You' */
  }
  .turnIndicator {
    font-weight: bold;
    color: #ffa000; /* Amber/Orange for turn */
    margin-left: 8px;
  }
  .winnerIndicator {
    font-weight: bold;
    color: #388e3c; /* Green for winner */
    margin-left: 8px;
  }

  .messagesPanel {
    flex-grow: 1; /* Allows message panel to fill remaining space */
    display: flex;
    flex-direction: column;
  }

  .log {
    flex-grow: 1; /* Allows log div to fill message panel */
    max-height: 400px; /* Adjust max height as needed */
    overflow-y: auto;
    font-size: 0.9em;
    color: #333;
    border: 1px solid #ddd;
    padding: 10px;
    background-color: #fff;
    border-radius: 4px;
    line-height: 1.5;
  }
  .noMessages {
    color: #888;
    font-style: italic;
  }

  .gameArea {
    display: flex;
    flex-direction: column;
    align-items: center; /* Center canvas and button */
  }

  .statusTag {
    margin-bottom: 15px;
    font-size: 1.1em;
    font-weight: bold;
    padding: 8px 15px;
    border-radius: 4px;
    text-align: center;
    min-width: 200px;
  }
  /* Status-specific styles */
  .status-connecting {
    background-color: #eee;
    color: #555;
  }
  .status-waiting {
    background-color: #fff9c4;
    color: #795548;
    border: 1px solid #fff59d;
  }
  .status-ready {
    background-color: #c8e6c9;
    color: #2e7d32;
    border: 1px solid #a5d6a7;
  }
  .status-my-turn {
    background-color: #bbdefb;
    color: #1565c0;
    border: 1px solid #90caf9;
    animation: pulse 1.5s infinite;
  }
  .status-opponent-turn {
    background-color: #e0e0e0;
    color: #424242;
  }
  .status-winner {
    background-color: #a5d6a7;
    color: #1b5e20;
    border: 1px solid #81c784;
  }
  .status-disconnected {
    background-color: #ffcdd2;
    color: #c62828;
    border: 1px solid #ef9a9a;
  }
  .status-error {
    background-color: #ffab91;
    color: #d84315;
    border: 1px solid #ff8a65;
    font-weight: bold;
  }

  @keyframes pulse {
    0% {
      box-shadow: 0 0 0 0 rgba(25, 118, 210, 0.4);
    }
    70% {
      box-shadow: 0 0 0 10px rgba(25, 118, 210, 0);
    }
    100% {
      box-shadow: 0 0 0 0 rgba(25, 118, 210, 0);
    }
  }

  canvas {
    border: 2px solid #333;
    display: block;
    background-color: white;
    cursor: pointer; /* Default cursor */
    box-shadow: 0 1px 5px rgba(0, 0, 0, 0.2);
  }
  canvas.my-turn-canvas {
    cursor: crosshair; /* Change cursor when it's player's turn */
    border-color: #1976d2; /* Highlight border on player's turn */
    box-shadow: 0 0 8px rgba(25, 118, 210, 0.5);
  }

  .resetButton {
    margin-top: 20px;
    padding: 10px 20px;
    font-size: 1em;
    cursor: pointer;
    background-color: #f44336; /* Red */
    color: white;
    border: none;
    border-radius: 4px;
    transition: background-color 0.2s ease;
  }
  .resetButton:hover:not(:disabled) {
    background-color: #d32f2f;
  }
  .resetButton:disabled {
    background-color: #ccc;
    cursor: not-allowed;
    opacity: 0.7;
  }
</style>
