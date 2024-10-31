// game.ts
import { get } from "svelte/store";
import { currentUser } from "$lib/stores/currentUser";
import type { User } from "$lib/stores/currentUser";

const GRID_SIZE = 20;
const CANVAS_SIZE = 600;

let canvas: HTMLCanvasElement;
let ctx: CanvasRenderingContext2D | null;

let players: any = [];
let foods: any = [];
let userID: string = "";

const socket = new WebSocket("ws://localhost:8080/snake");

export function initializeGame(canvasElement: HTMLCanvasElement) {
  canvas = canvasElement;
  ctx = canvas.getContext("2d");

  canvas.width = CANVAS_SIZE;
  canvas.height = CANVAS_SIZE;

  const user: User | null = get(currentUser);
  if (!user) {
    throw new Error("User is not logged in");
  }
  userID = user.id;

  // WebSocket event handlers
  socket.onopen = () => {
    console.log("Connected to server");

    // Send userID to server on connection
    socket.send(JSON.stringify({ type: "init", playerId: userID }));
  };

  socket.onmessage = (event) => {
    // Receive message from the server
    const gameState = JSON.parse(event.data);

    if (gameState.type === "gameState") {
      players = gameState.players; // List of all the players
      foods = gameState.foods; // List of all the food
      draw();
    }
  };

  socket.onerror = (error) => {
    console.error("WebSocket error:", error);
  };

  window.addEventListener("keydown", handleKeydown);
}

function draw() {
  ctx?.clearRect(0, 0, canvas.width, canvas.height);

  drawGrid();
  foods.forEach((food: any) => drawFood(food));
  Object.values(players).forEach((player) => drawSnake(player));
  drawScores();
}

function drawGrid() {
  if (ctx) {
    ctx.strokeStyle = "#363636";
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
}

function drawFood(food: any) {
  if (ctx) {
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
}

function drawSnake(player: any) {
  if (ctx) {
    ctx.fillStyle = player.id === userID ? "#00ff00" : "#0000ff";
    player.body.forEach((segment: any) => {
      ctx?.fillRect(
        segment.x * GRID_SIZE,
        segment.y * GRID_SIZE,
        GRID_SIZE - 1,
        GRID_SIZE - 1
      );
    });
  }
}

function drawScores() {
  if (ctx) {
    ctx.fillStyle = "black";
    ctx.font = "16px Arial";
    let y = 20;
    Object.values(players).forEach((player: any) => {
      const text = `${player.id}: ${player.score} points`;
      ctx?.fillText(text, 10, y);
      y += 20;
    });
  }
}

const directions: Record<string, { x: number; y: number }> = {
  w: { x: 0, y: -1 },
  s: { x: 0, y: 1 },
  a: { x: -1, y: 0 },
  d: { x: 1, y: 0 },
};

function handleKeydown(event: KeyboardEvent) {
  const direction = directions[event.key];
  if (direction && socket.readyState === WebSocket.OPEN) {
    socket.send(JSON.stringify({ type: "direction", direction }));
  }
}
