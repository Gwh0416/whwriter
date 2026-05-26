<template>
  <div class="home-container">
    <div class="home-card">
      <div class="logo">
        <h1>文豪写作</h1>
        <p>AI 驱动的智能写作助手</p>
      </div>
      <div class="header">
        <h2>欢迎回来</h2>
        <button class="logout-btn" @click="logout">退出登录</button>
      </div>
      <div class="welcome-card">
        <div class="avatar">{{ user?.username?.[0] || '?' }}</div>
        <h3>{{ user?.username }}</h3>
        <p>{{ user?.email }}</p>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref(null)

onMounted(async () => {
  const token = localStorage.getItem('token')
  if (!token) {
    router.push('/login')
    return
  }
  const res = await fetch('/api/v1/me', {
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (res.status === 200) {
    user.value = await res.json()
  } else {
    localStorage.removeItem('token')
    router.push('/login')
  }
})

function logout() {
  localStorage.removeItem('token')
  router.push('/login')
}
</script>

<style scoped>
.home-container {
  min-height: 100vh;
  display: flex;
  align-items: center;
  justify-content: center;
  background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
}
.home-card {
  width: 420px;
  padding: 40px;
}
.logo {
  text-align: center;
  margin-bottom: 36px;
}
.logo h1 {
  font-size: 36px;
  font-weight: 700;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
  letter-spacing: 4px;
}
.logo p {
  color: #888;
  margin-top: 8px;
  font-size: 14px;
  letter-spacing: 2px;
}
.header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 24px;
}
.header h2 {
  font-size: 20px;
  color: #e0e0e0;
}
.logout-btn {
  padding: 8px 20px;
  border: 1px solid rgba(255,255,255,0.2);
  border-radius: 6px;
  background: transparent;
  color: #aaa;
  cursor: pointer;
  font-size: 14px;
}
.logout-btn:hover {
  color: #f12711;
  border-color: #f12711;
}
.welcome-card {
  background: rgba(255,255,255,0.06);
  border-radius: 12px;
  padding: 32px;
  text-align: center;
}
.avatar {
  width: 64px;
  height: 64px;
  border-radius: 50%;
  background: linear-gradient(135deg, #f5af19, #f12711);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 28px;
  color: #fff;
  margin: 0 auto 16px;
}
.welcome-card h3 {
  font-size: 22px;
  color: #e0e0e0;
  margin-bottom: 8px;
}
.welcome-card p {
  color: #888;
  font-size: 14px;
}
</style>
