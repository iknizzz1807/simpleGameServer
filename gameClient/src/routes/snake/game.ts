import { get } from "svelte/store";
import { currentUser } from "$lib/stores/currentUser";
import type { User } from "$lib/stores/currentUser";
import { messages, playersStore, numOfPlayers } from "./store";

const NUM_CELLS = 30;
const CANVAS_SIZE = 600; // Kích thước canvas tổng thể
const GRID_SIZE = CANVAS_SIZE / NUM_CELLS;

let canvas: HTMLCanvasElement;
let ctx: CanvasRenderingContext2D | null;

// let players: any = {}; // Không cần biến cục bộ này nữa nếu dùng store
let foods: any = [];
let userID: string = "";

let socket: WebSocket | null = null; // Khởi tạo là null

export function disconnect() {
  if (socket && socket.readyState === WebSocket.OPEN) {
    console.log("Disconnecting WebSocket...");
    socket.close();
  }
  window.removeEventListener("keydown", handleKeydown); // Gỡ listener khi disconnect
  // Reset stores
  messages.set([]);
  numOfPlayers.set(0);
  playersStore.set({});
}

export function initializeGame(canvasElement: HTMLCanvasElement) {
  // Nếu đã có socket đang mở, đóng nó trước khi tạo mới
  if (socket && socket.readyState === WebSocket.OPEN) {
    socket.close();
  }

  canvas = canvasElement;
  ctx = canvas.getContext("2d");
  if (!ctx) {
    console.error("Failed to get 2D context");
    return;
  }

  canvas.width = CANVAS_SIZE;
  canvas.height = CANVAS_SIZE;

  const user: User | null = get(currentUser);
  if (!user) {
    messages.set(["Error: User not logged in. Please refresh."]);
    console.error("User is not logged in");
    return; // Ngăn không kết nối socket
  }
  userID = user.id;

  // Tạo kết nối WebSocket mới
  socket = new WebSocket("ws://localhost:8080/snake");

  // WebSocket event handlers
  socket.onopen = () => {
    console.log("Connected to Snake server");
    messages.set(["Connected to server."]); // Reset messages on new connection
    numOfPlayers.set(0);
    playersStore.set({});
    // Send userID to server on connection
    socket?.send(JSON.stringify({ type: "init", playerId: userID }));
  };

  socket.onmessage = (event) => {
    try {
      const data = JSON.parse(event.data);

      if (data.type === "gameState") {
        // players = data.players; // Cập nhật store thay vì biến cục bộ
        playersStore.set(data.players || {}); // Cập nhật store người chơi
        foods = data.foods || []; // Cập nhật food
        draw(); // Vẽ lại game state
      } else if (data.type === "playerJoinedOrLeave") {
        // Cập nhật messages và số người chơi từ store
        messages.update((currentMessages) => [
          ...currentMessages,
          ...(data.message || []),
        ]);
        numOfPlayers.set(data.totalPlayer ?? 0);
      } else if (data.type === "error") {
        console.error("Server error:", data.message);
        messages.update((m) => [...m, `Server Error: ${data.message}`]);
      }
    } catch (error) {
      console.error("Failed to parse message or handle update:", error);
      messages.update((m) => [...m, "Error processing server message."]);
    }
  };

  socket.onerror = (error) => {
    console.error("WebSocket error:", error);
    messages.update((m) => [...m, "WebSocket connection error."]);
  };

  socket.onclose = (event) => {
    console.log("WebSocket closed:", event.reason);
    messages.update((m) => [...m, "Disconnected from server."]);
    // Không cần gọi disconnect() ở đây vì nó có thể gây vòng lặp nếu close do lỗi
    // Reset stores nếu cần
    numOfPlayers.set(0);
    playersStore.set({});
  };

  // Gỡ listener cũ trước khi thêm mới (phòng trường hợp re-initialize)
  window.removeEventListener("keydown", handleKeydown);
  window.addEventListener("keydown", handleKeydown);
}

function draw() {
  if (!ctx) return;
  ctx.clearRect(0, 0, canvas.width, canvas.height);

  drawGrid();
  foods.forEach((food: any) => drawFood(food));

  // Lấy players từ store để vẽ
  const currentPlayers = get(playersStore);
  Object.values(currentPlayers).forEach((player) => drawSnake(player));
}

function drawGrid() {
  if (!ctx) return;
  ctx.strokeStyle = "#E0E0E0";
  ctx.lineWidth = 1; // Giữ nguyên độ dày đường kẻ
  // Vòng lặp vẽ grid không cần thay đổi, vì nó dựa trên CANVAS_SIZE và GRID_SIZE
  for (let i = 0; i <= CANVAS_SIZE; i += GRID_SIZE) {
    ctx.beginPath();
    ctx.moveTo(i + 0.5, 0);
    ctx.lineTo(i + 0.5, CANVAS_SIZE);
    ctx.stroke();
    ctx.beginPath();
    ctx.moveTo(0, i + 0.5);
    ctx.lineTo(CANVAS_SIZE, i + 0.5);
    ctx.stroke();
  }
}

function drawFood(food: any) {
  if (!ctx) return;
  ctx.fillStyle = "#E53935";
  ctx.beginPath();
  // Tính toán vị trí và kích thước dựa trên GRID_SIZE mới
  ctx.arc(
    food.x * GRID_SIZE + GRID_SIZE / 2,
    food.y * GRID_SIZE + GRID_SIZE / 2,
    GRID_SIZE / 2.5, // Kích thước tương đối với ô
    0,
    Math.PI * 2
  );
  ctx.fill();

  ctx.fillStyle = "#7CB342";
  // Tính toán vị trí và kích thước cuống dựa trên GRID_SIZE mới
  const stemWidth = Math.max(1, GRID_SIZE / 7); // Đảm bảo cuống không quá nhỏ
  const stemHeight = Math.max(2, GRID_SIZE / 4);
  ctx.fillRect(
    food.x * GRID_SIZE + GRID_SIZE / 2 - stemWidth / 2,
    food.y * GRID_SIZE + GRID_SIZE / 2 - GRID_SIZE / 2, // Vị trí tương đối
    stemWidth,
    stemHeight
  );
}

function drawSnake(player: any) {
  if (!ctx || !player || !player.body) return;

  const isCurrentUser = player.id === userID;
  const snakeColor = isCurrentUser ? "#66BB6A" : "#42A5F5";
  const borderColor = isCurrentUser ? "#388E3C" : "#1E88E5";

  ctx.fillStyle = snakeColor;
  ctx.strokeStyle = borderColor;
  ctx.lineWidth = Math.max(1, GRID_SIZE / 10); // Độ dày viền tương đối

  player.body.forEach((segment: any, index: number) => {
    const x = segment.x * GRID_SIZE;
    const y = segment.y * GRID_SIZE;
    if (!ctx) return;

    // Vẽ thân rắn dựa trên GRID_SIZE mới
    const cornerRadius = Math.max(1, GRID_SIZE / 5); // Bo góc tương đối
    roundRect(ctx, x + 1, y + 1, GRID_SIZE - 2, GRID_SIZE - 2, cornerRadius);

    if (index === 0) {
      // Vẽ mắt dựa trên GRID_SIZE mới
      ctx.fillStyle = "#FFFFFF";
      const eyeSize = Math.max(1, GRID_SIZE / 5);
      const eyeBaseX = x + GRID_SIZE / 2;
      const eyeBaseY = y + GRID_SIZE / 3;
      const eyeDist = Math.max(1, GRID_SIZE / 4);

      const eyeX1 = eyeBaseX - eyeDist;
      const eyeY1 = eyeBaseY;
      const eyeX2 = eyeBaseX + eyeDist;
      const eyeY2 = eyeBaseY;

      ctx.beginPath();
      ctx.arc(eyeX1, eyeY1, eyeSize, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.arc(eyeX2, eyeY2, eyeSize, 0, Math.PI * 2);
      ctx.fill();

      ctx.fillStyle = "#000000";
      const pupilSize = Math.max(1, eyeSize / 2);
      ctx.beginPath();
      ctx.arc(eyeX1, eyeY1, pupilSize, 0, Math.PI * 2);
      ctx.fill();
      ctx.beginPath();
      ctx.arc(eyeX2, eyeY2, pupilSize, 0, Math.PI * 2);
      ctx.fill();

      ctx.fillStyle = snakeColor;
    }
  });
}

// --- BỎ HÀM NÀY ---
// function drawScores() { ... }

const directions: Record<string, { x: number; y: number }> = {
  w: { x: 0, y: -1 },
  s: { x: 0, y: 1 },
  a: { x: -1, y: 0 },
  d: { x: 1, y: 0 },
  W: { x: 0, y: -1 }, // Hỗ trợ cả chữ hoa
  S: { x: 0, y: 1 },
  A: { x: -1, y: 0 },
  D: { x: 1, y: 0 },
  ArrowUp: { x: 0, y: -1 }, // Hỗ trợ phím mũi tên
  ArrowDown: { x: 0, y: 1 },
  ArrowLeft: { x: -1, y: 0 },
  ArrowRight: { x: 1, y: 0 },
};

function handleKeydown(event: KeyboardEvent) {
  const direction = directions[event.key];
  if (direction && socket && socket.readyState === WebSocket.OPEN) {
    // Gửi yêu cầu thay đổi hướng
    socket.send(JSON.stringify({ type: "direction", direction }));
    // Ngăn hành vi mặc định của phím mũi tên (cuộn trang)
    if (event.key.startsWith("Arrow")) {
      event.preventDefault();
    }
  }
}

// Helper function to draw rounded rectangles (for snake body)
function roundRect(
  ctx: CanvasRenderingContext2D,
  x: number,
  y: number,
  width: number,
  height: number,
  radiusInput:
    | number
    | { tl?: number; tr?: number; br?: number; bl?: number } = 5
) {
  // Đảm bảo radius là object với các giá trị mặc định là 0 nếu không được cung cấp
  let radius: { tl: number; tr: number; br: number; bl: number };
  if (typeof radiusInput === "number") {
    radius = {
      tl: radiusInput,
      tr: radiusInput,
      br: radiusInput,
      bl: radiusInput,
    };
  } else {
    radius = {
      tl: radiusInput?.tl ?? 0, // Sử dụng ?? để gán 0 nếu undefined hoặc null
      tr: radiusInput?.tr ?? 0,
      br: radiusInput?.br ?? 0,
      bl: radiusInput?.bl ?? 0,
    };
  }

  // Đảm bảo radius không lớn hơn một nửa chiều rộng/cao
  radius.tl = Math.min(radius.tl, width / 2, height / 2);
  radius.tr = Math.min(radius.tr, width / 2, height / 2);
  radius.br = Math.min(radius.br, width / 2, height / 2);
  radius.bl = Math.min(radius.bl, width / 2, height / 2);

  ctx.beginPath();
  ctx.moveTo(x + radius.tl, y);
  ctx.lineTo(x + width - radius.tr, y);
  ctx.quadraticCurveTo(x + width, y, x + width, y + radius.tr);
  ctx.lineTo(x + width, y + height - radius.br);
  ctx.quadraticCurveTo(
    x + width,
    y + height,
    x + width - radius.br,
    y + height
  );
  ctx.lineTo(x + radius.bl, y + height);
  ctx.quadraticCurveTo(x, y + height, x, y + height - radius.bl);
  ctx.lineTo(x, y + radius.tl);
  ctx.quadraticCurveTo(x, y, x + radius.tl, y);
  ctx.closePath();
  ctx.fill(); // Fill the rectangle
  ctx.stroke(); // Draw the border
}
