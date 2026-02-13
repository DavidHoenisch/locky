<script>
  import { Router, Route } from './lib/router.js'
  import Login from './routes/Login.svelte'
  import Logout from './routes/Logout.svelte'
</script>

<Router>
  <Route path="/ui/login" component={Login} />
  <Route path="/ui/logout" component={Logout} />
</Router>