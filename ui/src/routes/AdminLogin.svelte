<script>
  import { onMount } from 'svelte'
  let error = ''
  onMount(() => {
    const params = new URLSearchParams(window.location.search)
    error = params.get('error') || ''
  })
</script>

<div class="admin-login">
  <div class="stage">
    <section class="hero" aria-hidden="true">
      <div class="hero-glow"></div>
      <p class="eyebrow">Locky Control Plane</p>
      <h1>Secure Access For Admin Operations</h1>
      <p class="hero-copy">
        Manage tenants, provision users, and run workspace actions from one polished console.
      </p>
      <div class="hero-metrics">
        <article>
          <span>Tenant Ops</span>
          <strong>Fast</strong>
        </article>
        <article>
          <span>User Management</span>
          <strong>Centralized</strong>
        </article>
        <article>
          <span>Session Controls</span>
          <strong>Local</strong>
        </article>
      </div>
    </section>

    <section class="shell">
      <div class="tag"><span class="dot" aria-hidden="true"></span>Admin Login</div>
      <h2>Welcome Back</h2>
      <p class="intro">Sign in to continue to the admin workspace.</p>
      {#if error}
        <p class="err" aria-live="polite">{error}</p>
      {/if}
      <form method="POST" action="/admin/ui/login">
        <label for="username">Username</label>
        <input id="username" name="username" type="text" autocomplete="username" required />
        <label for="password">Password</label>
        <input id="password" name="password" type="password" autocomplete="current-password" required />
        <button type="submit">Sign In</button>
      </form>
    </section>
  </div>
</div>

<style>
  .admin-login {
    min-height: 100vh;
    display: grid;
    place-items: center;
    padding: 16px;
    font-family: 'Inter', 'Avenir Next', 'SF Pro Text', 'Helvetica Neue', sans-serif;
    color: #0f1f2f;
    background:
      radial-gradient(circle at 14% 12%, rgba(45, 212, 191, 0.2), transparent 30%),
      radial-gradient(circle at 86% 10%, rgba(56, 189, 248, 0.2), transparent 34%),
      linear-gradient(180deg, #f7fbff 0%, #eff5ff 100%);
  }

  .stage {
    width: min(980px, 100%);
    display: grid;
    grid-template-columns: minmax(0, 1fr) minmax(0, 420px);
    gap: 14px;
    border: 1px solid #d7e3f5;
    border-radius: 24px;
    background: rgba(255, 255, 255, 0.78);
    backdrop-filter: blur(8px);
    box-shadow: 0 24px 52px rgba(15, 23, 42, 0.14);
    overflow: hidden;
  }

  .hero {
    position: relative;
    overflow: hidden;
    padding: 34px;
    background:
      radial-gradient(circle at 6% 8%, rgba(45, 212, 191, 0.26), transparent 36%),
      linear-gradient(150deg, #10253f 0%, #102a46 44%, #13395e 100%);
    color: #e6f0fe;
  }

  .hero-glow {
    position: absolute;
    right: -90px;
    bottom: -90px;
    width: 260px;
    height: 260px;
    border-radius: 50%;
    background: rgba(52, 211, 153, 0.22);
    filter: blur(2px);
  }

  .hero h1 {
    margin: 10px 0 0;
    font-size: clamp(30px, 3vw, 40px);
    line-height: 1.08;
    letter-spacing: -0.03em;
    text-wrap: balance;
  }

  .hero-copy {
    margin: 14px 0 0;
    max-width: 40ch;
    line-height: 1.6;
    color: #c6ddf7;
  }

  .hero-metrics {
    margin-top: 26px;
    display: grid;
    gap: 8px;
  }

  .hero-metrics article {
    border: 1px solid #2f557e;
    border-radius: 12px;
    background: rgba(20, 40, 66, 0.82);
    padding: 10px 12px;
    min-width: 0;
  }

  .hero-metrics span {
    display: block;
    font-size: 12px;
    color: #9ec2e8;
  }

  .hero-metrics strong {
    display: block;
    margin-top: 4px;
    font-size: 18px;
  }

  .shell {
    background: #fcfeff;
    padding: 30px;
    display: flex;
    flex-direction: column;
    justify-content: center;
  }

  .tag {
    display: inline-flex;
    gap: 8px;
    align-items: center;
    width: fit-content;
    margin-bottom: 16px;
    border-radius: 999px;
    border: 1px solid #b8d4ef;
    background: #e5f2ff;
    color: #1e547f;
    padding: 7px 12px;
    font-size: 12px;
    letter-spacing: 0.04em;
    text-transform: uppercase;
    font-weight: 700;
  }

  .eyebrow {
    margin: 0;
    color: #9dc0e2;
    letter-spacing: 0.08em;
    text-transform: uppercase;
    font-size: 11px;
    font-weight: 700;
  }

  .dot {
    width: 8px;
    height: 8px;
    border-radius: 999px;
    background: #38bdf8;
    box-shadow: 0 0 0 4px rgba(56, 189, 248, 0.24);
  }

  h2 {
    margin: 0;
    font-size: 32px;
    line-height: 1.1;
    letter-spacing: -0.03em;
    text-wrap: balance;
  }

  .intro {
    margin: 10px 0 16px;
    color: #4d6480;
    line-height: 1.5;
  }

  label {
    display: block;
    font-size: 13px;
    margin: 14px 0 6px;
    color: #17354e;
    font-weight: 700;
  }

  input {
    width: 100%;
    border: 1px solid #c8d9ec;
    background: #fafdff;
    border-radius: 11px;
    padding: 11px 12px;
    font: inherit;
    color: inherit;
    box-sizing: border-box;
  }

  input:focus-visible {
    border-color: #38bdf8;
    outline: 3px solid rgba(56, 189, 248, 0.2);
    outline-offset: 2px;
  }

  button {
    margin-top: 20px;
    width: 100%;
    border: 1px solid transparent;
    border-radius: 11px;
    padding: 12px 14px;
    font-weight: 700;
    font: inherit;
    color: #fff;
    background: linear-gradient(135deg, #0f766e 0%, #14b8a6 100%);
    cursor: pointer;
    transition: transform 130ms ease, box-shadow 130ms ease, filter 130ms ease;
    touch-action: manipulation;
    -webkit-tap-highlight-color: rgba(20, 184, 166, 0.18);
  }

  button:hover:not(:disabled) {
    transform: translateY(-1px);
    box-shadow: 0 10px 22px rgba(15, 118, 110, 0.24);
    filter: saturate(1.06);
  }

  button:focus-visible {
    outline: 3px solid rgba(20, 184, 166, 0.24);
    outline-offset: 2px;
  }

  button:disabled {
    opacity: 0.7;
    cursor: not-allowed;
  }

  .err {
    margin: 0 0 12px;
    padding: 10px 12px;
    border-radius: 10px;
    color: #b42318;
    background: #fff2f0;
    border: 1px solid #f8c9c5;
    font-size: 13px;
  }

  @media (max-width: 920px) {
    .stage {
      grid-template-columns: 1fr;
    }

    .hero {
      display: none;
    }

    .shell {
      padding: 22px;
    }

    h2 {
      font-size: 28px;
    }
  }

  @media (prefers-reduced-motion: reduce) {
    *,
    *::before,
    *::after {
      animation: none !important;
      transition: none !important;
      scroll-behavior: auto !important;
    }
  }
</style>
