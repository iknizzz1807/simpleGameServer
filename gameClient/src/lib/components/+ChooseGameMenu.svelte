<script lang="ts">
  import { goto } from "$app/navigation";
</script>

<div class="menu-container">
  <h2>Choose Your Game</h2>
  <div class="button-group">
    <button class="game-button snake" on:click={() => goto("/snake")}>
      🐍 Snake
    </button>
    <button class="game-button caro" on:click={() => goto("/caro")}>
      🏁 Caro
    </button>
    <button class="game-button graph" on:click={() => goto("/graph")}>
      📊 Graph Game
    </button>
    <!-- Thêm các nút game khác nếu cần -->
  </div>
</div>

<style>
  .menu-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 30px;
    background-color: #ffffff;
    border-radius: 8px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
    text-align: center;
  }

  h2 {
    color: #333;
    margin-bottom: 25px;
    font-size: 1.8em;
  }

  .button-group {
    display: flex;
    flex-direction: column; /* Xếp nút theo chiều dọc */
    gap: 15px; /* Khoảng cách giữa các nút */
    width: 100%;
    max-width: 300px; /* Giới hạn chiều rộng tối đa */
  }

  .game-button {
    padding: 15px 25px;
    font-size: 1.2em;
    font-weight: bold;
    border: none;
    border-radius: 6px;
    cursor: pointer;
    transition:
      transform 0.2s ease,
      box-shadow 0.2s ease;
    color: white;
    text-shadow: 1px 1px 2px rgba(0, 0, 0, 0.3);
  }

  .game-button:hover {
    transform: translateY(-3px);
    box-shadow: 0 6px 12px rgba(0, 0, 0, 0.15);
  }

  /* Màu sắc riêng cho từng nút */
  .game-button.snake {
    background-color: #4caf50; /* Green */
  }
  .game-button.snake:hover {
    background-color: #45a049;
  }

  .game-button.caro {
    background-color: #2196f3; /* Blue */
  }
  .game-button.caro:hover {
    background-color: #1e88e5;
  }

  .game-button.graph {
    background-color: #ff9800; /* Orange */
  }
  .game-button.graph:hover {
    background-color: #fb8c00;
  }
</style>
