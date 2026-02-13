<script>
  import { onMount } from 'svelte'
  
  onMount(async () => {
    try {
      await fetch('/ui/logout', {
        method: 'POST',
        credentials: 'include',
      })
    } catch (err) {
      console.error('Logout error:', err)
    }
  })
</script>

<div class="logout-container">
  <div class="logout-card">
    <h1>Signed Out</h1>
    <p>You have been successfully signed out.</p>
    <a href="/ui/login">Sign in again</a>
  </div>
</div>

<style>
  .logout-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }
  
  .logout-card {
    background: white;
    padding: 40px;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    text-align: center;
  }
  
  h1 {
    margin-bottom: 16px;
    color: #1a1a1a;
  }
  
  p {
    color: #666;
    margin-bottom: 24px;
  }
  
  a {
    color: #4a90d9;
    text-decoration: none;
    font-weight: 500;
  }
  
  a:hover {
    text-decoration: underline;
  }
</style>