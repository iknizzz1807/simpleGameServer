<script lang="ts">
  import { onMount, onDestroy } from "svelte";
  import { currentUser } from "$lib/stores/currentUser";
  import { get } from "svelte/store";
  import { GraphGame } from "./game";
  import type { Monster } from "./game";

  let canvas: HTMLCanvasElement;
  let ctx: CanvasRenderingContext2D;
  let game: GraphGame;
  let expression = $state("");
  let coordinates = $state("(0, 0)");
  let messages = $state<string[]>([]);
  let numOfPlayers: number = $state(0);

  const width = 800;
  const height = 600;
  const socket = new WebSocket("ws://localhost:8080/graph");
  const user = get(currentUser);

  onMount(() => {
    ctx = canvas.getContext("2d")!;
    game = new GraphGame(width, height, socket);
    drawAxes();

    socket.onopen = () => {
      console.log("Connected to server");
      socket.send(
        JSON.stringify({
          type: "init",
          player: {
            id: user?.id,
            name: user?.username,
          },
        })
      );
    };

    socket.onmessage = (event) => {
      // Always running and listening on receiving messages
      const data = JSON.parse(event.data);
      if (data.type === "gameState") {
        drawAxes();
        drawMonsters(data.monsters);
      } else if (data.type === "playerJoinedOrLeave") {
        messages = data.message;
        numOfPlayers = data.totalPlayer;
      }
    };

    socket.onclose = () => {
      console.log("Disconnected from server");
    };
  });

  onDestroy(() => {
    socket.close();
  });

  function drawAxes() {
    ctx.clearRect(0, 0, width, height);

    ctx.save();

    ctx.strokeStyle = "black";
    ctx.lineWidth = 1;

    ctx.beginPath();
    ctx.moveTo(0, height / 2);
    ctx.lineTo(width, height / 2);
    ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(width / 2, 0);
    ctx.lineTo(width / 2, height);
    ctx.stroke();

    ctx.restore();
  }

  function drawMonsters(monsters: Monster[]) {
    ctx.save();

    monsters.forEach((monster) => {
      const { x, y } = game.toCanvasCoords(monster.x, monster.y);
      ctx.fillStyle = monster.type === "player" ? "blue" : "red";
      ctx.beginPath();
      ctx.arc(x, y, 5, 0, 2 * Math.PI);
      ctx.fill();
    });

    ctx.restore();
  }

  function addMonster(type: "player" | "other") {
    const monsters = game.addMonster(type);
    drawAxes();
    drawMonsters(monsters);
  }

  function handleMouseMove(event: MouseEvent) {
    const rect = canvas.getBoundingClientRect();
    const { x, y } = game.fromCanvasCoords(
      event.clientX - rect.left,
      event.clientY - rect.top
    );
    coordinates = `(${x.toFixed(2)}, ${y.toFixed(2)})`;
  }

  async function startGraphAnimation() {
    if (!game.validateExpression(expression)) {
      alert("Unaccepted expression!");
      return;
    }

    await game.animateGraph(expression, (points, monsters) => {
      drawAxes();
      drawMonsters(monsters);

      ctx.save();
      ctx.strokeStyle = "orange";
      ctx.lineWidth = 2;

      ctx.beginPath();
      points.forEach((point, index) => {
        if (index === 0) {
          ctx.moveTo(point.x, point.y);
        } else {
          ctx.lineTo(point.x, point.y);
        }
      });
      ctx.stroke();

      ctx.restore();
    });
  }
</script>

<div class="container">
  <div class="messages">
    {#each messages as message}
      <div>{message}</div>
    {/each}
    <div>Number of players: {numOfPlayers}</div>
  </div>
  <div class="gameArea">
    <div class="controls">
      <input
        type="text"
        bind:value={expression}
        placeholder="y = f(x) | Ex: x^2 + 2x + 1"
      />
      <button onclick={() => addMonster("player")}>Add player monster</button>
      <button onclick={() => addMonster("other")}>Add other monster</button>
      <button onclick={startGraphAnimation}>Draw</button>
    </div>

    <canvas bind:this={canvas} {width} {height} onmousemove={handleMouseMove}
    ></canvas>

    <div class="coordinates">{coordinates}</div>
  </div>
</div>

<style>
  .container {
    display: flex;
    flex-direction: row;
    width: 100%;
    font-family: Arial, sans-serif;
    justify-content: space-between;
  }

  canvas {
    border: 1px solid black;
  }

  .controls {
    display: flex;
    gap: 10px;
    margin: 8px;
    justify-content: center;
  }

  .gameArea {
    margin-right: 8px;
  }

  input,
  button {
    padding: 5px;
  }

  .coordinates {
    margin: 8px;
    font-weight: bold;
    text-align: center;
  }

  .messages {
    font-weight: bold;
    color: green;
    margin-left: 8px;
  }
</style>
