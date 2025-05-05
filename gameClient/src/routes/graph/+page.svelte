<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { currentUser } from "$lib/stores/currentUser";
  import { get } from "svelte/store";
  import { GraphGame, type Player, type Monster, type Point } from "./game"; // Assuming types are exported

  let canvas: HTMLCanvasElement;
  let ctx: CanvasRenderingContext2D;
  let game: GraphGame;
  let expression = $state("");
  let coordinates = $state("(0, 0)");
  let messages = $state<string[]>([]);
  let numOfPlayers = $state(0);
  let players = $state<Player[]>([]); // Use $state for reactive player list

  const width = 800;
  const height = 600;
  // Ensure WebSocket URL is correct for your deployment environment
  const socket = new WebSocket("ws://localhost:8080/graph");
  const user = get(currentUser);

  onMount(() => {
    if (!canvas) {
      console.error("Canvas element not found!");
      return;
    }
    ctx = canvas.getContext("2d")!;
    if (!ctx) {
      console.error("Failed to get 2D context");
      return;
    }
    game = new GraphGame(width, height, socket);
    drawAxes(); // Initial draw

    socket.onopen = () => {
      console.log("Connected to server");
      if (user) {
        socket.send(
          JSON.stringify({
            type: "init",
            // Ensure user object has username property
            player: { id: user.id, name: user.username || "Unknown" },
          })
        );
      } else {
        console.warn("User data not available on connect.");
        // Handle case where user is not logged in or data isn't ready
        // Maybe redirect to login or show a message
      }
    };

    socket.onmessage = (event) => {
      try {
        const data = JSON.parse(event.data);
        console.log("Received:", data);

        if (data.type === "gameState") {
          game.updateMonsters(data.monsters || []);
          game.updatePlayers(data.players || []);
          game.updateGraphs(data.graphs || {});
          players = game.getPlayers(); // Update reactive state
          numOfPlayers = players.length;
          redrawCanvas(); // Centralized redraw function
        } else if (data.type === "playerJoinedOrLeave") {
          messages = data.message || [];
          numOfPlayers = data.totalPlayer || 0;
          // Server should send updated player list in gameState or here
          if (data.players) {
            game.updatePlayers(data.players);
            players = game.getPlayers(); // Update reactive state
            redrawCanvas(); // Redraw scores etc.
          } else {
            // Request gameState if player list isn't included?
          }
        } else if (data.type === "graphUpdate") {
          // This might be redundant if gameState includes graphs
          game.updateGraphs(data.graphs || {});
          redrawCanvas();
        } else if (data.type === "error") {
          console.error("Server error:", data.message);
          messages = [...messages, `Error: ${data.message}`];
        }
      } catch (error) {
        console.error("Failed to process message:", error);
      }
    };

    socket.onerror = (error) => {
      console.error("WebSocket Error:", error);
      messages = [...messages, "Connection error."];
    };

    socket.onclose = () => {
      console.log("Disconnected from server");
      messages = [...messages, "Disconnected."];
      // Maybe implement reconnection logic here
    };

    // Initial draw when component mounts and game is ready
    redrawCanvas();
  });

  onDestroy(() => {
    if (socket && socket.readyState === WebSocket.OPEN) {
      socket.close();
    }
    // No need to remove canvas listeners if added via on:mousemove
  });

  // --- Central Redraw Function ---
  function redrawCanvas() {
    if (!ctx || !game) return;
    drawAxes();
    drawMonsters(game.getMonsters());
    drawScores(game.getPlayers()); // Use game state directly
    drawGraphs(game.getGraphs()); // Use game state directly
  }

  // --- Drawing Functions ---
  function drawAxes() {
    if (!ctx) return;
    ctx.clearRect(0, 0, width, height);
    ctx.strokeStyle = "#eee"; // Very light grid lines
    ctx.lineWidth = 0.5;

    const gridSize = game.getScale() || 20; // Use scale from game instance

    // Draw grid lines based on game scale
    // Vertical lines
    for (let x = width / 2; x <= width; x += gridSize) {
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, height);
      ctx.stroke();
    }
    for (let x = width / 2 - gridSize; x >= 0; x -= gridSize) {
      ctx.beginPath();
      ctx.moveTo(x, 0);
      ctx.lineTo(x, height);
      ctx.stroke();
    }
    // Horizontal lines
    for (let y = height / 2; y <= height; y += gridSize) {
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }
    for (let y = height / 2 - gridSize; y >= 0; y -= gridSize) {
      ctx.beginPath();
      ctx.moveTo(0, y);
      ctx.lineTo(width, y);
      ctx.stroke();
    }

    // Draw main axes
    ctx.strokeStyle = "black";
    ctx.lineWidth = 1.5;
    ctx.beginPath();
    ctx.moveTo(0, height / 2);
    ctx.lineTo(width, height / 2); // X-axis
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(width / 2, 0);
    ctx.lineTo(width / 2, height); // Y-axis
    ctx.stroke();
  }

  function drawMonsters(monsters: Monster[]) {
    if (!ctx || !game) return;
    ctx.save();
    monsters.forEach((monster) => {
      // Convert game coordinates to canvas coordinates for drawing
      const canvasCoords = game.toCanvasCoords(monster.x, monster.y);
      ctx.fillStyle =
        monster.ofPlayer === user?.id
          ? "rgba(0, 0, 255, 0.8)"
          : "rgba(255, 0, 0, 0.8)"; // Semi-transparent
      ctx.beginPath();
      ctx.arc(canvasCoords.x, canvasCoords.y, 6, 0, 2 * Math.PI);
      ctx.fill();
      ctx.strokeStyle = "black";
      ctx.lineWidth = 1;
      ctx.stroke();
    });
    ctx.restore();
  }

  function drawScores(playerList: Player[]) {
    if (!ctx) return;
    ctx.save();
    ctx.fillStyle = "black";
    ctx.font = "14px Arial";
    ctx.textAlign = "left";
    ctx.textBaseline = "top";
    let yPos = 10;
    playerList.forEach((player) => {
      ctx.fillText(`${player.name}: ${player.score || 0}`, 10, yPos);
      yPos += 18; // Adjust spacing
    });
    ctx.restore();
  }

  function drawGraphs(graphs: { [playerId: string]: Point[] }) {
    if (!ctx || !game) return;
    ctx.save();
    ctx.lineWidth = 2;

    Object.entries(graphs).forEach(([playerId, graphPoints]) => {
      // Use different colors for different players' graphs
      ctx.strokeStyle = playerId === user?.id ? "darkorange" : "purple";
      ctx.beginPath();
      graphPoints.forEach((point, index) => {
        // Convert game coordinates to canvas coordinates for drawing
        const canvasPoint = game.toCanvasCoords(point.x, point.y);
        if (index === 0) {
          ctx.moveTo(canvasPoint.x, canvasPoint.y);
        } else {
          // Add check for large jumps (potential discontinuities)
          const prevCanvasPoint = game.toCanvasCoords(
            graphPoints[index - 1].x,
            graphPoints[index - 1].y
          );
          if (Math.abs(canvasPoint.y - prevCanvasPoint.y) < height * 0.8) {
            // Heuristic check
            ctx.lineTo(canvasPoint.x, canvasPoint.y);
          } else {
            ctx.moveTo(canvasPoint.x, canvasPoint.y); // Start new line segment
          }
        }
      });
      ctx.stroke();
    });
    ctx.restore();
  }

  // --- Event Handlers ---
  function addMonster() {
    if (!game || !user) {
      console.error("Game or user not initialized");
      return;
    }
    // Add monster locally first for responsiveness (optional)
    // const monster = game.addMonster(user.id);
    // if (monster) {
    //    redrawCanvas(); // Redraw immediately locally
    // } else {
    //    console.error("Failed to add monster locally");
    //    return; // Don't send if local add failed
    // }

    // Send request to server
    if (socket.readyState === WebSocket.OPEN) {
      // Server should generate the monster and broadcast via gameState
      socket.send(JSON.stringify({ type: "addMonster", playerId: user.id }));
    } else {
      console.error("Cannot add monster: WebSocket not connected.");
      messages = [...messages, "Error: Not connected to server."];
    }
  }

  function handleMouseMove(event: MouseEvent) {
    if (!canvas || !game) return;
    const rect = canvas.getBoundingClientRect();
    // Convert canvas mouse coordinates to game coordinates
    const gameCoords = game.fromCanvasCoords(
      event.clientX - rect.left,
      event.clientY - rect.top
    );
    coordinates = `(${gameCoords.x.toFixed(1)}, ${gameCoords.y.toFixed(1)})`; // Show game coords
  }

  async function startGraphAnimation() {
    if (!game || !user) {
      console.error("Game or user not initialized");
      return;
    }
    if (!expression.trim()) {
      alert("Please enter a function expression.");
      return;
    }
    if (!game.validateExpression(expression)) {
      alert(
        "Invalid expression! Use 'x' as the variable and standard math functions (e.g., x^2, sin(x))."
      );
      return;
    }

    try {
      // Generate points locally first for animation
      const points = await game.generateGraphPoints(expression);

      // Send the graph data to the server
      if (socket.readyState === WebSocket.OPEN) {
        socket.send(
          JSON.stringify({
            type: "graph",
            playerId: user.id,
            expression: expression, // Send expression for server-side validation/storage
            points: points, // Send calculated points
          })
        );
      } else {
        console.error("Socket not open, cannot send graph.");
        messages = [...messages, "Error: Not connected to server."];
        return; // Don't animate if we can't send
      }

      // Animate locally using the generated points
      // The server will eventually send a gameState update to sync this graph permanently
      await game.animateGraph(expression, (currentAnimatingPoints, isDone) => {
        redrawCanvas(); // Redraw the base state (axes, existing graphs, monsters, scores)

        // Draw the *currently animating* graph segment on top
        ctx.save();
        ctx.strokeStyle = "rgba(255, 165, 0, 0.7)"; // Semi-transparent orange for animation
        ctx.lineWidth = 2.5; // Slightly thicker for emphasis
        ctx.beginPath();
        currentAnimatingPoints.forEach((point, index) => {
          // Convert game coordinates from animation step to canvas coordinates
          const canvasPoint = game.toCanvasCoords(point.x, point.y);
          if (index === 0) {
            ctx.moveTo(canvasPoint.x, canvasPoint.y);
          } else {
            // Add the same discontinuity check as in drawGraphs
            const prevCanvasPoint = game.toCanvasCoords(
              currentAnimatingPoints[index - 1].x,
              currentAnimatingPoints[index - 1].y
            );
            if (Math.abs(canvasPoint.y - prevCanvasPoint.y) < height * 0.8) {
              ctx.lineTo(canvasPoint.x, canvasPoint.y);
            } else {
              ctx.moveTo(canvasPoint.x, canvasPoint.y);
            }
          }
        });
        ctx.stroke();
        ctx.restore();

        // Optional: If animation is done, you might trigger another redrawCanvas
        // to ensure the final graph (from gameState) is shown cleanly.
        // if (isDone) {
        //    console.log("Local animation finished.");
        // }
      });
    } catch (error) {
      console.error("Error generating or animating graph:", error);
      alert(`Failed to process the expression: ${error}`);
      messages = [...messages, "Error processing expression."];
    }
  }
</script>

<!-- HTML Structure remains largely the same -->
<div class="pageContainer">
  <h1>Graphing Game</h1>

  <div class="mainContent">
    <!-- Messages Panel -->
    <div class="messagesPanel">
      <h2>Game Log & Players</h2>
      {#if messages.length > 0}
        <div class="log">
          {#each messages as message}
            <div>{message}</div>
          {/each}
        </div>
      {/if}
      <div class="playerInfo">
        Players: {numOfPlayers}
      </div>
      <!-- Display player list and scores -->
      <div class="playerScores">
        <strong>Scores:</strong>
        {#if players.length > 0}
          <ul>
            {#each players as p (p.id)}
              <li>{p.name}: {p.score || 0}</li>
            {/each}
          </ul>
        {:else}
          <div>No players yet.</div>
        {/if}
      </div>
      {#if user}
        <div class="userInfo">
          You are: {user.username || "Unknown"} ({user.id})
        </div>
      {/if}
    </div>

    <!-- Game Area -->
    <div class="gameArea">
      <div class="controls">
        <input
          type="text"
          bind:value={expression}
          placeholder="y = f(x) | Ex: x^2 + sin(x)"
          aria-label="Function input"
        />
        <button onclick={addMonster}>Add Monster</button>
        <button onclick={startGraphAnimation}>Draw Graph</button>
      </div>
      <canvas bind:this={canvas} {width} {height} onmousemove={handleMouseMove}
      ></canvas>
      <div class="coordinates">Coords: {coordinates}</div>
    </div>
  </div>
</div>

<!-- Styles remain largely the same, added styles for playerScores -->
<style>
  /* ... (previous styles) ... */

  .messagesPanel {
    /* ... */
    gap: 10px; /* Adjusted gap */
  }

  .playerScores {
    font-size: 0.9em;
    margin-top: 5px;
  }
  .playerScores strong {
    display: block;
    margin-bottom: 3px;
  }
  .playerScores ul {
    list-style: none;
    padding: 0;
    margin: 0;
  }
  .playerScores li {
    margin-bottom: 2px;
  }

  .log {
    max-height: 250px; /* Adjusted height */
    /* ... */
  }

  .playerInfo,
  .userInfo {
    /* ... */
    font-size: 0.95em; /* Slightly smaller */
  }

  .gameArea {
    /* ... */
    flex: 3; /* Allow game area to take more space */
    min-width: 400px;
  }

  .controls input[type="text"] {
    /* ... */
    min-width: 250px; /* Wider input */
  }

  canvas {
    border: 1px solid #aaa; /* Lighter border */
    display: block;
    max-width: 100%;
    height: auto;
    background-color: #fff; /* White background for contrast */
  }

  .coordinates {
    /* ... */
    min-height: 1.3em; /* Ensure space */
  }

  /* ... (responsive styles) ... */
  @media (max-width: 900px) {
    .mainContent {
      flex-direction: column;
      align-items: center;
    }
    .messagesPanel {
      max-width: 90%;
      order: 2; /* Messages below game */
      margin-top: 20px;
    }
    .gameArea {
      order: 1;
      width: 100%; /* Ensure game area takes full width */
    }
    /* Ensure canvas does not exceed viewport width */
    canvas {
      width: 100%;
      max-width: min(800px, 95vw); /* Limit max width, consider viewport */
      height: auto; /* Maintain aspect ratio */
    }
  }

  @media (max-width: 480px) {
    h1 {
      font-size: 1.5em;
    }
    .controls {
      flex-direction: column;
      align-items: stretch;
    }
    .controls input[type="text"] {
      min-width: unset;
    }
    .messagesPanel {
      max-width: 100%;
    }
  }
</style>
