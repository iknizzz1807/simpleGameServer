import { evaluate } from "mathjs";

export interface Point {
  x: number;
  y: number;
}

export interface Monster extends Point {
  ofPlayer: string; // ID of the player who owns the monster
}

export interface Player {
  id: string;
  name: string;
  score: number;
  // Add other player properties if needed
}

export class GraphGame {
  private width: number;
  private height: number;
  private monsters: Monster[] = [];
  private players: Player[] = []; // Added
  private graphs: { [playerId: string]: Point[] } = {}; // Added
  private SCALE = 20; // Pixels per unit
  private socket: WebSocket;

  constructor(width: number, height: number, socket: WebSocket) {
    this.width = width;
    this.height = height;
    this.socket = socket;
    // Initialize other properties if needed
  }

  // --- Coordinate Conversion ---
  public toCanvasCoords(x: number, y: number): Point {
    const canvasX = this.width / 2 + x * this.SCALE;
    const canvasY = this.height / 2 - y * this.SCALE; // Y is inverted
    return { x: canvasX, y: canvasY };
  }

  public fromCanvasCoords(canvasX: number, canvasY: number): Point {
    const x = (canvasX - this.width / 2) / this.SCALE;
    const y = (this.height / 2 - canvasY) / this.SCALE; // Y is inverted
    return { x, y };
  }

  // --- Game State Updates (Called from Svelte component based on WebSocket messages) ---
  public updateMonsters(newMonsters: Monster[]): void {
    this.monsters = newMonsters;
  }

  public updatePlayers(newPlayers: Player[]): void {
    this.players = newPlayers;
  }

  public updateGraphs(newGraphs: { [playerId: string]: Point[] }): void {
    this.graphs = newGraphs;
  }

  // --- Getters for Svelte component ---
  public getMonsters(): Monster[] {
    return [...this.monsters]; // Return a copy
  }

  public getPlayers(): Player[] {
    return [...this.players]; // Return a copy
  }

  public getGraphs(): { [playerId: string]: Point[] } {
    return { ...this.graphs }; // Return a copy
  }

  public getScale() {
    return this.SCALE;
  }

  // --- Game Actions ---
  public addMonster(playerId: string): Monster | null {
    // Example: Add a monster at a random *game* coordinate
    const gameX =
      Math.random() * (this.width / this.SCALE) - this.width / (2 * this.SCALE);
    const gameY =
      Math.random() * (this.height / this.SCALE) -
      this.height / (2 * this.SCALE);
    const monster: Monster = { x: gameX, y: gameY, ofPlayer: playerId };
    this.monsters.push(monster);
    console.log("Added monster locally:", monster);
    return monster;
  }

  // --- Expression Handling & Graph Generation ---
  public validateExpression(expr: string): boolean {
    // Basic check: Does it contain 'x'? More robust parsing might be needed.
    if (!expr.includes("x")) {
      return false;
    }
    try {
      // Test evaluation with a sample value
      evaluate(expr.replace(/=.*$/, "").trim(), { x: 1 }); // Remove 'y=' if present
      return true;
    } catch (error) {
      console.error("Invalid expression:", error);
      return false;
    }
  }

  public async generateGraphPoints(expr: string): Promise<Point[]> {
    const points: Point[] = [];
    const step = 0.1; // Resolution of the graph
    const startX = -this.width / (2 * this.SCALE);
    const endX = this.width / (2 * this.SCALE);

    return new Promise((resolve, reject) => {
      try {
        const compiledExpr = evaluate(`f(x) = ${expr}`); // Compile expression
        for (let x = startX; x <= endX; x += step) {
          const y = compiledExpr({ x });
          // Check for valid numbers, skip if NaN or Infinity
          if (Number.isFinite(y)) {
            points.push({ x, y });
          }
        }
        resolve(points);
      } catch (error) {
        console.error("Error evaluating expression:", error);
        reject(error);
      }
    });
  }

  // --- Animation ---
  public async animateGraph(
    expr: string,
    onStep: (currentPoints: Point[], isDone: boolean) => void
  ): Promise<void> {
    const allPoints = await this.generateGraphPoints(expr);
    let currentPoints: Point[] = [];
    const pointsPerFrame = 5; // Adjust for animation speed

    return new Promise<void>((resolve) => {
      let index = 0;
      const step = () => {
        if (index >= allPoints.length) {
          onStep(currentPoints, true); // Final step
          resolve();
          return;
        }

        // Add a chunk of points for this frame
        const endSlice = Math.min(index + pointsPerFrame, allPoints.length);
        currentPoints = allPoints.slice(0, endSlice);

        onStep(currentPoints, false); // Intermediate step

        index += pointsPerFrame;
        requestAnimationFrame(step); // Schedule next frame
      };
      requestAnimationFrame(step); // Start animation
    });
  }
}
