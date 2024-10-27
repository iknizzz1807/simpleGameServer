<script lang="ts">
  import { currentUser, clearUser } from "$lib/stores/currentUser";
  import { goto } from "$app/navigation";

  let { children } = $props();
  function handleLogout() {
    clearUser();
    goto("/");
  }
</script>

<main class="container">
  {@render children()}
  <div class="footer">
    {#if $currentUser}
      <p>ID: {$currentUser.id}</p>
      <p>Username: {$currentUser.username}</p>
      <button onclick={handleLogout}>Logout</button>
    {/if}
  </div>
</main>

<style>
  .container {
    height: 100vh;
    display: flex;
    flex-direction: column;
    justify-content: center;
    align-items: center;
  }

  .footer {
    width: 100%;
    text-align: center;
    position: absolute;
    bottom: 0;
  }
</style>