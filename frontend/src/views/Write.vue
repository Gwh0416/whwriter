<template>
  <div class="write-container">
    <aside class="sidebar">
      <div class="sidebar-logo">
        <h2>文豪写作</h2>
      </div>
      <nav class="sidebar-nav">
        <a href="#" class="nav-item" :class="{ active: tab === 'books' }" @click.prevent="tab = 'books'">书籍管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'genres' }" @click.prevent="tab = 'genres'; loadGenres()">题材管理</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'settings' }" @click.prevent="tab = 'settings'">个人设置</a>
        <a href="#" class="nav-item" :class="{ active: tab === 'conversations' }" @click.prevent="tab = 'conversations'">对话管理</a>
      </nav>
      <div class="sidebar-footer">
        <div class="user-info">
          <div class="avatar">{{ user?.username?.[0] }}</div>
          <div>
            <div class="user-name">{{ user?.username }}</div>
          </div>
        </div>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </aside>
    <main class="main-content">
      <header class="topbar">
        <h1>{{ tabTitle }}</h1>
        <div class="topbar-actions">
          <router-link v-if="tab === 'books'" to="/create-book" class="create-btn">+ 创建新小说</router-link>
        </div>
      </header>
      <div class="content">

        <div v-if="tab === 'books'">
          <div v-if="books.length === 0" class="empty-state">
            <div class="empty-icon">✍️</div>
            <h2>开始你的创作之旅</h2>
            <p>AI 驱动的智能写作助手，帮你完成从构思到成稿的全流程</p>
            <div class="empty-actions">
              <router-link to="/create-book" class="welcome-btn primary">创建新小说</router-link>
            </div>
          </div>
          <div v-else class="data-section">
            <div class="section-header">
              <div class="section-count">共 {{ books.length }} 本小说</div>
            </div>
            <div class="table-wrap">
              <table class="data-table">
                <thead>
                  <tr>
                    <th>书名</th>
                    <th>题材</th>
                    <th>平台</th>
                    <th>字数/章</th>
                    <th>目标章数</th>
                    <th>状态</th>
                    <th>创建时间</th>
                    <th>操作</th>
                  </tr>
                </thead>
                <tbody>
                  <tr v-for="b in books" :key="b.id">
                    <td class="book-title">{{ b.title }}</td>
                    <td>{{ b.genre?.name || '-' }}</td>
                    <td>{{ b.platform?.name || '-' }}</td>
                    <td>{{ b.chapter_word_count }}</td>
                    <td>{{ b.target_chapters }}</td>
                    <td>
                      <span class="status-tag" :class="b.status">{{ statusLabel(b.status) }}</span>
                    </td>
                    <td>{{ formatDate(b.created_at) }}</td>
                    <td>
                      <button class="action-btn">进入</button>
                    </td>
                  </tr>
                </tbody>
              </table>
            </div>
          </div>
        </div>

        <div v-if="tab === 'genres'" class="data-section">
          <div class="section-header">
            <div class="section-count">共 {{ genres.length }} 个题材</div>
            <button class="add-btn" @click="openGenreModal()">+ 新建题材</button>
          </div>
          <div class="table-wrap">
            <table class="data-table">
              <thead>
                <tr>
                  <th>名称</th>
                  <th>简介</th>
                  <th>类型</th>
                  <th>操作</th>
                </tr>
              </thead>
              <tbody>
                <tr v-for="g in genres" :key="g.id">
                  <td>{{ g.name }}</td>
                  <td>
                    <button class="view-btn" @click="viewMarkdown(g)">查看</button>
                  </td>
                  <td>
                    <span class="type-tag" :class="g.user_id === 0 ? 'public' : 'private'">
                      {{ g.user_id === 0 ? '公共' : '私有' }}
                    </span>
                  </td>
                  <td>
                    <div class="action-group" v-if="g.user_id !== 0">
                      <button class="action-btn" @click="openGenreModal(g)">编辑</button>
                      <button class="action-btn danger" @click="deleteMyGenre(g.id)">删除</button>
                    </div>
                    <span v-else class="no-action">—</span>
                  </td>
                </tr>
                <tr v-if="genres.length === 0">
                  <td colspan="4" class="empty-row">暂无可用的题材</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>

        <div v-if="tab === 'settings'" class="settings-section">
          <div class="settings-card">
            <div class="setting-item" @click="$router.push('/change-password')">
              <div class="setting-info">
                <div class="setting-label">修改密码</div>
                <div class="setting-desc">更新你的账户密码</div>
              </div>
              <div class="setting-arrow">→</div>
            </div>
          </div>
        </div>

        <div v-if="tab === 'conversations'" class="empty-state">
          <div class="empty-icon">💬</div>
          <h2>对话管理</h2>
          <p>即将上线，敬请期待</p>
        </div>

      </div>
    </main>

    <div v-if="showMarkdownModal" class="modal-overlay" @click.self="showMarkdownModal = false">
      <div class="modal md-modal">
        <h3>{{ markdownTarget?.name }}</h3>
        <div class="md-content" v-html="renderedMarkdown"></div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showMarkdownModal = false">关闭</button>
        </div>
      </div>
    </div>

    <div v-if="showGenreModal" class="modal-overlay" @click.self="showGenreModal = false">
      <div class="modal">
        <h3>{{ editingGenre ? '编辑题材' : '新建题材' }}</h3>
        <div class="form-group">
          <label>名称</label>
          <input v-model="genreForm.name" placeholder="题材名称" />
        </div>
        <div class="form-group">
          <label>简介 (Markdown)</label>
          <textarea v-model="genreForm.profile_markdown" placeholder="题材简介，支持 Markdown" rows="6"></textarea>
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showGenreModal = false">取消</button>
          <button class="save-btn" @click="saveGenre" :disabled="genreSaving">{{ genreSaving ? '保存中...' : '保存' }}</button>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { ref, computed, onMounted } from 'vue'
import { useRouter } from 'vue-router'
import { marked } from 'marked'
import yaml from 'js-yaml'

const router = useRouter()
const user = ref(null)
const tab = ref('books')
const books = ref([])
const genres = ref([])

const showMarkdownModal = ref(false)
const markdownTarget = ref(null)
const renderedMarkdown = computed(() => {
  const md = markdownTarget.value?.profile_markdown || ''
  if (!md) return '<p style="color:#94a3b8">暂无简介</p>'

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

const showGenreModal = ref(false)
const editingGenre = ref(null)
const genreSaving = ref(false)
const genreForm = ref({ name: '', profile_markdown: '' })

const token = localStorage.getItem('token')
const headers = { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }

const tabTitle = computed(() => {
  const titles = { books: '书籍管理', genres: '题材管理', settings: '个人设置', conversations: '对话管理' }
  return titles[tab.value] || ''
})

const statusLabels = {
  outlining: '大纲阶段',
  active: '创作中',
  paused: '已暂停',
  completed: '已完成',
}

function statusLabel(s) {
  return statusLabels[s] || s
}

function formatDate(d) {
  if (!d) return '-'
  return new Date(d).toLocaleString('zh-CN', { year: 'numeric', month: '2-digit', day: '2-digit' })
}

onMounted(async () => {
  const [userRes, booksRes] = await Promise.all([
    fetch('/api/v1/me', { headers }),
    fetch('/api/v1/books', { headers }),
  ])
  if (userRes.ok) user.value = await userRes.json()
  if (booksRes.ok) books.value = await booksRes.json()
})

async function loadGenres() {
  const res = await fetch('/api/v1/my-genres', { headers })
  if (res.ok) genres.value = await res.json()
}

function viewMarkdown(g) {
  markdownTarget.value = g
  showMarkdownModal.value = true
}

function openGenreModal(genre) {
  if (genre) {
    editingGenre.value = genre
    genreForm.value = { name: genre.name, profile_markdown: genre.profile_markdown || '' }
  } else {
    editingGenre.value = null
    genreForm.value = { name: '', profile_markdown: '' }
  }
  showGenreModal.value = true
}

async function saveGenre() {
  genreSaving.value = true
  try {
    const body = JSON.stringify(genreForm.value)
    let res
    if (editingGenre.value) {
      res = await fetch(`/api/v1/my-genres/${editingGenre.value.id}`, { method: 'PUT', headers, body })
    } else {
      res = await fetch('/api/v1/my-genres', { method: 'POST', headers, body })
    }
    if (res.ok) {
      showGenreModal.value = false
      await loadGenres()
    }
  } finally {
    genreSaving.value = false
  }
}

async function deleteMyGenre(id) {
  if (!confirm('确定删除该题材？')) return
  const res = await fetch(`/api/v1/my-genres/${id}`, { method: 'DELETE', headers })
  if (res.ok) await loadGenres()
}

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<style scoped>
.write-container {
  display: flex;
  min-height: 100vh;
  background: #f1f5f9;
}
.sidebar {
  width: 220px;
  background: #fff;
  border-right: 1px solid #e2e8f0;
  display: flex;
  flex-direction: column;
  padding: 24px 0;
}
.sidebar-logo {
  padding: 0 20px 20px;
  border-bottom: 1px solid #e2e8f0;
}
.sidebar-logo h2 {
  font-size: 20px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.sidebar-nav {
  flex: 1;
  padding: 12px 0;
}
.nav-item {
  display: block;
  padding: 12px 20px;
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
  padding: 16px 20px;
  border-top: 1px solid #e2e8f0;
}
.user-info {
  display: flex;
  align-items: center;
  gap: 10px;
  margin-bottom: 12px;
}
.avatar {
  width: 32px;
  height: 32px;
  border-radius: 50%;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 14px;
  color: #fff;
}
.user-name { font-size: 14px; color: #1e293b; }
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
  padding: 18px 32px;
  border-bottom: 1px solid #e2e8f0;
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.topbar h1 { font-size: 20px; color: #1e293b; }
.create-btn {
  padding: 8px 20px;
  border: none;
  border-radius: 6px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  font-size: 13px;
  cursor: pointer;
  text-decoration: none;
  transition: opacity 0.2s;
}
.create-btn:hover { opacity: 0.9; }
.content { padding: 32px; flex: 1; background: #f8fafc; }

.empty-state {
  display: flex;
  flex-direction: column;
  align-items: center;
  justify-content: center;
  padding: 80px 40px;
  text-align: center;
}
.empty-icon { font-size: 48px; margin-bottom: 20px; }
.empty-state h2 { font-size: 26px; color: #1e293b; margin-bottom: 12px; }
.empty-state p { color: #64748b; font-size: 15px; max-width: 420px; line-height: 1.6; }
.empty-actions { margin-top: 28px; }
.welcome-btn {
  padding: 12px 32px;
  border-radius: 8px;
  font-size: 15px;
  cursor: pointer;
  transition: all 0.3s;
  border: 1px solid #e2e8f0;
  background: transparent;
  color: #64748b;
  text-decoration: none;
  display: inline-block;
}
.welcome-btn:hover { border-color: #2563eb; color: #2563eb; }
.welcome-btn.primary {
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  border: none;
  color: #fff;
  font-weight: 600;
}
.welcome-btn.primary:hover { opacity: 0.9; }

.data-section { }
.section-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 16px;
}
.section-count { color: #64748b; font-size: 14px; }
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
.book-title { font-weight: 600; color: #1e293b; }
.desc-cell { max-width: 300px; overflow: hidden; text-overflow: ellipsis; white-space: nowrap; }
.empty-row { text-align: center; color: #94a3b8; padding: 32px !important; }
.status-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.status-tag.outlining { background: rgba(37,99,235,0.1); color: #2563eb; }
.status-tag.active { background: rgba(22,163,74,0.1); color: #16a34a; }
.status-tag.paused { background: #f1f5f9; color: #94a3b8; }
.status-tag.completed { background: rgba(139,92,246,0.1); color: #8b5cf6; }

.settings-section {
  max-width: 480px;
}
.settings-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  overflow: hidden;
}
.setting-item {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 20px 24px;
  cursor: pointer;
  transition: background 0.2s;
}
.setting-item:hover { background: #f8fafc; }
.setting-item + .setting-item { border-top: 1px solid #f1f5f9; }
.setting-label { font-size: 15px; color: #1e293b; margin-bottom: 4px; }
.setting-desc { font-size: 13px; color: #94a3b8; }
.setting-arrow { color: #94a3b8; font-size: 16px; }

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

.type-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
}
.type-tag.public { background: rgba(37,99,235,0.08); color: #2563eb; }
.type-tag.private { background: #f1f5f9; color: #64748b; }

.action-group {
  display: flex;
  gap: 6px;
}
.action-btn {
  padding: 4px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 4px;
  background: transparent;
  color: #64748b;
  font-size: 12px;
  cursor: pointer;
  transition: all 0.2s;
}
.action-btn:hover { color: #2563eb; border-color: #2563eb; }
.action-btn.danger:hover { color: #dc2626; border-color: #dc2626; }
.no-action { color: #94a3b8; font-size: 13px; }

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
.md-modal {
  width: 640px;
}
.modal h3 {
  color: #1e293b;
  font-size: 18px;
  margin-bottom: 24px;
}
.md-content {
  color: #334155;
  font-size: 14px;
  line-height: 1.8;
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
.form-group textarea {
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
.form-group textarea:focus {
  border-color: #2563eb;
}
.form-group textarea { resize: vertical; font-family: inherit; }
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
</style>
