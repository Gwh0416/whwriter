import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Landing from './views/Landing.vue'
import Write from './views/Write.vue'
import CreateBook from './views/CreateBook.vue'
import './style.css'

const routes = [
  { path: '/', component: Landing },
  { path: '/write', component: Write },
  { path: '/create-book', component: CreateBook },
  { path: '/radar', redirect: '/write?tab=radar' },
  { path: '/:pathMatch(.*)*', redirect: '/' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

createApp(App).use(router).mount('#app')
