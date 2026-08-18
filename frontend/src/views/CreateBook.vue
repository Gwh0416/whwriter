<template>
  <div class="create-book-container">
    <header class="topbar">
      <div class="topbar-left">
        <router-link to="/write" class="back-link">← 返回</router-link>
        <h1>创建新小说</h1>
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
          <h3>番茄标签 <span class="required">*</span></h3>
          <p class="hint">选择官方主题、角色和情节标签。写作时会加载这些标签下的“我的雷达”画像与规则。</p>
          <div class="tag-type-tabs">
            <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: activeTagType === group.key }" @click="activeTagType = group.key">{{ group.label }}</button>
          </div>
          <div v-if="radarTags.length === 0" class="empty-tags">
            暂无可写作标签。请先到「我的雷达」扫描书籍，生成画像后再聚合相关标签。
          </div>
          <div class="radar-tags">
            <label v-for="tag in activeTags" :key="tag.tag_key" class="radar-tag" :class="{ active: form.radar_tags.includes(tag.tag_key) }">
              <input type="checkbox" :value="tag.tag_key" v-model="form.radar_tags" />
              <span>{{ tag.tag_name }}</span>
            </label>
          </div>
          <div v-if="selectedTags.length" class="selected-tags">
            <span>已选</span>
            <button v-for="tag in selectedTags" :key="tag.tag_key" type="button" @click="removeTag(tag.tag_key)">{{ tag.tag_name }} ×</button>
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
              暂无可用模型，请先到「系统设置」配置模型
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
import { ref, reactive, onMounted, computed } from 'vue'
import { useRouter } from 'vue-router'

const router = useRouter()
const platforms = ref([])
const llmConfigs = ref([])
const radarTags = ref([])
const submitting = ref(false)
const error = ref('')
const activeTagType = ref('plot')

const form = reactive({
  title: '',
  platform_id: '',
  chapter_word_count: 3000,
  target_chapters: 200,
  description: '',
  llm_model_id: 0,
  radar_tags: [],
})

const headers = { 'Content-Type': 'application/json' }

const tagGroups = computed(() => {
  const groups = [
    { key: 'plot', label: '情节', items: [] },
    { key: 'role', label: '角色', items: [] },
    { key: 'theme', label: '主题', items: [] },
  ]
  const byKey = Object.fromEntries(groups.map(group => [group.key, group]))
  for (const tag of radarTags.value) {
    const key = tag.tag_type || tag.category || 'plot'
    ;(byKey[key] || byKey.plot).items.push(tag)
  }
  return groups
})
const activeTags = computed(() => tagGroups.value.find(group => group.key === activeTagType.value)?.items || [])
const selectedTags = computed(() => {
  const selected = new Set(form.radar_tags)
  return radarTags.value.filter(tag => selected.has(tag.tag_key))
})

function removeTag(key) {
  form.radar_tags = form.radar_tags.filter(item => item !== key)
}

onMounted(async () => {
  try {
    const [platformRes, llmRes, radarRes] = await Promise.all([
      fetch('/api/v1/platforms', { headers }),
      fetch('/api/v1/llm-configs', { headers }),
      fetch('/api/v1/radar/taxonomies?ready_only=1', { headers }),
    ])

    if (platformRes.ok) platforms.value = await platformRes.json()
    if (llmRes.ok) llmConfigs.value = await llmRes.json()
    if (radarRes.ok) {
      const radar = await radarRes.json()
      radarTags.value = radar.tags || []
    }
  } catch (e) {
    error.value = '加载配置失败，请刷新重试'
  }
})

async function submit() {
  error.value = ''
  if (!form.radar_tags.length) {
    error.value = radarTags.value.length ? '请至少选择一个番茄标签' : '暂无可写作标签，请先到「我的雷达」生成画像并聚合相关标签'
    return
  }
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
      error.value = data.message || data.error?.message || data.error || '创建失败'
    }
  } catch (e) {
    error.value = '网络错误，请重试'
  } finally {
    submitting.value = false
  }
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

.tag-type-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin: 14px 0 12px;
}

.tag-type-tabs button {
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #475569;
  border-radius: 999px;
  padding: 8px 16px;
  cursor: pointer;
  font-weight: 600;
}

.tag-type-tabs button.active {
  color: #ea580c;
  background: #fff1eb;
  border-color: #fed7aa;
}

.radar-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  max-height: 180px;
  overflow-y: auto;
  padding-right: 4px;
}

.empty-tags {
  padding: 14px 16px;
  border: 1px dashed #cbd5e1;
  border-radius: 10px;
  color: #64748b;
  background: #f8fafc;
  font-size: 13px;
  line-height: 1.7;
}

.radar-tag {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  padding: 8px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  background: #f8fafc;
  color: #475569;
  font-size: 13px;
  cursor: pointer;
}

.radar-tag.active {
  color: #1d4ed8;
  background: #eff6ff;
  border-color: #bfdbfe;
}

.radar-tag input {
  width: auto;
}

.selected-tags {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
  color: #64748b;
  font-size: 12px;
}

.selected-tags button {
  border: none;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  padding: 5px 9px;
  cursor: pointer;
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
  .topbar-left {
    width: 100%;
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
