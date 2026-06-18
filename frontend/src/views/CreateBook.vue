<template>
  <div class="create-book-container">
    <header class="topbar">
      <div class="topbar-left">
        <router-link to="/write" class="back-link">← 返回</router-link>
        <h1>创建新小说</h1>
      </div>
      <div class="topbar-right">
        <span class="user-name">{{ user?.username }}</span>
        <button class="logout-btn" @click="logout">退出</button>
      </div>
    </header>

    <main class="main-content">
      <form class="book-form" @submit.prevent="submit">
        <div class="form-section">
          <h3>基本信息</h3>

          <div class="form-group">
            <label>书名 <span class="required">*</span></label>
            <input
              v-model="form.title"
              type="text"
              placeholder="给你的小说起个名字"
              maxlength="255"
              required
            />
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>题材 <span class="required">*</span></label>
              <select v-model="form.genre_id" required>
                <option value="" disabled>请选择题材</option>
                <option v-for="g in genres" :key="g.id" :value="g.id">{{ g.name }}</option>
              </select>
            </div>

            <div class="form-group">
              <label>目标平台 <span class="required">*</span></label>
              <select v-model="form.platform_id" required>
                <option value="" disabled>请选择平台</option>
                <option v-for="p in platforms" :key="p.id" :value="p.id">{{ p.name }}</option>
              </select>
            </div>
          </div>

          <div class="form-row">
            <div class="form-group">
              <label>每章目标字数 <span class="required">*</span></label>
              <input
                v-model.number="form.chapter_word_count"
                type="number"
                placeholder="3000"
                min="500"
                max="50000"
                required
              />
              <span class="hint">建议 2000-5000 字</span>
            </div>

            <div class="form-group">
              <label>目标总章数 <span class="required">*</span></label>
              <input
                v-model.number="form.target_chapters"
                type="number"
                placeholder="200"
                min="1"
                max="10000"
                required
              />
              <span class="hint">长篇建议 100-500 章</span>
            </div>
          </div>
        </div>

        <div class="form-section">
          <h3>书籍简介</h3>
          <div class="form-group">
            <textarea
              v-model="form.description"
              placeholder="简要描述你的故事背景、核心冲突和主角设定，AI 将基于此生成世界观和角色..."
              rows="6"
              maxlength="5000"
            ></textarea>
            <span class="hint char-count">{{ form.description.length }}/5000</span>
          </div>
        </div>

        <div class="form-section">
          <h3>AI 模型配置</h3>
          <div class="form-group">
            <label>写作模型</label>
            <select v-model="form.llm_model_id">
              <option :value="0">使用默认模型</option>
              <optgroup v-for="cfg in llmConfigs" :key="cfg.id" :label="cfg.label">
                <option v-for="m in cfg.models" :key="m.id" :value="m.id" :disabled="!m.is_enabled">
                  {{ m.model_name }}{{ m.is_default ? ' (默认)' : '' }}{{ m.is_enabled ? '' : ' (已禁用)' }}
                </option>
              </optgroup>
            </select>
            <span class="hint" v-if="llmConfigs.length === 0">
              暂无可用模型，请联系管理员配置
            </span>
          </div>
        </div>

        <div class="form-actions">
          <button type="button" class="btn-cancel" @click="$router.push('/write')">取消</button>
          <button type="submit" class="btn-submit" :disabled="submitting">
            {{ submitting ? '创建并初始化中...' : '创建小说' }}
          </button>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>
      </form>
    </main>
  </div>
</template>

<script setup>
import { ref, reactive, onMounted } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const user = ref(null)
const genres = ref([])
const platforms = ref([])
const llmConfigs = ref([])
const submitting = ref(false)
const error = ref('')

const form = reactive({
  title: '',
  genre_id: '',
  platform_id: '',
  chapter_word_count: 3000,
  target_chapters: 200,
  description: '',
  llm_model_id: 0,
})

const token = localStorage.getItem('token')
const headers = { 'Authorization': 'Bearer ' + token, 'Content-Type': 'application/json' }

onMounted(async () => {
  try {
    const [userRes, genreRes, platformRes, llmRes] = await Promise.all([
      fetch('/api/v1/me', { headers }),
      fetch('/api/v1/genres', { headers }),
      fetch('/api/v1/platforms', { headers }),
      fetch('/api/v1/llm-configs', { headers }),
    ])

    if (userRes.ok) user.value = await userRes.json()
    if (genreRes.ok) genres.value = await genreRes.json()
    if (platformRes.ok) platforms.value = await platformRes.json()
    if (llmRes.ok) llmConfigs.value = await llmRes.json()
  } catch (e) {
    error.value = '加载配置失败，请刷新重试'
  }
})

async function submit() {
  error.value = ''
  submitting.value = true

  const body = { ...form }
  if (body.llm_model_id === 0) delete body.llm_model_id

  try {
    const res = await fetch('/api/v1/books', {
      method: 'POST',
      headers,
      body: JSON.stringify(body),
    })

    if (res.ok) {
      router.push('/write')
    } else {
      const data = await res.json()
      error.value = data.error || '创建失败'
    }
  } catch (e) {
    error.value = '网络错误，请重试'
  } finally {
    submitting.value = false
  }
}

function logout() {
  localStorage.removeItem('token')
  localStorage.removeItem('role')
  router.push('/login')
}
</script>

<style scoped>
.create-book-container {
  min-height: 100vh;
  background: #f8fafc;
  display: flex;
  flex-direction: column;
}

.topbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  padding: 16px 32px;
  background: #fff;
  border-bottom: 1px solid #e2e8f0;
}

.topbar-left {
  display: flex;
  align-items: center;
  gap: 20px;
}

.back-link {
  color: #64748b;
  text-decoration: none;
  font-size: 14px;
  transition: color 0.2s;
}

.back-link:hover { color: #2563eb; }

.topbar-left h1 {
  font-size: 20px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}

.topbar-right {
  display: flex;
  align-items: center;
  gap: 16px;
}

.user-name { color: #64748b; font-size: 14px; }

.logout-btn {
  padding: 6px 16px;
  border: 1px solid #e2e8f0;
  border-radius: 6px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  font-size: 13px;
}

.logout-btn:hover { color: #dc2626; border-color: #dc2626; }

.main-content {
  flex: 1;
  display: flex;
  justify-content: center;
  padding: 40px 20px;
}

.book-form {
  width: 100%;
  max-width: 680px;
}

.form-section {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 28px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}

.form-section h3 {
  font-size: 16px;
  color: #1e293b;
  margin-bottom: 20px;
  padding-bottom: 12px;
  border-bottom: 1px solid #e2e8f0;
}

.form-group {
  margin-bottom: 18px;
}

.form-group:last-child { margin-bottom: 0; }

.form-group label {
  display: block;
  font-size: 13px;
  color: #475569;
  margin-bottom: 6px;
}

.required { color: #dc2626; }

.form-group input,
.form-group select,
.form-group textarea {
  width: 100%;
  padding: 10px 14px;
  background: #f8fafc;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  color: #1e293b;
  font-size: 14px;
  font-family: inherit;
  outline: none;
  transition: border-color 0.2s;
}

.form-group input:focus,
.form-group select:focus,
.form-group textarea:focus {
  border-color: #2563eb;
}

.form-group select {
  cursor: pointer;
  appearance: none;
  background-image: url("data:image/svg+xml,%3Csvg xmlns='http://www.w3.org/2000/svg' width='12' height='12' viewBox='0 0 12 12'%3E%3Cpath fill='%2364748b' d='M6 8L1 3h10z'/%3E%3C/svg%3E");
  background-repeat: no-repeat;
  background-position: right 12px center;
  padding-right: 36px;
}

.form-group select option {
  background: #fff;
  color: #1e293b;
}

.form-group textarea {
  resize: vertical;
  min-height: 120px;
  line-height: 1.6;
}

.form-row {
  display: grid;
  grid-template-columns: 1fr 1fr;
  gap: 16px;
}

.hint {
  display: block;
  font-size: 12px;
  color: #94a3b8;
  margin-top: 4px;
}

.char-count {
  text-align: right;
}

.form-actions {
  display: flex;
  justify-content: flex-end;
  gap: 12px;
  margin-top: 8px;
}

.btn-cancel {
  padding: 10px 24px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: transparent;
  color: #64748b;
  cursor: pointer;
  font-size: 14px;
  transition: all 0.2s;
}

.btn-cancel:hover { color: #1e293b; border-color: #94a3b8; }

.btn-submit {
  padding: 10px 32px;
  border: none;
  border-radius: 8px;
  background: linear-gradient(135deg, #2563eb, #1d4ed8);
  color: #fff;
  cursor: pointer;
  font-size: 14px;
  font-weight: 600;
  transition: opacity 0.2s;
}

.btn-submit:hover { opacity: 0.9; }
.btn-submit:disabled { opacity: 0.5; cursor: not-allowed; }

.error-msg {
  margin-top: 16px;
  padding: 12px 16px;
  background: rgba(220,38,38,0.08);
  border: 1px solid rgba(220,38,38,0.2);
  border-radius: 8px;
  color: #dc2626;
  font-size: 13px;
  text-align: center;
}

@media (max-width: 900px) {
  .topbar {
    flex-direction: column;
    align-items: flex-start;
    gap: 12px;
    padding: 16px 20px;
  }
  .topbar-left,
  .topbar-right {
    width: 100%;
    justify-content: space-between;
  }
  .main-content {
    padding: 24px 16px;
  }
  .form-section {
    padding: 22px 18px;
  }
  .form-row {
    grid-template-columns: 1fr;
    gap: 0;
  }
  .form-actions {
    flex-direction: column-reverse;
  }
  .btn-cancel,
  .btn-submit {
    width: 100%;
  }
}
</style>
