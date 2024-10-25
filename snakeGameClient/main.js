const GRID_SIZE = 20;
const CANVAS_SIZE = 400;

const canvas = document.getElementById("gameCanvas");
const ctx = canvas.getContext("2d");

canvas.width = CANVAS_SIZE;
canvas.height = CANVAS_SIZE;

const socket = new WebSocket("ws://localhost:8080/ws");

// Game state
let players = [];
let foods = [];
let myPlayerId = null;

// Socket event handlers
socket.onopen = () => {
  console.log("Connected to server");
};

socket.onmessage = (event) => {
  const gameState = JSON.parse(event.data);

  if (gameState.type === "init") {
    myPlayerId = gameState.playerId;
    console.log("My Player ID:", myPlayerId);
  } else if (gameState.type === "gameState") {
    players = gameState.players;
    foods = gameState.foods;
    draw();
  }
};

socket.onerror = (error) => {
  console.error("WebSocket error:", error);
};

function draw() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  drawGrid();

  foods.forEach((food) => {
    drawFood(food);
  });

  Object.values(players).forEach((player) => {
    drawSnake(player);
  });

  drawScores();
}

function drawGrid() {
  ctx.strokeStyle = "#ddd";
  ctx.lineWidth = 0.5;

  for (let i = 0; i <= CANVAS_SIZE; i += GRID_SIZE) {
    ctx.beginPath();
    ctx.moveTo(i, 0);
    ctx.lineTo(i, CANVAS_SIZE);
    ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(0, i);
    ctx.lineTo(CANVAS_SIZE, i);
    ctx.stroke();
  }
}

function drawFood(food) {
  ctx.fillStyle = "red";
  ctx.beginPath();
  ctx.arc(
    food.x * GRID_SIZE + GRID_SIZE / 2,
    food.y * GRID_SIZE + GRID_SIZE / 2,
    GRID_SIZE / 2,
    0,
    Math.PI * 2
  );
  ctx.fill();
}

function drawSnake(player) {
  // Draw head
  ctx.fillStyle = player.id === myPlayerId ? "#00ff00" : "#0000ff";
  player.body.forEach((segment, index) => {
    ctx.fillRect(
      segment.x * GRID_SIZE,
      segment.y * GRID_SIZE,
      GRID_SIZE - 1,
      GRID_SIZE - 1
    );
  });
}

function drawScores() {
  ctx.fillStyle = "black";
  ctx.font = "16px Arial";
  let y = 20;
  Object.values(players).forEach((player) => {
    const text = `${player.id}: ${player.score} points`;
    ctx.fillText(text, 10, y);
    y += 20;
  });
}
// Input map
const directions = {
  w: { x: 0, y: -1 },
  s: { x: 0, y: 1 },
  a: { x: -1, y: 0 },
  d: { x: 1, y: 0 },
};

window.addEventListener("keydown", (event) => {
  const direction = directions[event.key];
  if (direction && socket.readyState === WebSocket.OPEN) {
    // Send message to the server
    socket.send(
      JSON.stringify({
        type: "direction",
        direction: direction,
      })
    );
  }
});
