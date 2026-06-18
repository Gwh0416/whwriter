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
        <a href="#" class="nav-item" :class="{ active: tab === 'models' }" @click.prevent="tab = 'models'; loadModels()">模型管理</a>
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

        <div v-if="tab === 'models'" class="data-section">
          <div class="token-summary">
            <div class="token-card total">
              <div class="token-value">{{ (totalTokenUsage || 0).toLocaleString() }}</div>
              <div class="token-label">总 Token 消耗</div>
            </div>
            <div class="token-card" v-for="d in tokenDetails" :key="d.id">
              <div class="token-value">{{ (d.token_usage || 0).toLocaleString() }}</div>
              <div class="token-label">{{ d.provider }} / {{ d.model_name }}</div>
            </div>
          </div>
          <div class="section-header">
            <div class="section-count">共 {{ modelConfigs.length }} 个厂商</div>
            <button class="add-btn" @click="openProviderModal()">+ 新增厂商</button>
          </div>

          <div v-for="cfg in modelConfigs" :key="cfg.id" class="provider-card">
            <div class="provider-header">
              <div class="provider-info">
                <span class="provider-name">{{ cfg.label }}</span>
                <span class="provider-url">{{ cfg.base_url }}</span>
              </div>
              <div class="provider-actions">
                <button class="action-btn" @click="openProviderModal(cfg)">编辑</button>
                <button class="action-btn" @click="openTestConnection(cfg)">测试连接</button>
                <button class="action-btn danger" @click="deleteProvider(cfg.id)">删除</button>
              </div>
            </div>

            <div v-if="testingId === cfg.id" class="test-result" :class="{ success: testSuccess, error: !testSuccess }">
              <div v-if="testLoading">正在连接 {{ cfg.base_url }}/models ...</div>
              <div v-else-if="testSuccess">
                <div class="test-ok">✓ 连接成功，发现 {{ testModels.length }} 个模型</div>
                <div class="test-models">
                  <label v-for="m in testModels" :key="m" class="model-checkbox">
                    <input type="checkbox" :value="m" v-model="selectedTestModels" />
                    <span>{{ m }}</span>
                  </label>
                </div>
                <div class="test-actions">
                  <button class="save-btn sm" @click="importModels(cfg.id)" :disabled="selectedTestModels.length === 0">导入选中模型</button>
                  <button class="cancel-btn sm" @click="testingId = null">取消</button>
                </div>
              </div>
              <div v-else>{{ testError }}</div>
            </div>

            <div class="provider-models">
              <div v-if="cfg.models && cfg.models.length > 0" class="models-grid">
                <div v-for="m in cfg.models" :key="m.id" class="model-item" :class="{ disabled: !m.is_enabled }">
                  <div class="model-top">
                    <span class="model-name">{{ m.model_name }}</span>
                    <span v-if="m.is_default" class="default-badge">默认</span>
                  </div>
                  <div class="model-bottom">
                    <span class="model-usage">{{ (m.token_usage || 0).toLocaleString() }} tokens</span>
                    <div class="model-toggle">
                      <label class="toggle-label">
                        <input type="checkbox" :checked="m.is_enabled" @change="toggleModel(m)" />
                        <span>{{ m.is_enabled ? '启用' : '禁用' }}</span>
                      </label>
                      <button v-if="!m.is_default" class="action-btn sm" @click="setDefaultModel(m.id)">设为默认</button>
                    </div>
                  </div>
                </div>
              </div>
              <div v-else class="no-models">暂无模型，请点击「测试连接」获取模型列表</div>
            </div>
          </div>

          <div v-if="modelConfigs.length === 0" class="empty-state">
            <p>暂无模型厂商配置</p>
          </div>
        </div>

        <div v-if="tab === 'settings'" class="data-section">
          <div class="init-card">
            <div class="init-header">
              <h3>基础数据初始化</h3>
              <p class="init-desc">这里只初始化内置题材和平台配置，不会创建任何模型厂商或模型。模型请在「模型管理」中由管理员手动配置。</p>
            </div>
            <button class="init-btn" :disabled="initializing" @click="initialize">
              {{ initializing ? '初始化中...' : '初始化题材和平台' }}
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

    <div v-if="showProviderModal" class="modal-overlay" @click.self="showProviderModal = false">
      <div class="modal">
        <h3>{{ editingProvider ? '编辑厂商' : '新增厂商' }}</h3>
        <div class="form-group">
          <label>厂商名称</label>
          <input v-model="providerForm.provider" placeholder="如：OpenAI、DeepSeek" />
        </div>
        <div class="form-group">
          <label>显示标签</label>
          <input v-model="providerForm.label" placeholder="如：OpenAI" />
        </div>
        <div class="form-group">
          <label>Base URL</label>
          <input v-model="providerForm.base_url" placeholder="https://api.openai.com/v1" />
        </div>
        <div class="form-group">
          <label>API Key</label>
          <input v-model="providerForm.api_key" type="password" :placeholder="editingProvider ? '留空则不修改' : '请输入 API Key'" />
        </div>
        <div class="modal-actions">
          <button class="cancel-btn" @click="showProviderModal = false">取消</button>
          <button class="save-btn" @click="saveProvider" :disabled="providerSaving">{{ providerSaving ? '保存中...' : '保存' }}</button>
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

const modelConfigs = ref([])
const totalTokenUsage = ref(0)
const tokenDetails = ref([])

const showProviderModal = ref(false)
const editingProvider = ref(null)
const providerSaving = ref(false)
const providerForm = ref({ provider: '', label: '', base_url: '', api_key: '' })

const testingId = ref(null)
const testLoading = ref(false)
const testSuccess = ref(false)
const testError = ref('')
const testModels = ref([])
const selectedTestModels = ref([])

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
  const titles = { dashboard: '仪表盘', users: '用户管理', genres: '题材管理', platforms: '平台管理', models: '模型管理', settings: '系统设置' }
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
      initMsg.value = '基础数据初始化成功，题材和平台配置已就绪。模型需要在模型管理中手动添加。'
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

async function loadModels() {
  const [configsRes, usageRes] = await Promise.all([
    fetch('/api/v1/admin/llm-configs', { headers }),
    fetch('/api/v1/admin/llm-configs/token-usage', { headers }),
  ])
  if (configsRes.ok) modelConfigs.value = await configsRes.json()
  if (usageRes.ok) {
    const data = await usageRes.json()
    totalTokenUsage.value = data.total_usage
    tokenDetails.value = data.details
  }
}

function openProviderModal(cfg) {
  if (cfg) {
    editingProvider.value = cfg
    providerForm.value = { provider: cfg.provider, label: cfg.label, base_url: cfg.base_url, api_key: '' }
  } else {
    editingProvider.value = null
    providerForm.value = { provider: '', label: '', base_url: '', api_key: '' }
  }
  showProviderModal.value = true
}

async function saveProvider() {
  providerSaving.value = true
  try {
    const body = JSON.stringify(providerForm.value)
    let res
    if (editingProvider.value) {
      res = await fetch(`/api/v1/admin/llm-configs/${editingProvider.value.id}`, { method: 'PUT', headers, body })
    } else {
      res = await fetch('/api/v1/admin/llm-configs', { method: 'POST', headers, body })
    }
    if (res.ok) {
      showProviderModal.value = false
      await loadModels()
    }
  } finally {
    providerSaving.value = false
  }
}

async function deleteProvider(id) {
  if (!confirm('确定删除该厂商及其所有模型？')) return
  const res = await fetch(`/api/v1/admin/llm-configs/${id}`, { method: 'DELETE', headers })
  if (res.ok) await loadModels()
}

async function openTestConnection(cfg) {
  testingId.value = cfg.id
  testLoading.value = true
  testSuccess.value = false
  testError.value = ''
  testModels.value = []
  selectedTestModels.value = []

  try {
    const body = JSON.stringify({ config_id: cfg.id, base_url: cfg.base_url, api_key: '' })
    const res = await fetch('/api/v1/admin/llm-configs/test-connection', { method: 'POST', headers, body })
    const data = await res.json()
    if (data.success) {
      testSuccess.value = true
      testModels.value = data.models || []
    } else {
      testError.value = data.error || '连接失败'
    }
  } catch {
    testError.value = '网络错误'
  } finally {
    testLoading.value = false
  }
}

async function importModels(configID) {
  const models = selectedTestModels.value.map(name => ({
    model_name: name,
    is_enabled: true,
    is_default: false,
  }))
  const body = JSON.stringify({ models })
  const res = await fetch(`/api/v1/admin/llm-configs/${configID}/models`, { method: 'POST', headers, body })
  if (res.ok) {
    testingId.value = null
    await loadModels()
  }
}

async function toggleModel(m) {
  const body = JSON.stringify({
    models: modelConfigs.value
      .find(c => c.id === m.llm_config_id)
      .models.map(x => ({
        model_name: x.model_name,
        is_enabled: x.id === m.id ? !m.is_enabled : x.is_enabled,
        is_default: x.is_default,
      })),
  })
  const res = await fetch(`/api/v1/admin/llm-configs/${m.llm_config_id}/models`, { method: 'POST', headers, body })
  if (res.ok) await loadModels()
}

async function setDefaultModel(id) {
  const res = await fetch(`/api/v1/admin/llm-models/${id}/default`, { method: 'POST', headers })
  if (res.ok) await loadModels()
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
  overflow-x: auto;
  overflow-y: hidden;
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

.token-summary {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(180px, 1fr));
  gap: 16px;
  margin-bottom: 24px;
}
.token-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 20px;
  text-align: center;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.token-card.total {
  border-color: #2563eb;
  background: linear-gradient(135deg, rgba(37,99,235,0.04), rgba(37,99,235,0.01));
}
.token-value {
  font-size: 24px;
  font-weight: 700;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.token-label {
  font-size: 13px;
  color: #64748b;
  margin-top: 6px;
}

.provider-tag {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
  font-weight: 500;
}

.default-badge {
  display: inline-block;
  padding: 2px 10px;
  border-radius: 4px;
  font-size: 12px;
  background: rgba(37,99,235,0.1);
  color: #2563eb;
  font-weight: 500;
}

.action-btn.sm {
  padding: 2px 8px;
  font-size: 11px;
}

.save-btn.sm {
  padding: 6px 14px;
  font-size: 12px;
}

.cancel-btn.sm {
  padding: 6px 14px;
  font-size: 12px;
}

.provider-card {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  margin-bottom: 16px;
  overflow: hidden;
}
.provider-header {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 20px;
  background: #f8fafc;
  border-bottom: 1px solid #e2e8f0;
}
.provider-info {
  display: flex;
  flex-direction: column;
  gap: 4px;
}
.provider-name {
  font-size: 15px;
  font-weight: 600;
  color: #1e293b;
}
.provider-url {
  font-size: 12px;
  color: #94a3b8;
  font-family: monospace;
}
.provider-actions {
  display: flex;
  gap: 8px;
}

.test-result {
  padding: 16px 20px;
  border-bottom: 1px solid #e2e8f0;
  font-size: 13px;
}
.test-result.success { background: #f0fdf4; }
.test-result.error { background: #fef2f2; color: #dc2626; }
.test-ok { color: #16a34a; font-weight: 500; margin-bottom: 12px; }
.test-models {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-bottom: 12px;
}
.model-checkbox {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 4px 12px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  font-size: 13px;
  color: #334155;
  cursor: pointer;
}
.model-checkbox:hover { border-color: #2563eb; }
.test-actions {
  display: flex;
  gap: 8px;
}

.provider-models {
  padding: 16px 20px;
}
.models-grid {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(280px, 1fr));
  gap: 12px;
}
.model-item {
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  padding: 12px 16px;
  background: #fff;
}
.model-item.disabled {
  opacity: 0.5;
  background: #f8fafc;
}
.model-top {
  display: flex;
  justify-content: space-between;
  align-items: center;
  margin-bottom: 8px;
}
.model-name {
  font-size: 14px;
  font-weight: 500;
  color: #1e293b;
  font-family: monospace;
}
.model-bottom {
  display: flex;
  justify-content: space-between;
  align-items: center;
}
.model-usage {
  font-size: 12px;
  color: #94a3b8;
}
.model-toggle {
  display: flex;
  align-items: center;
  gap: 8px;
}
.toggle-label {
  display: flex;
  align-items: center;
  gap: 4px;
  font-size: 12px;
  color: #64748b;
  cursor: pointer;
}
.no-models {
  text-align: center;
  color: #94a3b8;
  font-size: 13px;
  padding: 20px;
}
.empty-state {
  text-align: center;
  padding: 60px 20px;
  color: #94a3b8;
  font-size: 14px;
}

@media (max-width: 1100px) {
  .admin-container {
    flex-direction: column;
  }
  .sidebar {
    width: 100%;
    border-right: none;
    border-bottom: 1px solid #e2e8f0;
    padding: 16px 0;
  }
  .sidebar-nav {
    display: flex;
    flex-wrap: wrap;
    gap: 8px;
    padding: 16px 20px 0;
  }
  .nav-item {
    border-left: none;
    border: 1px solid #e2e8f0;
    border-radius: 999px;
    padding: 8px 14px;
  }
  .nav-item:hover,
  .nav-item.active {
    border-left-color: transparent;
    border-color: #2563eb;
    background: rgba(37,99,235,0.08);
  }
  .sidebar-footer {
    padding: 12px 20px 0;
  }
  .content {
    padding: 24px 16px;
  }
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
    gap: 16px;
  }
  .section-header,
  .provider-header {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
  }
  .provider-actions,
  .action-group,
  .test-actions {
    flex-wrap: wrap;
  }
  .form-row {
    grid-template-columns: 1fr;
  }
}

@media (max-width: 640px) {
  .topbar {
    padding: 16px 20px;
  }
  .content {
    padding: 16px 12px;
  }
  .stats {
    grid-template-columns: 1fr;
  }
  .data-table th,
  .data-table td {
    padding: 12px 14px;
    white-space: nowrap;
  }
  .modal,
  .md-modal {
    width: min(92vw, 680px);
    padding: 20px 16px;
  }
  .token-summary,
  .models-grid {
    grid-template-columns: 1fr;
  }
}
</style>
