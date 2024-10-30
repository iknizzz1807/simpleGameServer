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
      <div>
        <p>ID: {$currentUser.id}</p>
        <p>Username: {$currentUser.username}</p>
      </div>
      <button onclick={handleLogout} class="logout-btn">Logout</button>
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
    position: absolute;
    bottom: 0;
    display: flex;
    flex-direction: row;
    justify-content: space-between;
    align-items: flex-end;
  }

  .logout-btn {
    height: fit-content;
  }

  p {
    margin: 0;
  }
</style>