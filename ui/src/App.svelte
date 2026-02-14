<script>
  import { Router, Route } from './lib/router.js'
  import Login from './routes/Login.svelte'
  import Logout from './routes/Logout.svelte'
  import AdminLogin from './routes/AdminLogin.svelte'
  import Admin from './routes/Admin.svelte'
</script>

<Router>
  <Route path="/ui/login" component={Login} />
  <Route path="/ui/logout" component={Logout} />
  <Route path="/admin/ui/login" component={AdminLogin} />
  <Route path="/admin/ui" component={Admin} />
</Router>