<!-- LoginForm.svelte -->
<script lang="ts">
  import { setUser } from "../stores/currentUser";

  let username: string = $state("");

  function handleLogin() {
    if (username.trim()) {
      const id: string = generateRandomId();
      setUser({ id, username });
    } else {
      alert("Please enter a username.");
    }
  }

  function generateRandomId(): string {
    return Math.random().toString(36).substr(2, 9);
  }
</script>

<main>
  <form action="">
    <input type="text" placeholder="Pick a name..." bind:value={username} />
    <button onclick={handleLogin} type="submit">PLAY</button>
  </form>
</main>

<style>
</style>
