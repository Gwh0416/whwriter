import { createApp } from 'vue'
import { createRouter, createWebHistory } from 'vue-router'
import App from './App.vue'
import Login from './views/Login.vue'
import Admin from './views/Admin.vue'
import Write from './views/Write.vue'
import ChangePassword from './views/ChangePassword.vue'
import './style.css'

const routes = [
  { path: '/admin', component: Admin, meta: { requiresAuth: true, role: 'admin' } },
  { path: '/write', component: Write, meta: { requiresAuth: true, role: 'user' } },
  { path: '/change-password', component: ChangePassword, meta: { requiresAuth: true } },
  { path: '/login', component: Login },
  { path: '/:pathMatch(.*)*', redirect: '/login' },
]

const router = createRouter({
  history: createWebHistory(),
  routes,
})

router.beforeEach((to, from, next) => {
  const token = localStorage.getItem('token')
  const role = localStorage.getItem('role')

  if (to.path === '/login') {
    if (token) {
      next(role === 'admin' ? '/admin' : '/write')
    } else {
      next()
    }
    return
  }

  if (!token) {
    next('/login')
    return
  }

  if (to.meta.role && to.meta.role !== role) {
    next(role === 'admin' ? '/admin' : '/write')
    return
  }

  if (to.path === '/') {
    next(role === 'admin' ? '/admin' : '/write')
    return
  }

  next()
})

createApp(App).use(router).mount('#app')
