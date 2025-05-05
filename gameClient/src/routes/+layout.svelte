<script lang="ts">
  import { currentUser, clearUser } from "$lib/stores/currentUser";
  import { goto } from "$app/navigation";
  import { onMount, onDestroy } from "svelte";
  import { browser } from "$app/environment";

  let { children } = $props();
  let showUserDropdown = $state(false);
  let dropdownRef: HTMLDivElement | null = null;
  let buttonRef: HTMLButtonElement | null = null;

  function handleLogout() {
    clearUser();
    goto("/");
  }

  function toggleUserDropdown() {
    showUserDropdown = !showUserDropdown;
  }

  function handleClickOutside(event: MouseEvent) {
    if (
      showUserDropdown &&
      dropdownRef &&
      !dropdownRef.contains(event.target as Node) &&
      buttonRef &&
      !buttonRef.contains(event.target as Node)
    ) {
      showUserDropdown = false;
    }
  }

  onMount(() => {
    // Only add event listener in the browser
    if (browser) {
      window.addEventListener("click", handleClickOutside);
    }
  });

  onDestroy(() => {
    // Only remove event listener in the browser
    if (browser) {
      window.removeEventListener("click", handleClickOutside);
    }
  });
</script>

<div class="layout-wrapper">
  {#if $currentUser}
    <nav class="sidebar">
      <div class="sidebar-top">
        <div class="user-dropdown-container">
          <button
            bind:this={buttonRef}
            onclick={toggleUserDropdown}
            class="user-icon-btn"
            title="User Info"
            aria-haspopup="true"
            aria-expanded={showUserDropdown}
          >
            <!-- User Icon SVG -->
            <svg
              xmlns="http://www.w3.org/2000/svg"
              viewBox="0 0 24 24"
              fill="currentColor"
              width="24"
              height="24"
            >
              <path
                fill-rule="evenodd"
                d="M18.685 19.097A9.723 9.723 0 0 0 21.75 12c0-5.385-4.365-9.75-9.75-9.75S2.25 6.615 2.25 12a9.723 9.723 0 0 0 3.065 7.097A9.716 9.716 0 0 0 12 21.75a9.716 9.716 0 0 0 6.685-2.653Zm-12.54-1.285A7.486 7.486 0 0 1 12 15a7.486 7.486 0 0 1 5.855 2.812A8.224 8.224 0 0 1 12 20.25a8.224 8.224 0 0 1-5.855-2.438ZM15.75 9a3.75 3.75 0 1 1-7.5 0 3.75 3.75 0 0 1 7.5 0Z"
                clip-rule="evenodd"
              />
            </svg>
          </button>

          {#if showUserDropdown}
            <div bind:this={dropdownRef} class="user-dropdown-panel">
              <div class="dropdown-item user-id" title={$currentUser.id}>
                ID: {$currentUser.id}
              </div>
              <div class="dropdown-item username">
                {$currentUser.username}
              </div>
            </div>
          {/if}
        </div>
      </div>
      <div class="sidebar-bottom">
        <button onclick={handleLogout} class="logout-btn" title="Logout">
          <!-- Logout Icon SVG -->
          <svg
            xmlns="http://www.w3.org/2000/svg"
            viewBox="0 0 24 24"
            fill="currentColor"
            width="20"
            height="20"
          >
            <path
              fill-rule="evenodd"
              d="M7.5 3.75A1.5 1.5 0 0 0 6 5.25v13.5a1.5 1.5 0 0 0 1.5 1.5h6a1.5 1.5 0 0 0 1.5-1.5V15a.75.75 0 0 1 1.5 0v3.75a3 3 0 0 1-3 3h-6a3 3 0 0 1-3-3V5.25a3 3 0 0 1 3-3h6a3 3 0 0 1 3 3V9A.75.75 0 0 1 15 9V5.25a1.5 1.5 0 0 0-1.5-1.5h-6Zm10.72 4.72a.75.75 0 0 1 1.06 0l3 3a.75.75 0 0 1 0 1.06l-3 3a.75.75 0 1 1-1.06-1.06l1.72-1.72H9a.75.75 0 0 1 0-1.5h10.94l-1.72-1.72a.75.75 0 0 1 0-1.06Z"
              clip-rule="evenodd"
            />
          </svg>
        </button>
      </div>
    </nav>
  {/if}

  <main class="main-content">
    {@render children()}
  </main>
</div>

<style>
  .layout-wrapper {
    display: flex;
    min-height: 100vh;
  }

  .sidebar {
    width: 80px;
    height: 100vh;
    background-color: #343a40;
    color: #f8f9fa;
    display: flex;
    flex-direction: column;
    justify-content: space-between;
    align-items: center;
    padding: 20px 0;
    box-sizing: border-box;
    flex-shrink: 0;
    position: fixed;
    left: 0;
    top: 0;
    z-index: 10;
  }

  .sidebar-top {
    width: 100%;
    display: flex;
    justify-content: center; /* Căn giữa container dropdown */
  }

  .user-dropdown-container {
    position: relative; /* Để định vị panel dropdown */
    display: inline-block; /* Hoặc flex nếu cần */
  }

  .user-icon-btn {
    background: none;
    border: none;
    color: #ced4da; /* Màu icon mặc định */
    cursor: pointer;
    padding: 8px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition:
      background-color 0.2s ease,
      color 0.2s ease;
  }

  .user-icon-btn:hover,
  .user-icon-btn[aria-expanded="true"] {
    /* Thêm style khi mở */
    background-color: rgba(255, 255, 255, 0.1);
    color: #f8f9fa; /* Sáng hơn khi hover/active */
  }

  .user-dropdown-panel {
    display: block; /* Sẽ được kiểm soát bởi #if */
    position: absolute;
    top: 100%; /* Ngay dưới nút icon */
    transform: translateX(-10%); /* Căn giữa panel */
    background-color: #495057; /* Nền tối hơn một chút */
    border-radius: 6px;
    box-shadow: 0 4px 12px rgba(0, 0, 0, 0.2);
    padding: 10px 0; /* Padding trên dưới cho panel */
    min-width: 160px; /* Chiều rộng tối thiểu */
    z-index: 11; /* Đảm bảo panel ở trên sidebar */
    margin-top: 8px; /* Khoảng cách nhỏ từ icon */
    opacity: 1;
    visibility: visible;
    transition:
      opacity 0.1s ease,
      visibility 0.1s ease; /* Hiệu ứng mờ dần (tùy chọn) */
  }

  .dropdown-item {
    padding: 8px 15px; /* Padding cho các mục */
    font-size: 0.9em;
    white-space: nowrap;
    color: #f8f9fa;
  }

  .dropdown-item.user-id {
    color: #adb5bd;
    font-size: 0.8em;
    border-bottom: 1px solid #6c757d; /* Đường kẻ phân cách */
    padding-bottom: 8px;
    margin-bottom: 5px;
  }

  .dropdown-item.username {
    font-weight: bold;
  }

  .sidebar-bottom {
    width: 100%;
    display: flex;
    justify-content: center;
  }

  .logout-btn {
    background: none;
    border: none;
    color: #e53935;
    cursor: pointer;
    padding: 10px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    transition:
      background-color 0.2s ease,
      color 0.2s ease;
  }

  .logout-btn:hover {
    background-color: rgba(255, 255, 255, 0.1);
    color: #f44336;
  }

  .main-content {
    flex-grow: 1;
    margin-left: 80px;
    padding: 20px;
    box-sizing: border-box;
  }
</style>
