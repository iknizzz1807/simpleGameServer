let canvas: HTMLCanvasElement;
let playerStatusTag: HTMLElement;
let ctx: CanvasRenderingContext2D;

const BOARD_SIZE = 15;
const CELL_SIZE = 40;
const WIN_CONDITION = 5;

type Player = "X" | "O";
let currentPlayer: Player = "X";
let endGame: boolean = false;
const board: (Player | null)[][] = Array.from({ length: BOARD_SIZE }, () =>
  Array(BOARD_SIZE).fill(null)
);

export function initializeGame(
  canvasElement: HTMLCanvasElement,
  statusTag: HTMLElement
) {
  canvas = canvasElement;
  playerStatusTag = statusTag;
  ctx = canvas.getContext("2d")!;

  canvas.width = BOARD_SIZE * CELL_SIZE;
  canvas.height = BOARD_SIZE * CELL_SIZE;

  drawBoard();
}

function drawBoard() {
  ctx.clearRect(0, 0, canvas.width, canvas.height);
  ctx.strokeStyle = "#000";

  for (let i = 0; i <= BOARD_SIZE; i++) {
    ctx.beginPath();
    ctx.moveTo(i * CELL_SIZE, 0);
    ctx.lineTo(i * CELL_SIZE, canvas.height);
    ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(0, i * CELL_SIZE);
    ctx.lineTo(canvas.width, i * CELL_SIZE);
    ctx.stroke();
  }
  setPlayerStatus(currentPlayer, "turn");
}

function drawMark(x: number, y: number, player: Player) {
  const halfSize = CELL_SIZE / 2;
  const centerX = x * CELL_SIZE + halfSize;
  const centerY = y * CELL_SIZE + halfSize;

  ctx.strokeStyle = player === "X" ? "red" : "blue";
  ctx.lineWidth = 1;

  if (player === "X") {
    ctx.beginPath();
    ctx.moveTo(centerX - 10, centerY - 10);
    ctx.lineTo(centerX + 10, centerY + 10);
    ctx.stroke();

    ctx.beginPath();
    ctx.moveTo(centerX + 10, centerY - 10);
    ctx.lineTo(centerX - 10, centerY + 10);
    ctx.stroke();
  } else {
    ctx.beginPath();
    ctx.arc(centerX, centerY, 10, 0, Math.PI * 2);
    ctx.stroke();
  }
}

function drawWinningLine(coords: [number, number][]) {
  if (coords.length < 2) return;

  ctx.strokeStyle = "green";
  ctx.lineWidth = 1;
  ctx.beginPath();

  const [startX, startY] = coords[0];
  const [endX, endY] = coords[coords.length - 1];

  ctx.moveTo(
    startX * CELL_SIZE + CELL_SIZE / 2,
    startY * CELL_SIZE + CELL_SIZE / 2
  );
  ctx.lineTo(
    endX * CELL_SIZE + CELL_SIZE / 2,
    endY * CELL_SIZE + CELL_SIZE / 2
  );
  ctx.stroke();
}

export function handleCanvasClick(event: MouseEvent) {
  const rect = canvas.getBoundingClientRect();
  const x = Math.floor((event.clientX - rect.left) / CELL_SIZE);
  const y = Math.floor((event.clientY - rect.top) / CELL_SIZE);

  if (board[y][x] === null && !endGame) {
    board[y][x] = currentPlayer;
    drawMark(x, y, currentPlayer);

    const [hasWon, winningCoords] = checkWin(x, y, currentPlayer);
    if (hasWon) {
      setPlayerStatus(currentPlayer, "win");
      drawWinningLine(winningCoords);
      endGame = true;
      return;
    }

    currentPlayer = currentPlayer === "X" ? "O" : "X";
    setPlayerStatus(currentPlayer, "turn");
  }
}

function checkWin(
  x: number,
  y: number,
  player: Player
): [boolean, [number, number][]] {
  const directions = [
    { dx: 1, dy: 0 },
    { dx: 0, dy: 1 },
    { dx: 1, dy: 1 },
    { dx: 1, dy: -1 },
  ];

  for (const { dx, dy } of directions) {
    const winningCoords = getWinningCoordinates(x, y, player, dx, dy);
    if (winningCoords.length >= WIN_CONDITION) {
      return [true, winningCoords];
    }
  }

  return [false, []];
}

function getWinningCoordinates(
  x: number,
  y: number,
  player: Player,
  dx: number,
  dy: number
): [number, number][] {
  let coords: [number, number][] = [[x, y]];

  coords = coords.concat(getConsecutiveCoordinates(x, y, player, dx, dy));
  coords = coords.concat(getConsecutiveCoordinates(x, y, player, -dx, -dy));

  return coords;
}

function getConsecutiveCoordinates(
  x: number,
  y: number,
  player: Player,
  dx: number,
  dy: number
): [number, number][] {
  let coords: [number, number][] = [];
  let nx = x + dx;
  let ny = y + dy;

  while (
    nx >= 0 &&
    ny >= 0 &&
    nx < BOARD_SIZE &&
    ny < BOARD_SIZE &&
    board[ny][nx] === player
  ) {
    coords.push([nx, ny]);
    nx += dx;
    ny += dy;
  }

  return coords;
}

export function setPlayerStatus(player: Player, type: string) {
  if (playerStatusTag) {
    if (type === "turn") {
      playerStatusTag.textContent = `${currentPlayer}'s turn`;
    } else if (type === "win") {
      playerStatusTag.textContent = `${currentPlayer} won!`;
    }
  }
}

export function resetGame() {
  for (let y = 0; y < BOARD_SIZE; y++) {
    for (let x = 0; x < BOARD_SIZE; x++) {
      board[y][x] = null;
    }
  }
  currentPlayer = "X";
  endGame = false;
  drawBoard();
}
