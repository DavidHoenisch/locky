<script>
  import { currentRoute } from '../lib/router.js'
  
  let email = ''
  let password = ''
  let error = ''
  let loading = false
  
  // Get return URL from query params
  const params = new URLSearchParams(window.location.search)
  const returnTo = params.get('return_to') || '/'
  
  async function handleSubmit() {
    error = ''
    loading = true
    
    try {
      const formData = new URLSearchParams()
      formData.append('email', email)
      formData.append('password', password)
      formData.append('return_to', returnTo)
      
      const response = await fetch('/ui/login', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/x-www-form-urlencoded',
        },
        body: formData,
      })
      
      if (response.redirected) {
        window.location.href = response.url
      } else if (!response.ok) {
        const data = await response.json()
        error = data.message || 'Login failed'
      }
    } catch (err) {
      error = 'Network error. Please try again.'
    } finally {
      loading = false
    }
  }
</script>

<div class="login-container">
  <div class="login-card">
    <h1>Sign In</h1>
    <p class="subtitle">Welcome back! Please sign in to continue.</p>
    
    {#if error}
      <div class="error-message">{error}</div>
    {/if}
    
    <form on:submit|preventDefault={handleSubmit}>
      <div class="form-group">
        <label for="email">Email</label>
        <input
          type="email"
          id="email"
          bind:value={email}
          required
          placeholder="you@example.com"
          disabled={loading}
        />
      </div>
      
      <div class="form-group">
        <label for="password">Password</label>
        <input
          type="password"
          id="password"
          bind:value={password}
          required
          placeholder="Enter your password"
          disabled={loading}
        />
      </div>
      
      <button type="submit" disabled={loading}>
        {loading ? 'Signing in...' : 'Sign In'}
      </button>
    </form>
  </div>
</div>

<style>
  .login-container {
    min-height: 100vh;
    display: flex;
    align-items: center;
    justify-content: center;
    padding: 20px;
  }
  
  .login-card {
    background: white;
    padding: 40px;
    border-radius: 8px;
    box-shadow: 0 2px 10px rgba(0, 0, 0, 0.1);
    width: 100%;
    max-width: 400px;
  }
  
  h1 {
    margin-bottom: 8px;
    color: #1a1a1a;
  }
  
  .subtitle {
    color: #666;
    margin-bottom: 24px;
  }
  
  .form-group {
    margin-bottom: 20px;
  }
  
  label {
    display: block;
    margin-bottom: 8px;
    color: #333;
    font-weight: 500;
  }
  
  input {
    width: 100%;
    padding: 12px;
    border: 1px solid #ddd;
    border-radius: 4px;
    font-size: 16px;
    transition: border-color 0.2s;
  }
  
  input:focus {
    outline: none;
    border-color: #4a90d9;
  }
  
  button {
    width: 100%;
    padding: 12px;
    background: #4a90d9;
    color: white;
    border: none;
    border-radius: 4px;
    font-size: 16px;
    font-weight: 500;
    cursor: pointer;
    transition: background 0.2s;
  }
  
  button:hover:not(:disabled) {
    background: #357abd;
  }
  
  button:disabled {
    opacity: 0.6;
    cursor: not-allowed;
  }
  
  .error-message {
    background: #fee;
    color: #c33;
    padding: 12px;
    border-radius: 4px;
    margin-bottom: 20px;
  }
</style>