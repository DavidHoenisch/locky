import { writable } from 'svelte/store'
export { default as Router } from './Router.svelte'
export { default as Route } from './Route.svelte'

export const currentRoute = writable(window.location.pathname)

export function navigate(path) {
  window.history.pushState({}, '', path)
  currentRoute.set(path)
}

window.addEventListener('popstate', () => {
  currentRoute.set(window.location.pathname)
})