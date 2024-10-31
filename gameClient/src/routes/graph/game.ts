import { evaluate } from "mathjs";

export interface Monster {
  x: number;
  y: number;
  type: "player" | "other";
}

export class GraphGame {
  private width: number;
  private height: number;
  private monsters: Monster[] = [];
  private SCALE = 20;
  private currentX: number;
  private socket: WebSocket;

  constructor(width: number, height: number, socket: WebSocket) {
    this.width = width;
    this.height = height;
    this.currentX = this.getInitialX();
    this.socket = socket;

    this.socket.onmessage = (event) => {
      const data = JSON.parse(event.data);
      if (data.type === "gameState") {
        this.monsters = data.monsters;
      }
    };
  }

  private getInitialX(): number {
    return -this.width / 2 / this.SCALE;
  }

  private getFinalX(): number {
    return this.width / 2 / this.SCALE;
  }

  public toCanvasCoords(x: number, y: number) {
    const canvasX = this.width / 2 + x * this.SCALE;
    const canvasY = this.height / 2 - y * this.SCALE;
    return { x: canvasX, y: canvasY };
  }

  public fromCanvasCoords(canvasX: number, canvasY: number) {
    const x = (canvasX - this.width / 2) / this.SCALE;
    const y = (this.height / 2 - canvasY) / this.SCALE;
    return { x, y };
  }

  public addMonster(type: "player" | "other") {
    const x = Math.floor(Math.random() * 21) - 10;
    const y = Math.floor(Math.random() * 21) - 10;
    const monster = { x, y, type };
    this.monsters.push(monster);
    this.socket.send(JSON.stringify({ type: "addMonster", monster }));
    return [...this.monsters];
  }

  public getMonsters() {
    return [...this.monsters];
  }

  public isVerticalLine(expr: string): { isVertical: boolean; x: number } {
    const match = expr.match(/^\s*x\s*=\s*(-?\d*\.?\d+)\s*$/);
    if (match) {
      return { isVertical: true, x: parseFloat(match[1]) };
    }
    return { isVertical: false, x: 0 };
  }

  public validateExpression(expr: string): boolean {
    if (this.isVerticalLine(expr).isVertical) {
      return true;
    }

    try {
      evaluate(expr, { x: 0 });
      return true;
    } catch (error) {
      return false;
    }
  }

  private checkCollision(x: number, y: number) {
    let collisionOccurred = false;
    this.monsters = this.monsters.filter((monster) => {
      const { x: mx, y: my } = this.toCanvasCoords(monster.x, monster.y);
      const distance = Math.hypot(mx - x, my - y);
      return distance >= 5;
    });
    return collisionOccurred;
  }

  public async animateVerticalLine(
    x: number,
    onStep: (points: { x: number; y: number }[], monsters: Monster[]) => void
  ) {
    const points: { x: number; y: number }[] = [];
    const canvasX = this.toCanvasCoords(x, 0).x;

    for (let y = this.height; y >= 0; y -= 5) {
      points.push({ x: canvasX, y });
      this.checkCollision(canvasX, y);
      onStep(points, [...this.monsters]);
      await new Promise((resolve) => setTimeout(resolve, 10));
    }
  }

  public async animateGraph(
    expr: string,
    onStep: (points: { x: number; y: number }[], monsters: Monster[]) => void
  ) {
    const verticalCheck = this.isVerticalLine(expr);
    if (verticalCheck.isVertical) {
      await this.animateVerticalLine(verticalCheck.x, onStep);
      return;
    }

    this.currentX = this.getInitialX();
    const points: { x: number; y: number }[] = [];

    return new Promise<void>((resolve) => {
      const step = () => {
        if (this.currentX > this.getFinalX()) {
          resolve();
          return;
        }

        try {
          const y = evaluate(expr, { x: this.currentX });
          const { x: canvasX, y: canvasY } = this.toCanvasCoords(
            this.currentX,
            y
          );

          points.push({ x: canvasX, y: canvasY });
          this.checkCollision(canvasX, canvasY);

          onStep(points, [...this.monsters]);

          this.currentX += 0.1;
          requestAnimationFrame(step);
        } catch (error) {
          resolve();
        }
      };

      step();
      // Next player's turn
    });
  }
}
