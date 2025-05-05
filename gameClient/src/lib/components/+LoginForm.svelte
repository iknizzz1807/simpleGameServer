<script lang="ts">
  import { setUser } from "../stores/currentUser";

  let username: string = $state("");

  function handleLogin(event: Event) {
    event.preventDefault(); // Ngăn form submit theo cách truyền thống
    if (username.trim()) {
      const id: string = generateRandomId();
      setUser({ id, username: username.trim() }); // Trim username trước khi lưu
    } else {
      alert("Please enter a username.");
    }
  }

  function generateRandomId(): string {
    // Tạo ID ngẫu nhiên an toàn hơn một chút
    return Date.now().toString(36) + Math.random().toString(36).substring(2, 7);
  }
</script>

<div class="login-container">
  <h2>Welcome!</h2>
  <form on:submit={handleLogin} class="login-form">
    <input
      type="text"
      placeholder="Enter your nickname..."
      bind:value={username}
      required
      class="username-input"
      maxlength="20"
    />
    <button type="submit" class="play-button">PLAY</button>
  </form>
</div>

<style>
  .login-container {
    display: flex;
    flex-direction: column;
    align-items: center;
    padding: 40px 30px;
    background-color: #fff;
    border-radius: 8px;
    box-shadow: 0 4px 15px rgba(0, 0, 0, 0.1);
    text-align: center;
  }

  h2 {
    color: #333;
    margin-bottom: 25px;
    font-size: 1.8em;
  }

  .login-form {
    display: flex;
    flex-direction: column; /* Hoặc row nếu muốn input và button cùng hàng */
    gap: 15px;
    width: 100%;
    max-width: 350px;
  }

  .username-input {
    padding: 12px 15px;
    font-size: 1.1em;
    border: 1px solid #ccc;
    border-radius: 4px;
    box-sizing: border-box; /* Đảm bảo padding không làm tăng kích thước */
    transition: border-color 0.2s ease;
  }

  .username-input:focus {
    border-color: #2196f3; /* Highlight khi focus */
    outline: none;
  }

  .play-button {
    padding: 12px 20px;
    font-size: 1.1em;
    font-weight: bold;
    background-color: #4caf50; /* Green */
    color: white;
    border: none;
    border-radius: 4px;
    cursor: pointer;
    transition: background-color 0.2s ease;
  }

  .play-button:hover {
    background-color: #45a049;
  }
</style>
