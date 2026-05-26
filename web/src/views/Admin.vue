<template>
  <div class="admin-container">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h2>文豪写作</h2>
        <span class="badge">管理后台</span>
      </div>
      <nav class="sidebar-nav">
        <a href="#" class="nav-item active">仪表盘</a>
        <a href="#" class="nav-item">用户管理</a>
        <a href="#" class="nav-item">提示词管理</a>
        <a href="#" class="nav-item">模型配置</a>
        <a href="#" class="nav-item">系统设置</a>
      </nav>
      <div class="sidebar-footer">
        <div class="user-info">
          <div class="avatar">{{ user?.username?.[0] }}</div>
          <div>
            <div class="user-name">{{ user?.username }}</div>
            <div class="user-role">管理员</div>
          </div>
        </div>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </aside>
    <main class="main-content">
      <header class="topbar">
        <h1>仪表盘</h1>
      </header>
      <div class="content">
        <div class="stats">
          <div class="stat-card">
            <div class="stat-value">0</div>
            <div class="stat-label">注册用户</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">0</div>
            <div class="stat-label">进行中的小说</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">0</div>
            <div class="stat-label">已完成章节</div>
          </div>
          <div class="stat-card">
            <div class="stat-value">0</div>
            <div class="stat-label">今日调用</div>
          </div>
        </div>
        <div class="placeholder-card">
          <p>欢迎使用文豪写作管理后台</p>
          <p class="sub">更多管理功能即将上线</p>
        </div>
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
.admin-container {
  display: flex;
  min-height: 100vh;
  background: #0f0c29;
}
.sidebar {
  width: 240px;
  background: rgba(255,255,255,0.03);
  border-right: 1px solid rgba(255,255,255,0.06);
  display: flex;
  flex-direction: column;
  padding: 24px 0;
}
.sidebar-logo {
  padding: 0 24px 24px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.sidebar-logo h2 {
  font-size: 20px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.badge {
  font-size: 11px;
  color: #f5af19;
  border: 1px solid #f5af19;
  padding: 2px 8px;
  border-radius: 4px;
  margin-left: 8px;
}
.sidebar-nav {
  flex: 1;
  padding: 16px 0;
}
.nav-item {
  display: block;
  padding: 12px 24px;
  color: #888;
  text-decoration: none;
  font-size: 14px;
  transition: all 0.2s;
  border-left: 3px solid transparent;
}
.nav-item:hover, .nav-item.active {
  color: #e0e0e0;
  background: rgba(255,255,255,0.04);
  border-left-color: #f5af19;
}
.sidebar-footer {
  padding: 16px 24px;
  border-top: 1px solid rgba(255,255,255,0.06);
}
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.avatar {
  width: 36px;
  height: 36px;
  border-radius: 50%;
  background: linear-gradient(135deg, #f5af19, #f12711);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: #fff;
}
.user-name { font-size: 14px; color: #e0e0e0; }
.user-role { font-size: 12px; color: #888; }
.logout-btn {
  width: 100%;
  padding: 8px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 13px;
}
.logout-btn:hover { color: #f12711; border-color: #f12711; }
.main-content { flex: 1; display: flex; flex-direction: column; }
.topbar {
  padding: 20px 32px;
  border-bottom: 1px solid rgba(255,255,255,0.06);
}
.topbar h1 { font-size: 22px; color: #e0e0e0; }
.content { padding: 32px; flex: 1; }
.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}
.stat-card {
  background: rgba(255,255,255,0.04);
  border-radius: 12px;
  padding: 24px;
  text-align: center;
}
.stat-value {
  font-size: 32px;
  font-weight: 700;
  background: linear-gradient(135deg, #f5af19, #f12711);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.stat-label { font-size: 14px; color: #888; margin-top: 8px; }
.placeholder-card {
  background: rgba(255,255,255,0.04);
  border-radius: 12px;
  padding: 48px;
  text-align: center;
}
.placeholder-card p { font-size: 18px; color: #e0e0e0; }
.placeholder-card .sub { font-size: 14px; color: #666; margin-top: 8px; }
</style>
