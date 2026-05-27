<template>
  <div class="write-container">
    <header class="topbar">
      <div class="topbar-left">
        <h1>文豪写作</h1>
      </div>
      <div class="topbar-right">
        <router-link to="/change-password" class="cp-link">修改密码</router-link>
        <span class="user-name">{{ user?.username }}</span>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </header>
    <main class="main-content">
      <div class="welcome-card">
        <div class="welcome-icon">✍️</div>
        <h2>开始你的创作之旅</h2>
        <p>AI 驱动的智能写作助手，帮你完成从构思到成稿的全流程</p>
        <div class="actions">
          <router-link to="/create-book" class="action-btn primary">创建新小说</router-link>
          <button class="action-btn">导入大纲</button>
        </div>
      </div>
      <div class="empty-state">
        <p>还没有小说？点击上方按钮开始创作</p>
      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref(null)

onMounted(async () => {
  const token = localStorage.getItem('token')
  const res = await fetch('/api/v1/me', {
    headers: { 'Authorization': 'Bearer ' + token },
  })
  if (res.status === 200) {
    user.value = await res.json()
  }
})

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<style scoped>
.write-container {
  min-height: 100vh;
  background: linear-gradient(135deg, #0f0c29, #302b63, #24243e);
  display: flex;
  flex-direction: column;
}
.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 32px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.topbar-left h1 {
  font-size: 22px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}
.user-name { color: #aaa; font-size: 14px; }
.cp-link {
  color: #888;
  font-size: 13px;
  text-decoration: none;
  transition: color 0.2s;
}
.cp-link:hover { color: #f5af19; }
.logout-btn {
  padding: 6px 16px;
  border: 1px solid rgba(255,255,255,0.15);
  border-radius: 6px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 13px;
}
.logout-btn:hover { color: #f12711; border-color: #f12711; }
.main-content {
  flex: 1;
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 40px;
}
.welcome-card {
  text-align: center;
  margin-bottom: 40px;
}
.welcome-icon { font-size: 48px; margin-bottom: 16px; }
.welcome-card h2 { font-size: 28px; color: #e0e0e0; margin-bottom: 12px; }
.welcome-card p { color: #888; font-size: 15px; max-width: 400px; }
.actions {
  display: flex;
  gap: 16px;
  justify-content: center;
  margin-top: 28px;
}
.action-btn {
  padding: 12px 32px;
  border-radius: 8px;
  font-size: 15px;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid rgba(255,255,255,0.15);
  background: transparent;
  color: #aaa;
  text-decoration: none;
  display: inline-block;
}
.action-btn:hover { border-color: #f5af19; color: #f5af19; }
.action-btn.primary {
  background: linear-gradient(135deg, #f5af19, #f12711);
  border: none;
  color: #fff;
  font-weight: 600;
}
.action-btn.primary:hover { opacity: 0.9; }
.empty-state {
  color: #555;
  font-size: 14px;
}
</style>
