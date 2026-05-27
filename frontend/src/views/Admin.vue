<template>
  <div class="admin-container">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h2>文豪写作</h2>
        <span class="badge">管理后台</span>
      </div>
      <nav class="sidebar-nav">
        <a href="#" class="nav-item" :class="{ active: tab === 'dashboard' }" @click.prevent="tab = 'dashboard'">仪表盘</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'users' }" @click.prevent="tab = 'users'">用户管理</a>
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
        <h1>{{ tab === 'dashboard' ? '仪表盘' : '用户管理' }}</h1>
      </header>
      <div class="content">

        <div v-if="tab === 'dashboard'">
          <div class="stats">
            <div class="stat-card">
              <div class="stat-value">{{ stats.total_users }}</div>
              <div class="stat-label">注册用户</div>
            </div>
            <div class="stat-card">
              <div class="stat-value">{{ stats.active_books }}</div>
              <div class="stat-label">进行中的小说</div>
            </div>
            <div class="stat-card">
              <div class="stat-value">{{ stats.total_chapters }}</div>
              <div class="stat-label">已完成章节</div>
            </div>
            <div class="stat-card">
              <div class="stat-value">-</div>
              <div class="stat-label">今日调用</div>
            </div>
          </div>
          <div class="init-card">
            <div class="init-header">
              <h3>系统初始化</h3>
              <p class="init-desc">初始化内置题材、平台配置和 AI Agent 提示词。如果数据已存在则跳过。</p>
            </div>
            <button class="init-btn" :disabled="initializing" @click="initialize">
              {{ initializing ? '初始化中...' : '一键初始化' }}
            </button>
            <p v-if="initMsg" class="init-msg" :class="{ error: initError }">{{ initMsg }}</p>
          </div>
        </div>

        <div v-if="tab === 'users'" class="users-section">
          <div class="users-header">
            <div class="users-count">共 {{ userTotal }} 个用户</div>
          </div>
          <div class="table-wrap">
            <table class="users-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>用户名</th>
                  <th>邮箱</th>
                  <th>角色</th>
                  <th>注册时间</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="u in users" :key="u.id">
                  <td>{{ u.id }}</td>
                  <td>{{ u.username }}</td>
                  <td>{{ u.email }}</td>
                  <td>
                    <span class="role-tag" :class="u.role">{{ u.role === 'admin' ? '管理员' : '用户' }}</span>
                  </td>
                  <td>{{ formatDate(u.created_at) }}</td>
                </tr>
                <tr v-if="users.length === 0">
                  <td colspan="5" class="empty-row">暂无用户</td>
                </tr>
              </tbody>
            </table>
          </div>
          <div class="pagination" v-if="userTotal > pageSize">
            <button :disabled="page <= 1" @click="page--; loadUsers()">上一页</button>
            <span>第 {{ page }} / {{ totalPages }} 页</span>
            <button :disabled="page >= totalPages" @click="page++; loadUsers()">下一页</button>
          </div>
        </div>

      </div>
    </main>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref(null)
const tab = ref('dashboard')

const stats = ref({ total_users: 0, active_books: 0, total_chapters: 0 })
const users = ref([])
const userTotal = ref(0)
const page = ref(1)
const pageSize = 20
const initializing = ref(false)
const initMsg = ref('')
const initError = ref(false)

const totalPages = computed(() => Math.ceil(userTotal.value / pageSize) || 1)

const token = localStorage.getItem('token')
const headers = { 'Authorization': 'Bearer ' + token }

onMounted(async () => {
  const [userRes, statsRes] = await Promise.all([
    fetch('/api/v1/me', { headers }),
    fetch('/api/v1/admin/stats', { headers }),
  ])
  if (userRes.ok) user.value = await userRes.json()
  if (statsRes.ok) stats.value = await statsRes.json()
})

watch(tab, (val) => {
  if (val === 'users') loadUsers()
})

async function loadUsers() {
  const res = await fetch(`/api/v1/admin/users?page=${page.value}&page_size=${pageSize}`, { headers })
  if (res.ok) {
    const data = await res.json()
    users.value = data.users
    userTotal.value = data.total
  }
}

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit', hour: '2-digit', minute: '2-digit' })
}

async function initialize() {
  initializing.value = true
  initMsg.value = ''
  initError.value = false
  try {
    const res = await fetch('/api/v1/admin/initialize', { method: 'POST', headers })
    if (res.ok) {
      initMsg.value = '初始化成功！题材、平台和 Agent 提示词已就绪。'
    } else {
      const data = await res.json()
      initMsg.value = data.error || '初始化失败'
      initError.value = true
    }
  } catch {
    initMsg.value = '网络错误，请重试'
    initError.value = true
  } finally {
    initializing.value = false
  }
}

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
  cursor: pointer;
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
.init-card {
  background: rgba(255,255,255,0.04);
  border-radius: 12px;
  padding: 32px;
}
.init-header { margin-bottom: 20px; }
.init-header h3 { color: #e0e0e0; font-size: 18px; margin-bottom: 8px; }
.init-desc { color: #666; font-size: 13px; line-height: 1.6; }
.init-btn {
  padding: 10px 28px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #f5af19, #f12711);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s;
}
.init-btn:hover:not(:disabled) { opacity: 0.9; }
.init-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.init-msg { margin-top: 16px; font-size: 13px; color: #4caf50; }
.init-msg.error { color: #f44336; }

.users-section { }
.users-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.users-count { color: #888; font-size: 14px; }
.table-wrap {
  background: rgba(255,255,255,0.04);
  border-radius: 12px;
  overflow: hidden;
}
.users-table {
  width: 100%;
  border-collapse: collapse;
}
.users-table th {
  text-align: left;
  padding: 14px 20px;
  font-size: 13px;
  color: #888;
  border-bottom: 1px solid rgba(255,255,255,0.06);
  font-weight: 500;
}
.users-table td {
  padding: 14px 20px;
  font-size: 14px;
  color: #ccc;
  border-bottom: 1px solid rgba(255,255,255,0.03);
}
.users-table tr:last-child td { border-bottom: none; }
.users-table tr:hover td { background: rgba(255,255,255,0.02); }
.empty-row { text-align: center; color: #555; padding: 32px !important; }
.role-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.role-tag.admin { background: rgba(245,175,25,0.15); color: #f5af19; }
.role-tag.user { background: rgba(255,255,255,0.08); color: #aaa; }
.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 20px;
}
.pagination button {
  padding: 8px 16px;
  border: 1px solid rgba(255,255,255,0.1);
  border-radius: 6px;
  background: transparent;
  color: #888;
  cursor: pointer;
  font-size: 13px;
}
.pagination button:hover:not(:disabled) { color: #f5af19; border-color: #f5af19; }
.pagination button:disabled { opacity: 0.3; cursor: not-allowed; }
.pagination span { color: #888; font-size: 13px; }
</style>
