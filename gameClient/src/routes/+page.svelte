<script lang="ts">
  import { currentUser } from "$lib/stores/currentUser";
  import LoginForm from "$lib/components/+LoginForm.svelte";
  import ChooseGameMenu from "$lib/components/+ChooseGameMenu.svelte";
</script>

<main class="webpage">
  {#if $currentUser}
    <!-- Using this mean you let the framework subscribe to the StorageEvent
    automaticly for you, if you want to attach this store to another
    reactive state, you have to subscribe manually -->
    <ChooseGameMenu />
  {:else}
    <LoginForm />
  {/if}
</main>

<style>
  .webpage {
    display: flex;
    justify-content: center;
    align-items: center;
  }
</style>
