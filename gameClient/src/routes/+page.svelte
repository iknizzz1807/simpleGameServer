<script lang="ts">
  import { currentUser } from "$lib/stores/currentUser";
  import LoginForm from "$lib/components/+LoginForm.svelte";
  import ChooseGameMenu from "$lib/components/+ChooseGameMenu.svelte";
</script>

<!-- .page-content sẽ căn giữa nội dung bên trong khu vực main-content của layout -->
<div class="page-content">
  {#if $currentUser}
    <ChooseGameMenu />
  {:else}
    <LoginForm />
  {/if}
</div>

<style>
  .page-content {
    display: flex;
    flex-direction: column; /* Sắp xếp nội dung theo chiều dọc nếu cần */
    justify-content: center; /* Căn giữa theo chiều dọc */
    align-items: center; /* Căn giữa theo chiều ngang */
    flex-grow: 1; /* Đảm bảo nó cố gắng chiếm không gian trong main-content */
    padding: 20px; /* Thêm khoảng đệm */
    /* Đặt chiều cao tối thiểu để việc căn giữa dọc hoạt động tốt hơn trong một số trường hợp */
    min-height: calc(
      100vh - 150px
    ); /* Ví dụ: 100vh trừ đi chiều cao ước tính của header/footer */
    box-sizing: border-box;
  }
</style>
