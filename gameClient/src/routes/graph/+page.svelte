<script lang="ts">
  import { onMount } from "svelte";
  import { GraphGame } from "./game";

  let canvas: HTMLCanvasElement;
  let ctx: CanvasRenderingContext2D;
  let game: GraphGame;
  let expression = "";
  let coordinates = "(0, 0)";

  const width = 800;
  const height = 600;

  onMount(() => {
    ctx = canvas.getContext("2d")!;
    game = new GraphGame(width, height);
    drawAxes();
  });

  function drawAxes() {
    ctx.clearRect(0, 0, width, height);

    ctx.save();

    ctx.strokeStyle = "black";
    ctx.lineWidth = 1;

    // Draw X axis
    ctx.beginPath();
    ctx.moveTo(0, height / 2);
    ctx.lineTo(width, height / 2);
    ctx.stroke();

    // Draw Y axis
    ctx.beginPath();
    ctx.moveTo(width / 2, 0);
    ctx.lineTo(width / 2, height);
    ctx.stroke();

    ctx.restore();
  }

  function drawMonsters(monsters: Array<{ x: number; y: number }>) {
    ctx.save();

    monsters.forEach((monster) => {
      const { x, y } = game.toCanvasCoords(monster.x, monster.y);
      ctx.fillStyle = "black";
      ctx.beginPath();
      ctx.arc(x, y, 5, 0, 2 * Math.PI);
      ctx.fill();
    });

    ctx.restore();
  }

  function addMonster() {
    const monsters = game.addMonster();
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
      alert("Biểu thức không hợp lệ!");
      return;
    }

    await game.animateGraph(expression, (points, monsters) => {
      drawAxes();
      drawMonsters(monsters);

      ctx.save();
      ctx.strokeStyle = "orange";
      ctx.lineWidth = 2;

      // Draw graph
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
  <div class="controls">
    <input
      type="text"
      bind:value={expression}
      placeholder="y = f(x) | Ex: x^2 + 2x + 1"
    />
    <button on:click={addMonster}>Add monster</button>
    <button on:click={startGraphAnimation}>Draw</button>
  </div>

  <canvas bind:this={canvas} {width} {height} on:mousemove={handleMouseMove}
  ></canvas>

  <div class="coordinates">{coordinates}</div>
</div>

<style>
  .container {
    display: flex;
    flex-direction: column;
    align-items: center;
    font-family: Arial, sans-serif;
  }

  canvas {
    border: 1px solid black;
  }

  .controls {
    display: flex;
    gap: 10px;
    margin: 8px;
  }

  input,
  button {
    padding: 5px;
  }

  .coordinates {
    margin: 8px;
    font-weight: bold;
  }
</style>
