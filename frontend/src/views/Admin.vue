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
        <a href="#" class="nav-item" :class="{ active: tab === 'genres' }" @click.prevent="tab = 'genres'; loadGenres()">题材管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'platforms' }" @click.prevent="tab = 'platforms'; loadPlatforms()">平台管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'settings' }" @click.prevent="tab = 'settings'">系统设置</a>
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
        <h1>{{ tabTitle }}</h1>
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
        </div>

        <div v-if="tab === 'users'" class="data-section">
          <div class="section-header">
            <div class="section-count">共 {{ userTotal }} 个用户</div>
          </div>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>用户名</th>
                  <th>邮箱</th>
                  <th>角色</th>
                  <th>余额</th>
                  <th>状态</th>
                  <th>注册时间</th>
                  <th>操作</th>
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
                  <td>{{ (u.balance || 0).toLocaleString() }}</td>
                  <td>
                    <span class="status-tag" :class="(u.status || 'active') === 'active' ? 'active' : 'inactive'">
                      {{ (u.status || 'active') === 'active' ? '正常' : '已禁用' }}
                    </span>
                  </td>
                  <td>{{ formatDate(u.created_at) }}</td>
                  <td>
                    <div class="action-group">
                      <button
                        class="action-btn"
                        :class="(u.status || 'active') === 'active' ? 'danger' : ''"
                        @click="toggleUserStatus(u)"
                      >{{ (u.status || 'active') === 'active' ? '禁用' : '启用' }}</button>
                      <button class="action-btn" @click="openRechargeModal(u)">充值</button>
                    </div>
                  </td>
                </tr>
                <tr v-if="users.length === 0">
                  <td colspan="8" class="empty-row">暂无用户</td>
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

        <div v-if="tab === 'genres'" class="data-section">
          <div class="section-header">
            <div class="section-count">共 {{ genres.length }} 个题材</div>
            <button class="add-btn" @click="openGenreModal()">+ 新增题材</button>
          </div>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>内容</th>
                  <th>排序</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in genres" :key="g.id">
                  <td>{{ g.id }}</td>
                  <td>{{ g.name }}</td>
                  <td>
                    <button class="view-btn" @click="viewGenreContent(g)">查看</button>
                  </td>
                  <td>{{ g.sort_order }}</td>
                  <td>
                    <span class="status-tag" :class="g.is_active ? 'active' : 'inactive'">{{ g.is_active ? '启用' : '禁用' }}</span>
                  </td>
                  <td>
                    <button class="action-btn" @click="openGenreModal(g)">编辑</button>
                    <button class="action-btn danger" @click="deleteGenre(g.id)">删除</button>
                  </td>
                </tr>
                <tr v-if="genres.length === 0">
                  <td colspan="6" class="empty-row">暂无题材</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="tab === 'platforms'" class="data-section">
          <div class="section-header">
            <div class="section-count">共 {{ platforms.length }} 个平台</div>
            <button class="add-btn" @click="openPlatformModal()">+ 新增平台</button>
          </div>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>ID</th>
                  <th>名称</th>
                  <th>排序</th>
                  <th>状态</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="p in platforms" :key="p.id">
                  <td>{{ p.id }}</td>
                  <td>{{ p.name }}</td>
                  <td>{{ p.sort_order }}</td>
                  <td>
                    <span class="status-tag" :class="p.is_active ? 'active' : 'inactive'">{{ p.is_active ? '启用' : '禁用' }}</span>
                  </td>
                  <td>
                    <button class="action-btn" @click="openPlatformModal(p)">编辑</button>
                    <button class="action-btn danger" @click="deletePlatform(p.id)">删除</button>
                  </td>
                </tr>
                <tr v-if="platforms.length === 0">
                  <td colspan="5" class="empty-row">暂无平台</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="tab === 'settings'" class="data-section">
          <div class="init-card">
            <div class="init-header">
              <h3>系统初始化</h3>
              <p class="init-desc">初始化内置题材、平台配置。如果数据已存在则跳过。</p>
            </div>
            <button class="init-btn" :disabled="initializing" @click="initialize">
              {{ initializing ? '初始化中...' : '一键初始化' }}
            </button>
            <p v-if="initMsg" class="init-msg" :class="{ error: initError }">{{ initMsg }}</p>
          </div>
        </div>

      </div>
    </main>

    <div v-if="showGenreModal" class="modal-overlay" @click.self="showGenreModal = false">
      <div class="modal">
        <h3>{{ editingGenre ? '编辑题材' : '新增题材' }}</h3>
        <div class="form-group">
          <label>名称</label>
          <input v-model="genreForm.name" placeholder="题材名称" />
        </div>
        <div class="form-group">
          <label>简介 (Markdown)</label>
          <textarea v-model="genreForm.profile_markdown" placeholder="题材简介，支持 Markdown" rows="6"></textarea>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>排序</label>
            <input v-model.number="genreForm.sort_order" type="number" placeholder="0" />
          </div>
          <div class="form-group">
            <label>状态</label>
            <select v-model="genreForm.is_active">
              <option :value="true">启用</option>
              <option :value="false">禁用</option>
            </select>
          </div>
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showGenreModal = false">取消</button>
          <button class="save-btn" @click="saveGenre" :disabled="genreSaving">{{ genreSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showPlatformModal" class="modal-overlay" @click.self="showPlatformModal = false">
      <div class="modal">
        <h3>{{ editingPlatform ? '编辑平台' : '新增平台' }}</h3>
        <div class="form-group">
          <label>名称</label>
          <input v-model="platformForm.name" placeholder="平台名称" />
        </div>
        <div class="form-group">
          <label>风格指南 (Markdown)</label>
          <textarea v-model="platformForm.style_guide" placeholder="平台风格指南，支持 Markdown" rows="6"></textarea>
        </div>
        <div class="form-row">
          <div class="form-group">
            <label>排序</label>
            <input v-model.number="platformForm.sort_order" type="number" placeholder="0" />
          </div>
          <div class="form-group">
            <label>状态</label>
            <select v-model="platformForm.is_active">
              <option :value="true">启用</option>
              <option :value="false">禁用</option>
            </select>
          </div>
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showPlatformModal = false">取消</button>
          <button class="save-btn" @click="savePlatform" :disabled="platformSaving">{{ platformSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showRechargeModal" class="modal-overlay" @click.self="showRechargeModal = false">
      <div class="modal">
        <h3>充值 - {{ rechargeTarget?.username }}</h3>
        <div class="form-group">
          <label>当前余额</label>
          <div class="balance-display">{{ (rechargeTarget?.balance || 0).toLocaleString() }}</div>
        </div>
        <div class="form-group">
          <label>充值金额（正数为增加，负数为减少）</label>
          <input v-model.number="rechargeAmount" type="number" placeholder="请输入金额" />
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showRechargeModal = false">取消</button>
          <button class="save-btn" @click="doRecharge" :disabled="recharging">{{ recharging ? '充值中...' : '确认' }}</button>
        </div>
      </div>
    </div>

    <div v-if="showContentModal" class="modal-overlay" @click.self="showContentModal = false">
      <div class="modal md-modal">
        <h3>{{ contentTarget?.name }}</h3>
        <div class="md-content" v-html="renderedMarkdown"></div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showContentModal = false">关闭</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted, watch } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import yaml from 'js-yaml'

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

const genres = ref([])
const platforms = ref([])

const showGenreModal = ref(false)
const editingGenre = ref(null)
const genreSaving = ref(false)
const genreForm = ref({ name: '', profile_markdown: '', sort_order: 0, is_active: true })

const showPlatformModal = ref(false)
const editingPlatform = ref(null)
const platformSaving = ref(false)
const platformForm = ref({ name: '', style_guide: '', sort_order: 0, is_active: true })

const showRechargeModal = ref(false)
const rechargeTarget = ref(null)
const rechargeAmount = ref(0)
const recharging = ref(false)

const showContentModal = ref(false)
const contentTarget = ref(null)

const renderedMarkdown = computed(() => {
  const md = contentTarget.value?.profile_markdown || ''
  if (!md) return '<p style="color:#94a3b8">暂无内容</p>'

  const frontMatterMatch = md.match(/^---\s*\n([\s\S]*?)\n---\s*\n?([\s\S]*)$/)
  let fmHTML = ''
  let bodyMD = md

  if (frontMatterMatch) {
    try {
      const parsed = yaml.load(frontMatterMatch[1])
      if (parsed && typeof parsed === 'object') {
        fmHTML = '<div class="fm-table-wrap"><table class="fm-table"><tbody>'
        for (const [key, val] of Object.entries(parsed)) {
          const displayVal = Array.isArray(val) ? val.join(', ') : String(val)
          fmHTML += `<tr><td class="fm-key">${key}</td><td class="fm-val">${displayVal}</td></tr>`
        }
        fmHTML += '</tbody></table></div>'
      }
      bodyMD = frontMatterMatch[2] || ''
    } catch {
      bodyMD = md
    }
  }

  const bodyHTML = bodyMD.trim() ? marked(bodyMD) : ''
  return fmHTML + bodyHTML
})

const totalPages = computed(() => Math.ceil(userTotal.value / pageSize) || 1)

const tabTitle = computed(() => {
  const titles = { dashboard: '仪表盘', users: '用户管理', genres: '题材管理', platforms: '平台管理', settings: '系统设置' }
  return titles[tab.value] || ''
})

const token = localStorage.getItem('token')
const headers = { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }

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

async function loadGenres() {
  const res = await fetch('/api/v1/admin/genres', { headers })
  if (res.ok) genres.value = await res.json()
}

async function loadPlatforms() {
  const res = await fetch('/api/v1/admin/platforms', { headers })
  if (res.ok) platforms.value = await res.json()
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
      initMsg.value = '初始化成功！题材和平台配置已就绪。'
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

function openGenreModal(genre) {
  if (genre) {
    editingGenre.value = genre
    genreForm.value = { name: genre.name, profile_markdown: genre.profile_markdown || '', sort_order: genre.sort_order, is_active: genre.is_active }
  } else {
    editingGenre.value = null
    genreForm.value = { name: '', profile_markdown: '', sort_order: 0, is_active: true }
  }
  showGenreModal.value = true
}

function viewGenreContent(g) {
  contentTarget.value = g
  showContentModal.value = true
}

async function saveGenre() {
  genreSaving.value = true
  try {
    const body = JSON.stringify(genreForm.value)
    let res
    if (editingGenre.value) {
      res = await fetch(`/api/v1/admin/genres/${editingGenre.value.id}`, { method: 'PUT', headers, body })
    } else {
      res = await fetch('/api/v1/admin/genres', { method: 'POST', headers, body })
    }
    if (res.ok) {
      showGenreModal.value = false
      await loadGenres()
    }
  } finally {
    genreSaving.value = false
  }
}

async function deleteGenre(id) {
  if (!confirm('确定删除该题材？')) return
  const res = await fetch(`/api/v1/admin/genres/${id}`, { method: 'DELETE', headers })
  if (res.ok) await loadGenres()
}

function openPlatformModal(platform) {
  if (platform) {
    editingPlatform.value = platform
    platformForm.value = { name: platform.name, style_guide: platform.style_guide || '', sort_order: platform.sort_order, is_active: platform.is_active }
  } else {
    editingPlatform.value = null
    platformForm.value = { name: '', style_guide: '', sort_order: 0, is_active: true }
  }
  showPlatformModal.value = true
}

async function savePlatform() {
  platformSaving.value = true
  try {
    const body = JSON.stringify(platformForm.value)
    let res
    if (editingPlatform.value) {
      res = await fetch(`/api/v1/admin/platforms/${editingPlatform.value.id}`, { method: 'PUT', headers, body })
    } else {
      res = await fetch('/api/v1/admin/platforms', { method: 'POST', headers, body })
    }
    if (res.ok) {
      showPlatformModal.value = false
      await loadPlatforms()
    }
  } finally {
    platformSaving.value = false
  }
}

async function deletePlatform(id) {
  if (!confirm('确定删除该平台？')) return
  const res = await fetch(`/api/v1/admin/platforms/${id}`, { method: 'DELETE', headers })
  if (res.ok) await loadPlatforms()
}

function openRechargeModal(user) {
  rechargeTarget.value = user
  rechargeAmount.value = 0
  showRechargeModal.value = true
}

async function doRecharge() {
  if (!rechargeAmount.value) return
  recharging.value = true
  try {
    const body = JSON.stringify({ amount: rechargeAmount.value })
    const res = await fetch(`/api/v1/admin/users/${rechargeTarget.value.id}/balance`, { method: 'POST', headers, body })
    if (res.ok) {
      showRechargeModal.value = false
      await loadUsers()
    }
  } finally {
    recharging.value = false
  }
}

async function toggleUserStatus(u) {
  const newStatus = u.status === 'active' ? 'disabled' : 'active'
  const action = newStatus === 'disabled' ? '禁用' : '启用'
  if (!confirm(`确定${action}用户「${u.username}」？`)) return
  const body = JSON.stringify({ status: newStatus })
  const res = await fetch(`/api/v1/admin/users/${u.id}/status`, { method: 'PUT', headers, body })
  if (res.ok) await loadUsers()
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
  background: #f1f5f9;
}
.sidebar {
  width: 240px;
  background: #fff;
  border-right: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  padding: 24px 0;
}
.sidebar-logo {
  padding: 0 24px 24px;
  border-bottom: 1px solid #e2e8f0;
}
.sidebar-logo h2 {
  font-size: 20px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.badge {
  font-size: 11px;
  color: #2563eb;
  border: 1px solid #2563eb;
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
  color: #64748b;
  text-decoration: none;
  font-size: 14px;
  transition: all 0.2s;
  border-left: 3px solid transparent;
  cursor: pointer;
}
.nav-item:hover, .nav-item.active {
  color: #1e293b;
  background: #f1f5f9;
  border-left-color: #2563eb;
}
.sidebar-footer {
  padding: 16px 24px;
  border-top: 1px solid #e2e8f0;
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
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
  color: #fff;
}
.user-name { font-size: 14px; color: #1e293b; }
.user-role { font-size: 12px; color: #94a3b8; }
.logout-btn {
  width: 100%;
  padding: 8px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  font-size: 13px;
}
.logout-btn:hover { color: #dc2626; border-color: #dc2626; }
.main-content { flex: 1; display: flex; flex-direction: column; background: #fff; }
.topbar {
  padding: 20px 32px;
  border-bottom: 1px solid #e2e8f0;
}
.topbar h1 { font-size: 22px; color: #1e293b; }
.content { padding: 32px; flex: 1; background: #f8fafc; }
.stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 20px;
  margin-bottom: 32px;
}
.stat-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 24px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.stat-value {
  font-size: 32px;
  font-weight: 700;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.stat-label { font-size: 14px; color: #64748b; margin-top: 8px; }

.data-section { }
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-count { color: #64748b; font-size: 14px; }
.add-btn {
  padding: 8px 20px;
  border: 1px solid #2563eb;
  border-radius: 6px;
  background: transparent;
  color: #2563eb;
  font-size: 13px;
  cursor: pointer;
  transition: all 0.2s;
}
.add-btn:hover { background: rgba(37,99,235,0.08); }

.table-wrap {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  overflow: hidden;
}
.data-table {
  width: 100%;
  border-collapse: collapse;
}
.data-table th {
  text-align: left;
  padding: 14px 20px;
  font-size: 13px;
  color: #64748b;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
  font-weight: 600;
}
.data-table td {
  padding: 14px 20px;
  font-size: 14px;
  color: #334155;
  border-bottom: 1px solid #f1f5f9;
}
.data-table tr:last-child td { border-bottom: none; }
.data-table tr:hover td { background: #f8fafc; }
.empty-row { text-align: center; color: #94a3b8; padding: 32px !important; }
.role-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.role-tag.admin { background: rgba(37,99,235,0.1); color: #2563eb; }
.role-tag.user { background: #f1f5f9; color: #64748b; }
.status-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.status-tag.active { background: rgba(22,163,74,0.1); color: #16a34a; }
.status-tag.inactive { background: #f1f5f9; color: #94a3b8; }
.action-btn {
  padding: 4px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: transparent;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  margin-right: 8px;
  transition: all 0.2s;
}
.action-btn:hover { color: #2563eb; border-color: #2563eb; }
.action-btn.danger:hover { color: #dc2626; border-color: #dc2626; }
.action-group {
  display: flex;
  gap: 6px;
}
.balance-display {
  color: #2563eb;
  font-size: 20px;
  font-weight: 600;
  padding: 8px 0;
}

.pagination {
  display: flex;
  justify-content: center;
  align-items: center;
  gap: 16px;
  margin-top: 20px;
}
.pagination button {
  padding: 8px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  font-size: 13px;
}
.pagination button:hover:not(:disabled) { color: #2563eb; border-color: #2563eb; }
.pagination button:disabled { opacity: 0.3; cursor: not-allowed; }
.pagination span { color: #64748b; font-size: 13px; }

.init-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 32px;
  max-width: 560px;
}
.init-header { margin-bottom: 20px; }
.init-header h3 { color: #1e293b; font-size: 18px; margin-bottom: 8px; }
.init-desc { color: #94a3b8; font-size: 13px; line-height: 1.6; }
.init-btn {
  padding: 10px 28px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 14px;
  cursor: pointer;
  transition: opacity 0.2s;
}
.init-btn:hover:not(:disabled) { opacity: 0.9; }
.init-btn:disabled { opacity: 0.5; cursor: not-allowed; }
.init-msg { margin-top: 16px; font-size: 13px; color: #16a34a; }
.init-msg.error { color: #dc2626; }

.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(0,0,0,0.4);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 100;
}
.modal {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 16px;
  padding: 32px;
  width: 520px;
  max-height: 80vh;
  overflow-y: auto;
  box-shadow: 0 8px 40px rgba(0,0,0,0.12);
}
.modal h3 {
  color: #1e293b;
  font-size: 18px;
  margin-bottom: 24px;
}
.form-group {
  margin-bottom: 16px;
}
.form-group label {
  display: block;
  color: #475569;
  font-size: 13px;
  margin-bottom: 6px;
}
.form-group input,
.form-group textarea,
.form-group select {
  width: 100%;
  padding: 10px 14px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1e293b;
  font-size: 14px;
  outline: none;
  transition: border-color 0.2s;
  box-sizing: border-box;
}
.form-group input:focus,
.form-group textarea:focus,
.form-group select:focus {
  border-color: #2563eb;
}
.form-group textarea { resize: vertical; font-family: inherit; }
.form-group select option { background: #fff; color: #1e293b; }
.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}
.modal-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 24px;
}
.cancel-btn {
  padding: 8px 20px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  font-size: 13px;
  cursor: pointer;
}
.cancel-btn:hover { color: #1e293b; }
.save-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
}
.save-btn:hover:not(:disabled) { opacity: 0.9; }
.save-btn:disabled { opacity: 0.5; cursor: not-allowed; }

.view-btn {
  padding: 4px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: transparent;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.view-btn:hover { color: #2563eb; border-color: #2563eb; }

.md-modal { width: 680px; }
.md-content {
  color: #334155;
  font-size: 14px;
  line-height: 1.8;
  max-height: 60vh;
  overflow-y: auto;
}
.md-content :deep(h1),
.md-content :deep(h2),
.md-content :deep(h3) { color: #1e293b; margin: 16px 0 8px; }
.md-content :deep(p) { margin: 8px 0; }
.md-content :deep(ul),
.md-content :deep(ol) { padding-left: 20px; margin: 8px 0; }
.md-content :deep(code) {
  background: #f1f5f9;
  padding: 2px 6px;
  border-radius: 4px;
  font-size: 13px;
}
.md-content :deep(pre) {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 16px;
  overflow-x: auto;
}
.md-content :deep(blockquote) {
  border-left: 3px solid #2563eb;
  padding-left: 16px;
  color: #64748b;
  margin: 12px 0;
}
.md-content :deep(.fm-table-wrap) {
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 10px;
  padding: 16px;
  margin-bottom: 20px;
}
.md-content :deep(.fm-table) {
  width: 100%;
  border-collapse: collapse;
}
.md-content :deep(.fm-table td) {
  padding: 8px 12px;
  font-size: 13px;
  border-bottom: 1px solid #e2e8f0;
}
.md-content :deep(.fm-table tr:last-child td) { border-bottom: none; }
.md-content :deep(.fm-key) {
  color: #2563eb;
  font-weight: 600;
  white-space: nowrap;
  width: 1%;
  padding-right: 20px !important;
}
.md-content :deep(.fm-val) {
  color: #334155;
  word-break: break-all;
}
</style>
