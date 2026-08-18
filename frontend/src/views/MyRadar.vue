<template>
  <div class="radar-panel">
    <div class="radar-toolbar">
      <div>
        <h2>番茄写法学习库</h2>
        <p>按番茄官方标签扫描或手动添加高价值书籍，沉淀单书画像、标签画像和可执行规则。</p>
      </div>
      <div class="radar-toolbar-actions">
        <span v-if="hasActiveJobs" class="polling-badge">自动刷新中</span>
        <button class="refresh-btn" @click="loadOverview" :disabled="loading">{{ loading ? '刷新中...' : '刷新进度' }}</button>
      </div>
    </div>

    <div class="content">
      <section v-if="activeSubTab === 'scan' || activeSubTab === 'intros'" class="panel radar-model-panel">
        <div class="form-group">
          <label>雷达模型</label>
          <select v-model.number="radarModelID">
            <option :value="0">使用默认模型</option>
            <optgroup v-for="cfg in llmConfigs" :key="cfg.id" :label="cfg.label || cfg.provider">
              <option v-for="m in cfg.models || []" :key="m.id" :value="m.id" :disabled="!m.is_enabled">
                {{ m.model_name }}{{ m.is_default ? ' (默认)' : '' }}{{ m.is_enabled ? '' : ' (已禁用)' }}
              </option>
            </optgroup>
          </select>
          <p class="hint">用于单书画像分析、标签规则生成和简介生成；不选则使用系统默认模型。</p>
        </div>
      </section>

      <div v-if="activeSubTab === 'scan'" class="grid">
        <section class="panel">
          <h3>自动扫描标签</h3>
          <div class="form-group">
            <label>番茄标签</label>
            <div class="tag-type-tabs">
              <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: scanTagType === group.key }" @click="scanTagType = group.key">{{ group.label }}</button>
            </div>
            <div class="tag-chip-grid">
              <button v-for="tag in tagsByType(scanTagType)" :key="tag.tag_key" type="button" class="tag-chip" :class="{ active: scanForm.category === tag.tag_key }" @click="scanForm.category = tag.tag_key">{{ tag.tag_name }}</button>
            </div>
          </div>
          <div class="form-group">
            <label>目标书籍数</label>
            <input v-model.number="scanForm.target_count" type="number" min="1" max="20" />
          </div>
          <div class="button-row">
            <button class="primary-btn" @click="startScan" :disabled="acting || !scanForm.category">开始扫描</button>
            <button class="action-btn" @click="openFanqieBrowserPage" :disabled="acting">打开番茄登录/验证页</button>
          </div>
          <p class="hint">自动扫描按番茄官方标签从书库发现书籍并保存样本；画像和规则在下方手动生成。</p>
        </section>

          <section class="panel">
            <h3>人工添加书籍</h3>
            <div class="form-row">
              <div class="form-group">
                <label>番茄 book_id 或 URL</label>
                <input v-model="sourceForm.source_book_id" placeholder="例如 729123456 或 https://fanqienovel.com/page/..." />
              </div>
            </div>
            <div class="form-row">
              <div class="form-group">
                <label>书名（可选）</label>
                <input v-model="sourceForm.title" />
              </div>
              <div class="form-group">
                <label>作者（可选）</label>
                <input v-model="sourceForm.author" />
              </div>
            </div>
            <div class="form-group">
              <label>样本文本（可选，抓取失败时建议粘贴前几章）</label>
              <textarea v-model="sourceForm.sample_text" rows="5" placeholder="粘贴章节样本，系统会用它生成单书画像"></textarea>
            </div>
            <button class="primary-btn" @click="createSource" :disabled="acting || !sourceForm.source_book_id">保存样本</button>
          </section>
        </div>

        <div v-if="error" class="error-msg">{{ error }}</div>
        <div v-if="message" class="success-msg">{{ message }}</div>

        <section v-if="activeSubTab === 'scan'" class="panel">
          <div class="section-header">
            <h3>扫描任务</h3>
          </div>
          <table class="data-table">
            <thead>
              <tr><th>ID</th><th>标签</th><th>模式</th><th>状态</th><th>进度</th><th>错误</th><th>操作</th></tr>
            </thead>
            <tbody>
              <tr v-for="job in jobs" :key="job.id">
                <td>{{ job.id }}</td>
                <td>{{ tagName(job.category) }}</td>
                <td>{{ job.mode }}</td>
                <td><span class="status-tag" :class="job.status">{{ job.status }}</span></td>
                <td>{{ job.scanned_count }} / {{ job.target_count }}</td>
                <td class="error-cell">{{ job.error_message || '-' }}</td>
                <td>
                  <button class="action-btn danger" @click="deleteScanJob(job)" :disabled="acting">删除</button>
                </td>
              </tr>
              <tr v-if="jobs.length === 0"><td colspan="7" class="empty-row">暂无任务</td></tr>
            </tbody>
          </table>
        </section>

        <section v-if="activeSubTab === 'scan'" class="panel">
          <div class="section-header panel-header-with-controls stacked">
            <div>
              <h3>书籍样本</h3>
              <p class="section-desc">{{ sampleScopeLabel }}共 {{ filteredSources.length }} 本样本</p>
            </div>
            <div class="section-controls full-width">
              <div class="inline-tag-picker">
                <div class="tag-type-tabs small">
                  <button type="button" :class="{ active: sampleCategory === sampleAllCategory }" @click="sampleCategory = sampleAllCategory">全部标签</button>
                  <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: sampleCategory !== sampleAllCategory && sampleTagType === group.key }" @click="selectTagType('sample', group.key)">{{ group.label }}</button>
                </div>
                <div class="tag-chip-row">
                  <button v-for="tag in visibleTagsByType(sampleTagType, sampleTagKeys)" :key="tag.tag_key" type="button" class="tag-chip small" :class="{ active: sampleCategory === tag.tag_key }" @click="sampleCategory = tag.tag_key">{{ tag.tag_name }}</button>
                  <span v-if="visibleTagsByType(sampleTagType, sampleTagKeys).length === 0" class="empty-tag-row">当前分组暂无样本标签</span>
                </div>
              </div>
              <button class="action-btn" @click="analyzeSelectedCategory" :disabled="acting || !sampleCategory || filteredSources.length === 0">生成单书画像</button>
              <button class="action-btn" @click="synthesizeSelectedCategory" :disabled="acting || !sampleCategory || filteredSources.length === 0">生成聚合画像和规则</button>
              <button class="action-btn danger bulk-delete-btn" @click="deleteSelectedSources" :disabled="acting || selectedSourceIDs.length === 0">删除已选 {{ selectedSourceIDs.length ? `(${selectedSourceIDs.length})` : '' }}</button>
            </div>
          </div>
          <table class="data-table">
            <thead>
              <tr>
                <th class="select-col">
                  <input type="checkbox" :checked="allFilteredSourcesSelected" :disabled="filteredSources.length === 0" aria-label="全选书籍样本" @change="toggleAllSources($event.target.checked)" />
                </th>
                <th>书名</th>
                <th>主标签</th>
                <th>全部标签</th>
                <th>样本章节</th>
                <th>画像版本</th>
                <th>操作</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="source in filteredSources" :key="source.id">
                <td class="select-col">
                  <input v-model="selectedSourceIDs" type="checkbox" :value="source.id" :aria-label="`选择书籍样本 ${source.title || source.id}`" />
                </td>
                <td>{{ source.title }}</td>
                <td>{{ tagName(source.category) }}</td>
                <td>{{ tagNames(source.tags_json).join('、') || '-' }}</td>
                <td>{{ source.chapter_count }}</td>
                <td>{{ source.profile_version || 0 }}</td>
                <td>
                  <button class="action-btn" @click="viewSourceChapters(source)" :disabled="acting">查看章节</button>
                  <button v-if="bookProfileForSource(source.id)" class="action-btn" @click="viewBookProfile(source)" :disabled="acting">查看画像</button>
                </td>
              </tr>
              <tr v-if="filteredSources.length === 0"><td colspan="7" class="empty-row">当前标签暂无样本</td></tr>
            </tbody>
          </table>
        </section>

        <div v-if="activeSubTab === 'intros'" class="grid">
          <section class="panel">
            <h3>扫描简介样本</h3>
            <div class="form-group">
              <label>番茄标签</label>
              <div class="tag-type-tabs">
                <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: introScanTagType === group.key }" @click="introScanTagType = group.key">{{ group.label }}</button>
              </div>
              <div class="tag-chip-grid">
                <button v-for="tag in tagsByType(introScanTagType)" :key="tag.tag_key" type="button" class="tag-chip" :class="{ active: introScanForm.category === tag.tag_key }" @click="introScanForm.category = tag.tag_key">{{ tag.tag_name }}</button>
              </div>
            </div>
            <div class="form-group">
              <label>目标书籍数</label>
              <input v-model.number="introScanForm.target_count" type="number" min="1" max="30" />
            </div>
            <button class="primary-btn" @click="scanIntroSamples" :disabled="acting || !introScanForm.category">扫描简介</button>
            <p class="hint">只读取书名、标签和作品简介，不抓章节正文，适合学习番茄简介钩子和卖点表达。</p>
          </section>

          <section class="panel">
            <h3>生成书籍简介</h3>
            <div class="form-group">
              <label>参考标签</label>
              <div class="tag-type-tabs">
                <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: introTagType === group.key }" @click="selectTagType('intro', group.key)">{{ group.label }}</button>
              </div>
              <div class="tag-chip-grid">
                <button v-for="tag in visibleTagsByType(introTagType, introTagKeys)" :key="tag.tag_key" type="button" class="tag-chip" :class="{ active: hasIntroTag(tag.tag_key) }" @click="toggleIntroTag(tag.tag_key)">{{ tag.tag_name }}</button>
                <span v-if="visibleTagsByType(introTagType, introTagKeys).length === 0" class="empty-tag-row">当前分组暂无简介样本标签</span>
              </div>
              <div v-if="selectedIntroTags.length" class="selected-tag-row">
                <span>已选</span>
                <button v-for="tag in selectedIntroTags" :key="tag.tag_key" type="button" @click="removeIntroTag(tag.tag_key)">{{ tag.tag_name }} ×</button>
              </div>
            </div>
            <div class="form-group">
              <label>新书要求</label>
              <textarea v-model="introRequirement" rows="6" placeholder="例如：大学校园恋爱，都市日常，单女主，男主普通但有幽默感，女主清冷学姐。"></textarea>
            </div>
            <button class="primary-btn" @click="generateIntro" :disabled="acting || introCategories.length === 0 || filteredIntroSamples.length === 0">生成简介</button>
          </section>
        </div>

        <section v-if="activeSubTab === 'intros' && introResult" class="panel intro-result-card">
          <div class="section-header">
            <h3>生成结果</h3>
          </div>
          <h4>{{ introResult.title }}</h4>
          <p>{{ introResult.intro }}</p>
          <div v-if="introResult.selling_points?.length" class="intro-selling-points">
            <span v-for="point in introResult.selling_points" :key="point">{{ point }}</span>
          </div>
        </section>

        <section v-if="activeSubTab === 'intros'" class="panel">
          <div class="section-header panel-header-with-controls stacked">
            <div>
              <h3>简介样本</h3>
              <p class="section-desc">当前选择共 {{ filteredIntroSamples.length }} 条简介样本</p>
            </div>
            <div class="section-controls full-width">
              <div class="inline-tag-picker">
                <div class="tag-type-tabs small">
                  <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: introTagType === group.key }" @click="selectTagType('intro', group.key)">{{ group.label }}</button>
                </div>
                <div class="tag-chip-row">
                  <button v-for="tag in visibleTagsByType(introTagType, introTagKeys)" :key="tag.tag_key" type="button" class="tag-chip small" :class="{ active: hasIntroTag(tag.tag_key) }" @click="toggleIntroTag(tag.tag_key)">{{ tag.tag_name }}</button>
                  <span v-if="visibleTagsByType(introTagType, introTagKeys).length === 0" class="empty-tag-row">当前分组暂无简介样本标签</span>
                </div>
                <div v-if="selectedIntroTags.length" class="selected-tag-row compact">
                  <span>已选</span>
                  <button v-for="tag in selectedIntroTags" :key="tag.tag_key" type="button" @click="removeIntroTag(tag.tag_key)">{{ tag.tag_name }} ×</button>
                </div>
              </div>
              <button class="action-btn danger bulk-delete-btn" @click="deleteSelectedIntroSamples" :disabled="acting || selectedIntroSampleIDs.length === 0">删除已选 {{ selectedIntroSampleIDs.length ? `(${selectedIntroSampleIDs.length})` : '' }}</button>
            </div>
          </div>
          <table class="data-table">
            <thead>
              <tr>
                <th class="select-col">
                  <input type="checkbox" :checked="allFilteredIntroSamplesSelected" :disabled="filteredIntroSamples.length === 0" aria-label="全选简介样本" @change="toggleAllIntroSamples($event.target.checked)" />
                </th>
                <th>书名</th>
                <th>主标签</th>
                <th>全部标签</th>
                <th>简介</th>
              </tr>
            </thead>
            <tbody>
              <tr v-for="sample in filteredIntroSamples" :key="sample.id">
                <td class="select-col">
                  <input v-model="selectedIntroSampleIDs" type="checkbox" :value="sample.id" :aria-label="`选择简介样本 ${sample.title || sample.id}`" />
                </td>
                <td>{{ sample.title }}</td>
                <td>{{ tagName(sample.category) }}</td>
                <td>{{ tagNames(sample.tags_json).join('、') || '-' }}</td>
                <td class="intro-cell">
                  <button type="button" class="intro-preview-btn" @click="viewIntroSample(sample)">{{ sample.intro }}</button>
                </td>
              </tr>
              <tr v-if="filteredIntroSamples.length === 0"><td colspan="5" class="empty-row">当前选择暂无简介样本</td></tr>
            </tbody>
          </table>
        </section>

        <section v-if="activeSubTab === 'profiles'" class="panel">
          <div class="section-header panel-header-with-controls stacked">
            <div>
              <h3>聚合画像</h3>
              <p class="section-desc">按标签查看已生成的聚合画像</p>
            </div>
            <div class="section-controls full-width">
              <div class="inline-tag-picker">
                <div class="tag-type-tabs small">
                  <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: profileTagType === group.key }" @click="selectTagType('profile', group.key)">{{ group.label }}</button>
                </div>
                <div class="tag-chip-row">
                  <button v-for="tag in visibleTagsByType(profileTagType, profileTagKeys)" :key="tag.tag_key" type="button" class="tag-chip small" :class="{ active: hasProfileTag(tag.tag_key) }" @click="toggleProfileTag(tag.tag_key)">{{ tag.tag_name }}</button>
                  <span v-if="visibleTagsByType(profileTagType, profileTagKeys).length === 0" class="empty-tag-row">当前分组暂无画像标签</span>
                </div>
                <div v-if="selectedProfileTags.length" class="selected-tag-row compact">
                  <span>已选</span>
                  <button v-for="tag in selectedProfileTags" :key="tag.tag_key" type="button" @click="removeProfileTag(tag.tag_key)">{{ tag.tag_name }} ×</button>
                </div>
              </div>
              <button class="action-btn danger bulk-delete-btn" @click="deleteSelectedProfiles" :disabled="acting || profileCategories.length === 0">删除已选标签画像 {{ profileCategories.length ? `(${profileCategories.length})` : '' }}</button>
            </div>
          </div>
          <div class="profile-grid">
            <div v-for="profile in filteredProfiles" :key="profile.id" class="profile-card">
              <div class="profile-card-head">
                <div class="profile-title">{{ tagName(profile.category) }}</div>
                <button class="action-btn danger" @click="deleteProfile(profile)" :disabled="acting">删除</button>
              </div>
              <div class="profile-meta">{{ profile.source_count }} 本书 · {{ profile.sample_chapter_count }} 章 · 置信度 {{ formatConfidence(profile.confidence) }}</div>
              <p>{{ profile.writer_brief || profile.profile_summary || '暂无摘要' }}</p>
            </div>
            <div v-if="filteredProfiles.length === 0" class="empty-card">当前标签暂无聚合画像。请先在“书籍样本”中生成单书画像，再点击“生成聚合画像和规则”。</div>
          </div>
        </section>

        <section v-if="activeSubTab === 'rules'" class="panel">
          <div class="section-header panel-header-with-controls stacked">
            <div>
              <h3>写作规则</h3>
              <p class="section-desc">按标签查看可注入写作链路的规则</p>
            </div>
            <div class="section-controls full-width">
              <div class="inline-tag-picker">
                <div class="tag-type-tabs small">
                  <button v-for="group in tagGroups" :key="group.key" type="button" :class="{ active: ruleTagType === group.key }" @click="selectTagType('rule', group.key)">{{ group.label }}</button>
                </div>
                <div class="tag-chip-row">
                  <button v-for="tag in visibleTagsByType(ruleTagType, ruleTagKeys)" :key="tag.tag_key" type="button" class="tag-chip small" :class="{ active: hasRuleTag(tag.tag_key) }" @click="toggleRuleTag(tag.tag_key)">{{ tag.tag_name }}</button>
                  <span v-if="visibleTagsByType(ruleTagType, ruleTagKeys).length === 0" class="empty-tag-row">当前分组暂无规则标签</span>
                </div>
                <div v-if="selectedRuleTags.length" class="selected-tag-row compact">
                  <span>已选</span>
                  <button v-for="tag in selectedRuleTags" :key="tag.tag_key" type="button" @click="removeRuleTag(tag.tag_key)">{{ tag.tag_name }} ×</button>
                </div>
              </div>
              <button class="action-btn danger bulk-delete-btn" @click="deleteSelectedRules" :disabled="acting || ruleCategories.length === 0">删除已选标签规则 {{ ruleCategories.length ? `(${ruleCategories.length})` : '' }}</button>
            </div>
          </div>
          <div class="rule-list">
            <div v-for="rule in filteredRules" :key="rule.id" class="rule-card">
              <div class="rule-head">
                <div>
                  <span class="rule-type" :title="rule.rule_type">{{ ruleTypeLabel(rule.rule_type) }}</span>
                  <strong>{{ rule.title }}</strong>
                </div>
                <button class="action-btn danger" @click="deleteRule(rule)" :disabled="acting">删除</button>
              </div>
              <p>{{ rule.content }}</p>
              <div class="rule-meta">权重 {{ rule.weight }} · 置信度 {{ formatConfidence(rule.confidence) }} · {{ tagName(rule.category) }}</div>
            </div>
            <div v-if="filteredRules.length === 0" class="empty-card">当前标签暂无规则</div>
          </div>
        </section>
    </div>

    <div v-if="viewingBookProfile" class="modal-overlay" @click.self="viewingBookProfile = null">
      <div class="profile-modal">
        <div class="profile-modal-head">
          <div>
            <h3>{{ viewingBookProfile.title }}</h3>
            <p>{{ tagNames(viewingBookProfile.profile.tags_json).join('、') || '暂无标签' }} · 置信度 {{ formatConfidence(viewingBookProfile.profile.confidence) }}</p>
          </div>
          <button class="action-btn" @click="viewingBookProfile = null">关闭</button>
        </div>
        <pre>{{ viewingBookProfile.profile.profile_markdown || '暂无单书画像内容' }}</pre>
      </div>
    </div>

    <div v-if="viewingChapterSamples" class="modal-overlay" @click.self="viewingChapterSamples = null">
      <div class="chapter-modal">
        <div class="profile-modal-head">
          <div>
            <h3>{{ viewingChapterSamples.title }}</h3>
            <p>{{ viewingChapterSamples.chapters.length }} 章样本</p>
          </div>
          <button class="action-btn" @click="viewingChapterSamples = null">关闭</button>
        </div>
        <div v-if="viewingChapterSamples.chapters.length" class="chapter-viewer">
          <aside class="chapter-list">
            <button
              v-for="(chapter, idx) in viewingChapterSamples.chapters"
              :key="chapter.id"
              type="button"
              :class="{ active: selectedChapterIndex === idx }"
              @click="selectedChapterIndex = idx"
            >
              <span>第 {{ chapter.chapter_no }} 章</span>
              <strong>{{ chapter.title || '未命名章节' }}</strong>
            </button>
          </aside>
          <section class="chapter-content">
            <h4>{{ activeChapter?.title || `第 ${activeChapter?.chapter_no || ''} 章` }}</h4>
            <div class="chapter-meta">
              <span>{{ activeChapter?.word_count || 0 }} 字</span>
              <span>{{ activeChapter?.paragraph_count || 0 }} 段</span>
              <span>对话 {{ formatRatio(activeChapter?.dialogue_ratio) }}</span>
            </div>
            <pre>{{ activeChapter?.content || '暂无正文' }}</pre>
          </section>
        </div>
        <div v-else class="empty-card">暂无章节正文样本。请先确认本地 Chrome 已登录番茄并重新抓取章节。</div>
      </div>
    </div>

    <div v-if="viewingIntroSample" class="modal-overlay" @click.self="viewingIntroSample = null">
      <div class="profile-modal intro-modal">
        <div class="profile-modal-head">
          <div>
            <h3>{{ viewingIntroSample.title }}</h3>
            <p>{{ tagNames(viewingIntroSample.tags_json).join('、') || '暂无标签' }}</p>
          </div>
          <button class="action-btn" @click="viewingIntroSample = null">关闭</button>
        </div>
        <pre>{{ viewingIntroSample.intro || '暂无简介' }}</pre>
      </div>
    </div>

    <div v-if="generationProgress.visible" class="modal-overlay" @click.self="closeGenerationProgress">
      <div class="progress-modal">
        <div class="profile-modal-head">
          <div>
            <h3>{{ generationProgress.title }}</h3>
            <p>{{ generationProgress.running ? '正在执行，请保持页面打开' : '任务已结束' }}</p>
          </div>
          <button v-if="!generationProgress.running" class="action-btn" @click="closeGenerationProgress">关闭</button>
        </div>
        <div class="progress-body">
          <div class="progress-row">
            <span>{{ generationProgress.completed }} / {{ generationProgress.total }}</span>
            <strong>{{ progressPercent }}%</strong>
          </div>
          <div class="progress-track">
            <div class="progress-bar" :style="{ width: progressPercent + '%' }"></div>
          </div>
          <div class="progress-current">{{ generationProgress.current || '准备中...' }}</div>
          <div class="progress-stats">
            <span>成功 {{ generationProgress.success }}</span>
            <span>跳过 {{ generationProgress.skipped }}</span>
            <span>失败 {{ generationProgress.failed }}</span>
          </div>
          <div v-if="generationProgress.errors.length" class="progress-errors">
            <div v-for="err in generationProgress.errors.slice(0, 5)" :key="err">{{ err }}</div>
          </div>
          <div class="progress-actions">
            <button v-if="generationProgress.running" class="action-btn danger" @click="stopRadarGeneration" :disabled="generationProgress.stopping">{{ generationProgress.stopping ? '停止中...' : '停止生成' }}</button>
            <button v-else class="primary-btn" @click="closeGenerationProgress">完成</button>
          </div>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup>
import { computed, ref, reactive, onMounted, onUnmounted } from 'vue'
const props = defineProps({
  activeTab: {
    type: String,
    default: 'scan',
  },
})
const tags = ref([])
const jobs = ref([])
const sources = ref([])
const bookProfiles = ref([])
const introSamples = ref([])
const profiles = ref([])
const rules = ref([])
const llmConfigs = ref([])
const radarModelID = ref(0)
const sampleCategory = ref('')
const profileCategories = ref([])
const ruleCategories = ref([])
const scanTagType = ref('plot')
const sampleTagType = ref('plot')
const profileTagType = ref('plot')
const ruleTagType = ref('plot')
const introScanTagType = ref('plot')
const introTagType = ref('plot')
const introCategories = ref([])
const introRequirement = ref('')
const introResult = ref(null)
const selectedSourceIDs = ref([])
const selectedIntroSampleIDs = ref([])
const loading = ref(false)
const acting = ref(false)
const error = ref('')
const message = ref('')
const viewingBookProfile = ref(null)
const viewingChapterSamples = ref(null)
const selectedChapterIndex = ref(0)
const viewingIntroSample = ref(null)
let overviewPollTimer = null
let activeRadarAbortController = null

const generationProgress = reactive({
  visible: false,
  running: false,
  stopping: false,
  title: '',
  current: '',
  total: 0,
  completed: 0,
  success: 0,
  skipped: 0,
  failed: 0,
  errors: [],
})

const headers = { 'Content-Type': 'application/json' }
const sampleAllCategory = '__all__'

const scanForm = reactive({ platform: 'fanqie', category: '', target_count: 5 })
const introScanForm = reactive({ platform: 'fanqie', category: '', target_count: 10 })
const sourceForm = reactive({
  platform: 'fanqie',
  source_book_id: '',
  title: '',
  author: '',
  category: '',
  sample_text: '',
})

const hasActiveJobs = computed(() => jobs.value.some(job => job.status === 'queued' || job.status === 'running'))
const activeSubTab = computed(() => ['scan', 'profiles', 'rules', 'intros'].includes(props.activeTab) ? props.activeTab : 'scan')
const bookProfileMap = computed(() => {
  const m = new Map()
  for (const profile of bookProfiles.value) {
    if (!m.has(profile.source_id) || (profile.version || 0) > (m.get(profile.source_id).version || 0)) {
      m.set(profile.source_id, profile)
    }
  }
  return m
})
const filteredSources = computed(() => {
  if (!sampleCategory.value || sampleCategory.value === sampleAllCategory) return sources.value
  return sources.value.filter(source => source.category === sampleCategory.value || parseTags(source.tags_json).includes(sampleCategory.value))
})
const filteredSourceIDs = computed(() => filteredSources.value.map(source => source.id))
const allFilteredSourcesSelected = computed(() => filteredSourceIDs.value.length > 0 && filteredSourceIDs.value.every(id => selectedSourceIDs.value.includes(id)))
const sampleScopeLabel = computed(() => sampleCategory.value === sampleAllCategory ? '全部标签' : '当前标签')
const filteredIntroSamples = computed(() => {
  if (introCategories.value.length === 0) return introSamples.value
  const selected = new Set(introCategories.value)
  return introSamples.value.filter(sample => selected.has(sample.category) || parseTags(sample.tags_json).some(tag => selected.has(tag)))
})
const filteredIntroSampleIDs = computed(() => filteredIntroSamples.value.map(sample => sample.id))
const allFilteredIntroSamplesSelected = computed(() => filteredIntroSampleIDs.value.length > 0 && filteredIntroSampleIDs.value.every(id => selectedIntroSampleIDs.value.includes(id)))
const progressPercent = computed(() => generationProgress.total > 0 ? Math.round((generationProgress.completed / generationProgress.total) * 100) : 0)
const activeChapter = computed(() => viewingChapterSamples.value?.chapters?.[selectedChapterIndex.value] || null)
const filteredProfiles = computed(() => {
  if (profileCategories.value.length === 0) return profiles.value
  const selected = new Set(profileCategories.value)
  return profiles.value.filter(profile => selected.has(profile.category))
})
const filteredRules = computed(() => {
  if (ruleCategories.value.length === 0) return rules.value
  const selected = new Set(ruleCategories.value)
  return rules.value.filter(rule => selected.has(rule.category))
})
const selectedProfileTags = computed(() => {
  const selected = new Set(profileCategories.value)
  return tags.value.filter(tag => selected.has(tag.tag_key))
})
const selectedRuleTags = computed(() => {
  const selected = new Set(ruleCategories.value)
  return tags.value.filter(tag => selected.has(tag.tag_key))
})
const selectedIntroTags = computed(() => {
  const selected = new Set(introCategories.value)
  return tags.value.filter(tag => selected.has(tag.tag_key))
})
const sampleTagKeys = computed(() => {
  const keys = []
  for (const source of sources.value) {
    if (source.category) keys.push(source.category)
    keys.push(...parseTags(source.tags_json))
  }
  return uniqueKnownTagKeys(keys)
})
const profileTagKeys = computed(() => uniqueKnownTagKeys(profiles.value.map(profile => profile.category)))
const ruleTagKeys = computed(() => uniqueKnownTagKeys(rules.value.map(rule => rule.category)))
const introTagKeys = computed(() => {
  const keys = []
  for (const sample of introSamples.value) {
    if (sample.category) keys.push(sample.category)
    keys.push(...parseTags(sample.tags_json))
  }
  return uniqueKnownTagKeys(keys)
})
const tagGroups = [
  { key: 'plot', label: '情节' },
  { key: 'role', label: '角色' },
  { key: 'theme', label: '主题' },
]
const relatedTagKeys = computed(() => {
  const keys = []
  for (const source of filteredSources.value) {
    if (source.category) keys.push(source.category)
    keys.push(...parseTags(source.tags_json))
  }
  return uniqueKnownTagKeys(keys)
})

onMounted(async () => {
  await Promise.all([loadOverview(), loadModelConfigs()])
  startOverviewPolling()
})

onUnmounted(() => {
  stopOverviewPolling()
})

async function api(path, options = {}) {
  const res = await fetch('/api/v1' + path, {
    ...options,
    headers: { ...headers, ...(options.headers || {}) },
  })
  const data = await res.json().catch(() => ({}))
  if (!res.ok) {
    throw new Error(data.message || data.error?.message || data.error || '请求失败')
  }
  return data
}

async function loadOverview() {
  loading.value = true
  error.value = ''
  try {
    const data = await api('/radar/overview')
    tags.value = data.tags || []
    jobs.value = data.jobs || []
    sources.value = data.sources || []
    bookProfiles.value = data.book_profiles || []
    introSamples.value = data.intro_samples || []
    profiles.value = data.profiles || []
    rules.value = data.rules || []
    pruneSelectedSources()
    pruneSelectedIntroSamples()
    ensureCategorySelections()
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function ensureCategorySelections() {
  const fallback = scanForm.category || tags.value[0]?.tag_key || ''
  if (!sampleCategory.value || (sampleCategory.value !== sampleAllCategory && !sampleTagKeys.value.includes(sampleCategory.value))) sampleCategory.value = sampleTagKeys.value[0] || fallback
  introCategories.value = introCategories.value.filter(key => introTagKeys.value.includes(key))
  if (introCategories.value.length === 0 && introTagKeys.value.length > 0) introCategories.value = [introTagKeys.value[0]]
  profileCategories.value = profileCategories.value.filter(key => profileTagKeys.value.includes(key))
  if (profileCategories.value.length === 0 && profileTagKeys.value.length > 0) profileCategories.value = [profileTagKeys.value[0]]
  ruleCategories.value = ruleCategories.value.filter(key => ruleTagKeys.value.includes(key))
  if (ruleCategories.value.length === 0 && ruleTagKeys.value.length > 0) ruleCategories.value = [ruleTagKeys.value[0]]
  if (!scanForm.category && fallback) scanForm.category = fallback
  if (!introScanForm.category && fallback) introScanForm.category = fallback
  scanTagType.value = tagTypeOf(scanForm.category, scanTagType.value)
  introScanTagType.value = tagTypeOf(introScanForm.category, introScanTagType.value)
  introTagType.value = tagTypeOf(introCategories.value[0], introTagType.value)
  sampleTagType.value = tagTypeOf(sampleCategory.value, sampleTagType.value)
  profileTagType.value = tagTypeOf(profileCategories.value[0], profileTagType.value)
  ruleTagType.value = tagTypeOf(ruleCategories.value[0], ruleTagType.value)
}

async function loadModelConfigs() {
  try {
    llmConfigs.value = await api('/llm-configs')
  } catch {
    llmConfigs.value = []
  }
}

function startOverviewPolling() {
  stopOverviewPolling()
  overviewPollTimer = setInterval(async () => {
    if (!hasActiveJobs.value || loading.value) return
    await loadOverview()
  }, 3000)
}

function stopOverviewPolling() {
  if (overviewPollTimer) {
    clearInterval(overviewPollTimer)
    overviewPollTimer = null
  }
}

async function startScan() {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api('/radar/scan-jobs', {
      method: 'POST',
      body: JSON.stringify({ ...scanForm, ...radarModelPayload() }),
    })
    message.value = '扫描任务已创建，稍后刷新查看进度'
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function openFanqieBrowserPage() {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    const data = await api('/radar/browser/open', { method: 'POST' })
    if (!data.connected) {
      throw new Error(`${data.message || '无法打开本地 Chrome'}。请先用命令启动 Chrome：open -na "Google Chrome" --args --remote-debugging-port=9222 --user-data-dir="$HOME/.whwriter-chrome"`)
    }
    message.value = data.message || '已打开番茄登录/验证页'
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function createSource() {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api('/radar/sources', {
      method: 'POST',
      body: JSON.stringify(sourceForm),
    })
    sourceForm.source_book_id = ''
    sourceForm.title = ''
    sourceForm.author = ''
    sourceForm.sample_text = ''
    message.value = '书籍样本已保存'
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function analyzeSelectedCategory() {
  if (!sampleCategory.value) return
  const scopedSources = filteredSources.value
  const targets = scopedSources.filter(source => !(source.profile_version > 0))
  if (targets.length === 0) {
    message.value = '当前范围内的书籍都已有单书画像，无需重复生成'
    return
  }
  acting.value = true
  error.value = ''
  message.value = ''
  startGenerationProgress('生成单书画像', scopedSources.length)
  generationProgress.skipped = scopedSources.length - targets.length
  generationProgress.completed = generationProgress.skipped
  try {
    for (const source of targets) {
      if (generationProgress.stopping) break
      generationProgress.current = source.title || `书籍样本 #${source.id}`
      activeRadarAbortController = new AbortController()
      try {
        await api(`/radar/sources/${source.id}/analyze`, {
          method: 'POST',
          body: JSON.stringify(radarModelPayload()),
          signal: activeRadarAbortController.signal,
        })
        generationProgress.success++
      } catch (e) {
        if (isAbortError(e)) {
          generationProgress.stopping = true
          break
        }
        generationProgress.failed++
        generationProgress.errors.push(`${generationProgress.current}: ${e.message}`)
      } finally {
        activeRadarAbortController = null
        generationProgress.completed++
      }
    }
    finishGenerationProgress()
    const stoppedText = generationProgress.stopping ? '，已停止' : ''
    message.value = `已生成 ${generationProgress.success} 本书的单书画像，失败 ${generationProgress.failed} 本${stoppedText}。下一步请点击“生成聚合画像和规则”。`
    if (sampleCategory.value !== sampleAllCategory) {
      profileCategories.value = [sampleCategory.value]
      ruleCategories.value = [sampleCategory.value]
    }
    profileTagType.value = sampleTagType.value
    ruleTagType.value = sampleTagType.value
    await loadOverview()
  } catch (e) {
    error.value = e.message
    finishGenerationProgress()
  } finally {
    acting.value = false
  }
}

async function synthesizeSelectedCategory() {
  if (!sampleCategory.value) return
  const keys = relatedTagKeys.value.length ? relatedTagKeys.value : [sampleCategory.value]
  acting.value = true
  error.value = ''
  message.value = ''
  startGenerationProgress('生成聚合画像和规则', keys.length)
  try {
    for (const key of keys) {
      if (generationProgress.stopping) break
      generationProgress.current = tagName(key)
      activeRadarAbortController = new AbortController()
      try {
        await api('/radar/synthesize', {
          method: 'POST',
          body: JSON.stringify({ platform: 'fanqie', category: key, ...radarModelPayload() }),
          signal: activeRadarAbortController.signal,
        })
        generationProgress.success++
      } catch (e) {
        if (isAbortError(e)) {
          generationProgress.stopping = true
          break
        }
        generationProgress.failed++
        generationProgress.errors.push(`${generationProgress.current}: ${e.message}`)
      } finally {
        activeRadarAbortController = null
        generationProgress.completed++
      }
    }
    finishGenerationProgress()
    const stoppedText = generationProgress.stopping ? '，已停止' : ''
    message.value = `已更新 ${generationProgress.success} 个标签的聚合画像和写作规则，失败 ${generationProgress.failed} 个${stoppedText}`
    const succeededKeys = keys.slice(0, generationProgress.completed).filter((_, idx) => idx < generationProgress.success + generationProgress.failed)
    profileCategories.value = succeededKeys.length ? succeededKeys : keys
    ruleCategories.value = succeededKeys.length ? succeededKeys : keys
    await loadOverview()
  } catch (e) {
    error.value = e.message
    finishGenerationProgress()
  } finally {
    acting.value = false
  }
}

async function scanIntroSamples() {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    const data = await api('/radar/intros/scan', {
      method: 'POST',
      body: JSON.stringify(introScanForm),
    })
    message.value = `已扫描 ${data.count || 0} 条简介样本`
    introCategories.value = [introScanForm.category]
    introTagType.value = introScanTagType.value
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function generateIntro() {
  if (introCategories.value.length === 0) return
  acting.value = true
  error.value = ''
  message.value = ''
  introResult.value = null
  try {
    const data = await api('/radar/intros/generate', {
      method: 'POST',
      body: JSON.stringify({
        platform: 'fanqie',
        tags: introCategories.value,
        requirement: introRequirement.value,
        ...radarModelPayload(),
      }),
    })
    introResult.value = data.result
    message.value = '简介已生成'
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

function radarModelPayload() {
  return radarModelID.value > 0 ? { model_id: radarModelID.value } : {}
}

function startGenerationProgress(title, total) {
  generationProgress.visible = true
  generationProgress.running = true
  generationProgress.stopping = false
  generationProgress.title = title
  generationProgress.current = ''
  generationProgress.total = total
  generationProgress.completed = 0
  generationProgress.success = 0
  generationProgress.skipped = 0
  generationProgress.failed = 0
  generationProgress.errors = []
}

function finishGenerationProgress() {
  generationProgress.running = false
  generationProgress.current = generationProgress.stopping ? '已停止' : '已完成'
  activeRadarAbortController = null
}

function stopRadarGeneration() {
  generationProgress.stopping = true
  if (activeRadarAbortController) {
    activeRadarAbortController.abort()
  }
}

function closeGenerationProgress() {
  if (generationProgress.running) return
  generationProgress.visible = false
}

function isAbortError(e) {
  return e?.name === 'AbortError'
}

function bookProfileForSource(sourceID) {
  return bookProfileMap.value.get(sourceID)
}

function viewBookProfile(source) {
  const profile = bookProfileForSource(source.id)
  if (!profile) return
  viewingBookProfile.value = {
    title: source.title || `书籍样本 #${source.id}`,
    profile,
  }
}

async function viewSourceChapters(source) {
  if (!source?.id) return
  acting.value = true
  error.value = ''
  try {
    const data = await api(`/radar/sources/${source.id}/chapters`)
    selectedChapterIndex.value = 0
    viewingChapterSamples.value = {
      title: source.title || `书籍样本 #${source.id}`,
      chapters: data.chapters || [],
    }
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

function viewIntroSample(sample) {
  viewingIntroSample.value = sample
}

async function deleteScanJob(job) {
  if (!job) return
  if (!confirm(`确定删除扫描任务 #${job.id}？`)) return
  await deleteRadarRecord(`/radar/scan-jobs/${job.id}`, '扫描任务已删除')
}

async function deleteProfile(profile) {
  if (!profile || !confirm(`确定删除「${tagName(profile.category)}」的聚合画像？`)) return
  await deleteRadarRecord(`/radar/profiles/${profile.id}`, '聚合画像已删除')
}

async function deleteRule(rule) {
  if (!rule || !confirm(`确定删除规则「${rule.title || rule.id}」？`)) return
  await deleteRadarRecord(`/radar/rules/${rule.id}`, '写作规则已删除')
}

async function deleteSelectedProfiles() {
  const categories = [...profileCategories.value]
  if (categories.length === 0) return
  if (!confirm(`确定删除已选 ${categories.length} 个标签下的所有聚合画像？`)) return
  await deleteRadarCategories('/radar/profiles/delete', categories, `已删除 ${categories.length} 个标签下的聚合画像`)
}

async function deleteSelectedRules() {
  const categories = [...ruleCategories.value]
  if (categories.length === 0) return
  if (!confirm(`确定删除已选 ${categories.length} 个标签下的所有写作规则？`)) return
  await deleteRadarCategories('/radar/rules/delete', categories, `已删除 ${categories.length} 个标签下的写作规则`)
}

async function deleteSelectedSources() {
  const ids = [...selectedSourceIDs.value]
  if (ids.length === 0) return
  if (!confirm(`确定删除已选的 ${ids.length} 本书籍样本？其章节样本和单书画像也会一起删除。`)) return
  await deleteSourcesByIDs(ids, `已删除 ${ids.length} 本书籍样本`)
}

async function deleteSelectedIntroSamples() {
  const ids = [...selectedIntroSampleIDs.value]
  if (ids.length === 0) return
  if (!confirm(`确定删除已选的 ${ids.length} 条简介样本？`)) return
  await deleteIntroSamplesByIDs(ids, `已删除 ${ids.length} 条简介样本`)
}

async function deleteSourcesByIDs(ids, okMessage) {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api('/radar/sources/delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    })
    selectedSourceIDs.value = selectedSourceIDs.value.filter(id => !ids.includes(id))
    message.value = okMessage
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function deleteIntroSamplesByIDs(ids, okMessage) {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api('/radar/intros/delete', {
      method: 'POST',
      body: JSON.stringify({ ids }),
    })
    selectedIntroSampleIDs.value = selectedIntroSampleIDs.value.filter(id => !ids.includes(id))
    message.value = okMessage
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

function toggleAllSources(checked) {
  const visibleIDs = filteredSourceIDs.value
  if (checked) {
    const next = new Set(selectedSourceIDs.value)
    visibleIDs.forEach(id => next.add(id))
    selectedSourceIDs.value = [...next]
    return
  }
  const visible = new Set(visibleIDs)
  selectedSourceIDs.value = selectedSourceIDs.value.filter(id => !visible.has(id))
}

function pruneSelectedSources() {
  const existing = new Set(sources.value.map(source => source.id))
  selectedSourceIDs.value = selectedSourceIDs.value.filter(id => existing.has(id))
}

function toggleAllIntroSamples(checked) {
  const visibleIDs = filteredIntroSampleIDs.value
  if (checked) {
    const next = new Set(selectedIntroSampleIDs.value)
    visibleIDs.forEach(id => next.add(id))
    selectedIntroSampleIDs.value = [...next]
    return
  }
  const visible = new Set(visibleIDs)
  selectedIntroSampleIDs.value = selectedIntroSampleIDs.value.filter(id => !visible.has(id))
}

function pruneSelectedIntroSamples() {
  const existing = new Set(introSamples.value.map(sample => sample.id))
  selectedIntroSampleIDs.value = selectedIntroSampleIDs.value.filter(id => existing.has(id))
}

async function deleteRadarRecord(path, okMessage) {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api(path, { method: 'DELETE' })
    message.value = okMessage
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

async function deleteRadarCategories(path, categories, okMessage) {
  acting.value = true
  error.value = ''
  message.value = ''
  try {
    await api(path, {
      method: 'POST',
      body: JSON.stringify({ platform: 'fanqie', categories }),
    })
    message.value = okMessage
    await loadOverview()
  } catch (e) {
    error.value = e.message
  } finally {
    acting.value = false
  }
}

function tagName(key) {
  return tags.value.find(tag => tag.tag_key === key)?.tag_name || key
}

function tagsByType(type) {
  return tags.value.filter(tag => (tag.tag_type || tag.category || 'plot') === type)
}

function visibleTagsByType(type, keys) {
  const visible = new Set(keys.value || keys || [])
  return tagsByType(type).filter(tag => visible.has(tag.tag_key))
}

function selectTagType(scope, type) {
  if (scope === 'sample') {
    sampleTagType.value = type
    sampleCategory.value = visibleTagsByType(type, sampleTagKeys)[0]?.tag_key || sampleCategory.value
  } else if (scope === 'profile') {
    profileTagType.value = type
  } else if (scope === 'rule') {
    ruleTagType.value = type
  } else if (scope === 'intro') {
    introTagType.value = type
  }
}

function hasProfileTag(key) {
  return profileCategories.value.includes(key)
}

function toggleProfileTag(key) {
  if (!key) return
  if (profileCategories.value.includes(key)) {
    profileCategories.value = profileCategories.value.filter(item => item !== key)
    return
  }
  profileCategories.value = [...profileCategories.value, key]
}

function removeProfileTag(key) {
  profileCategories.value = profileCategories.value.filter(item => item !== key)
}

function hasRuleTag(key) {
  return ruleCategories.value.includes(key)
}

function toggleRuleTag(key) {
  if (!key) return
  if (ruleCategories.value.includes(key)) {
    ruleCategories.value = ruleCategories.value.filter(item => item !== key)
    return
  }
  ruleCategories.value = [...ruleCategories.value, key]
}

function removeRuleTag(key) {
  ruleCategories.value = ruleCategories.value.filter(item => item !== key)
}

function hasIntroTag(key) {
  return introCategories.value.includes(key)
}

function toggleIntroTag(key) {
  if (!key) return
  if (introCategories.value.includes(key)) {
    introCategories.value = introCategories.value.filter(item => item !== key)
    return
  }
  introCategories.value = [...introCategories.value, key]
}

function removeIntroTag(key) {
  introCategories.value = introCategories.value.filter(item => item !== key)
}

function tagNames(raw) {
  return parseTags(raw).map(tagName)
}

function tagTypeOf(key, fallback = 'plot') {
  return tags.value.find(tag => tag.tag_key === key)?.tag_type || tags.value.find(tag => tag.tag_key === key)?.category || fallback
}

function uniqueKnownTagKeys(values) {
  const known = new Set(tags.value.map(tag => tag.tag_key))
  const seen = new Set()
  const out = []
  for (const value of values) {
    if (!value || !known.has(value) || seen.has(value)) continue
    seen.add(value)
    out.push(value)
  }
  return out
}

function parseTags(raw) {
  try {
    return JSON.parse(raw || '[]')
  } catch {
    return []
  }
}

function formatConfidence(v) {
  return typeof v === 'number' ? Math.round(v * 100) + '%' : '-'
}

function formatRatio(v) {
  return typeof v === 'number' ? Math.round(v * 100) + '%' : '-'
}

function ruleTypeLabel(type) {
  const labels = {
    opening: '开篇',
    pacing: '节奏',
    hook: '钩子',
    dialogue: '对话',
    ending: '章尾',
    scene: '场面',
    taboo: '禁忌',
    style: '文风',
  }
  return labels[type] || type || '规则'
}

</script>

<style scoped>
.radar-panel {
  display: flex;
  flex-direction: column;
  gap: 20px;
}
.radar-toolbar {
  display: flex;
  justify-content: space-between;
  align-items: center;
  gap: 18px;
  padding: 22px 24px;
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 18px;
  box-shadow: 0 1px 3px rgba(15,23,42,0.04);
}
.radar-toolbar h2 {
  color: #1e293b;
  font-size: 22px;
  margin-bottom: 6px;
}
.radar-toolbar p {
  color: #64748b;
  line-height: 1.7;
}
.refresh-btn {
  border: 1px solid #e2e8f0;
  background: #fff;
  border-radius: 8px;
  padding: 8px 14px;
  color: #64748b;
  cursor: pointer;
  flex-shrink: 0;
}
.radar-toolbar-actions {
  display: flex;
  align-items: center;
  gap: 10px;
  flex-shrink: 0;
}
.polling-badge {
  display: inline-flex;
  align-items: center;
  padding: 6px 10px;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 700;
}
.content { padding: 0; }
.panel {
  background: #fff;
  border: 1px solid #e2e8f0;
  border-radius: 14px;
  padding: 22px;
  margin-bottom: 20px;
  box-shadow: 0 1px 3px rgba(0,0,0,0.04);
}
.radar-model-panel {
  margin-bottom: 20px;
}
.grid { display: grid; grid-template-columns: 1fr 1fr; gap: 20px; }
.panel h3 { margin-bottom: 16px; color: #1e293b; }
.panel-header-with-controls {
  align-items: flex-start;
  gap: 16px;
}
.panel-header-with-controls.stacked {
  display: flex;
  flex-direction: column;
}
.section-desc {
  color: #64748b;
  font-size: 12px;
  margin-top: 6px;
}
.section-controls {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
  justify-content: flex-end;
}
.section-controls.full-width {
  width: 100%;
  justify-content: flex-start;
  align-items: flex-start;
}
.button-row {
  display: flex;
  align-items: center;
  gap: 8px;
  flex-wrap: wrap;
}
.section-controls select {
  min-width: 180px;
  padding: 8px 10px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1e293b;
}
.form-row { display: grid; grid-template-columns: 1fr 1fr; gap: 12px; }
.form-group { margin-bottom: 14px; }
.form-group label { display: block; font-size: 13px; color: #475569; margin-bottom: 6px; }
.form-group input, .form-group select, .form-group textarea {
  width: 100%;
  padding: 10px 12px;
  border: 1px solid #e2e8f0;
  border-radius: 8px;
  background: #f8fafc;
  color: #1e293b;
  box-sizing: border-box;
}
.hint { color: #94a3b8; font-size: 12px; line-height: 1.6; margin-top: 10px; }
.tag-type-tabs {
  display: flex;
  gap: 8px;
  flex-wrap: wrap;
  margin-bottom: 10px;
}
.tag-type-tabs button {
  border: 1px solid #e2e8f0;
  background: #f8fafc;
  color: #475569;
  border-radius: 999px;
  padding: 7px 14px;
  cursor: pointer;
  font-weight: 600;
}
.tag-type-tabs button.active {
  color: #ea580c;
  background: #fff1eb;
  border-color: #fed7aa;
}
.tag-type-tabs.small button {
  padding: 5px 10px;
  font-size: 12px;
}
.tag-chip-grid,
.tag-chip-row {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
}
.tag-chip-grid {
  max-height: 160px;
  overflow-y: auto;
  padding-right: 4px;
}
.tag-chip-row {
  max-height: 86px;
  overflow-y: auto;
}
.tag-chip {
  border: 1px solid #e2e8f0;
  border-radius: 999px;
  background: #f8fafc;
  color: #475569;
  padding: 7px 12px;
  cursor: pointer;
  font-size: 13px;
}
.tag-chip.small {
  padding: 5px 9px;
  font-size: 12px;
}
.tag-chip.active {
  color: #2563eb;
  background: #eff6ff;
  border-color: #bfdbfe;
  font-weight: 700;
}
.selected-tag-row {
  display: flex;
  align-items: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 12px;
  color: #64748b;
  font-size: 12px;
}
.selected-tag-row.compact {
  margin-top: 4px;
}
.selected-tag-row button {
  border: none;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  padding: 5px 9px;
  cursor: pointer;
  font-weight: 600;
}
.empty-tag-row {
  color: #94a3b8;
  font-size: 12px;
  padding: 6px 0;
}
.inline-tag-picker {
  display: grid;
  gap: 8px;
  flex: 1;
  min-width: 0;
}
.section-controls.full-width .inline-tag-picker {
  width: 100%;
}
.primary-btn, .action-btn {
  border: none;
  border-radius: 8px;
  cursor: pointer;
}
.primary-btn {
  background: #2563eb;
  color: #fff;
  padding: 10px 18px;
  font-weight: 600;
}
.action-btn {
  background: #eff6ff;
  color: #2563eb;
  padding: 6px 10px;
  margin-right: 6px;
}
.action-btn.danger {
  background: #fef2f2;
  color: #dc2626;
}
.bulk-delete-btn {
  flex-shrink: 0;
  align-self: flex-start;
}
button:disabled { opacity: .5; cursor: not-allowed; }
.error-msg, .success-msg {
  padding: 12px 14px;
  border-radius: 8px;
  margin-bottom: 16px;
}
.error-msg { color: #dc2626; background: #fef2f2; }
.success-msg { color: #15803d; background: #f0fdf4; }
.data-table { width: 100%; border-collapse: collapse; }
.data-table th, .data-table td {
  padding: 12px;
  border-bottom: 1px solid #e2e8f0;
  text-align: left;
  font-size: 13px;
}
.data-table th { color: #475569; font-weight: 700; }
.data-table td {
  color: #334155;
  font-weight: 600;
}
.data-table .select-col {
  width: 40px;
  text-align: center;
}
.data-table input[type="checkbox"] {
  width: 16px;
  height: 16px;
  cursor: pointer;
}
.empty-row, .empty-card { color: #94a3b8; text-align: center; padding: 24px; }
.status-tag {
  display: inline-block;
  padding: 4px 8px;
  border-radius: 999px;
  background: #f1f5f9;
  color: #475569;
  font-size: 12px;
}
.status-tag.succeeded { background: #dcfce7; color: #15803d; }
.status-tag.running, .status-tag.queued { background: #dbeafe; color: #1d4ed8; }
.status-tag.failed { background: #fee2e2; color: #b91c1c; }
.error-cell { max-width: 260px; color: #b91c1c; }
.intro-cell {
  max-width: 520px;
  color: #334155;
  line-height: 1.7;
  white-space: pre-wrap;
}
.intro-preview-btn {
  width: 100%;
  max-height: 4.9em;
  overflow: hidden;
  border: none;
  background: transparent;
  color: #334155;
  cursor: pointer;
  display: -webkit-box;
  font: inherit;
  font-weight: 600;
  line-height: 1.65;
  padding: 0;
  text-align: left;
  line-clamp: 3;
  -webkit-box-orient: vertical;
  -webkit-line-clamp: 3;
}
.intro-preview-btn:hover {
  color: #1d4ed8;
}
.intro-result-card h4 {
  color: #1e293b;
  font-size: 18px;
  margin-bottom: 12px;
}
.intro-result-card p {
  color: #334155;
  line-height: 1.8;
  white-space: pre-wrap;
}
.intro-selling-points {
  display: flex;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 14px;
}
.intro-selling-points span {
  padding: 5px 9px;
  border-radius: 999px;
  background: #eff6ff;
  color: #2563eb;
  font-size: 12px;
  font-weight: 600;
}
.profile-grid { display: grid; grid-template-columns: 1fr; gap: 14px; }
.profile-card, .rule-card {
  border: 1px solid #e2e8f0;
  border-radius: 12px;
  padding: 14px;
  background: #f8fafc;
}
.profile-card-head {
  display: flex;
  align-items: flex-start;
  justify-content: space-between;
  gap: 12px;
  margin-bottom: 6px;
}
.profile-title { font-weight: 700; color: #1e293b; }
.profile-meta, .rule-meta { color: #64748b; font-size: 12px; margin-top: 8px; }
.profile-card p {
  color: #334155;
  line-height: 1.7;
  margin-top: 10px;
  white-space: pre-wrap;
}
.modal-overlay {
  position: fixed;
  inset: 0;
  background: rgba(15, 23, 42, 0.35);
  display: flex;
  align-items: center;
  justify-content: center;
  z-index: 1000;
  padding: 24px;
}
.profile-modal {
  width: min(920px, 92vw);
  max-height: 86vh;
  overflow: hidden;
  background: #fff;
  border-radius: 16px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 20px 50px rgba(15, 23, 42, 0.18);
  display: flex;
  flex-direction: column;
}
.intro-modal {
  width: min(760px, 92vw);
}
.chapter-modal {
  width: min(1120px, 94vw);
  height: min(780px, 88vh);
  overflow: hidden;
  background: #fff;
  border-radius: 16px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 20px 50px rgba(15, 23, 42, 0.18);
  display: flex;
  flex-direction: column;
}
.progress-modal {
  width: min(640px, 92vw);
  max-height: 86vh;
  overflow: hidden;
  background: #fff;
  border-radius: 16px;
  border: 1px solid #e2e8f0;
  box-shadow: 0 20px 50px rgba(15, 23, 42, 0.18);
  display: flex;
  flex-direction: column;
}
.profile-modal-head {
  display: flex;
  justify-content: space-between;
  gap: 16px;
  align-items: flex-start;
  padding: 18px 20px;
  border-bottom: 1px solid #e2e8f0;
}
.profile-modal-head h3 {
  margin: 0 0 6px;
}
.profile-modal-head p {
  color: #94a3b8;
  font-size: 12px;
}
.profile-modal pre {
  margin: 0;
  padding: 20px;
  overflow: auto;
  white-space: pre-wrap;
  line-height: 1.7;
  color: #334155;
  background: #f8fafc;
}
.chapter-viewer {
  min-height: 0;
  flex: 1;
  display: grid;
  grid-template-columns: 280px minmax(0, 1fr);
  background: #f8fafc;
}
.chapter-list {
  min-height: 0;
  overflow-y: auto;
  border-right: 1px solid #e2e8f0;
  background: #fff;
  padding: 12px;
}
.chapter-list button {
  width: 100%;
  display: grid;
  gap: 4px;
  border: 1px solid transparent;
  border-radius: 10px;
  background: transparent;
  color: #334155;
  cursor: pointer;
  padding: 10px 12px;
  text-align: left;
}
.chapter-list button.active {
  background: #eff6ff;
  border-color: #bfdbfe;
  color: #1d4ed8;
}
.chapter-list span {
  font-size: 12px;
  font-weight: 700;
}
.chapter-list strong {
  font-size: 13px;
  line-height: 1.45;
}
.chapter-content {
  min-width: 0;
  min-height: 0;
  overflow: auto;
  padding: 22px;
}
.chapter-content h4 {
  color: #1e293b;
  font-size: 18px;
  margin-bottom: 10px;
}
.chapter-meta {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  color: #64748b;
  font-size: 12px;
  font-weight: 700;
  margin-bottom: 16px;
}
.chapter-content pre {
  margin: 0;
  white-space: pre-wrap;
  color: #334155;
  line-height: 1.9;
  font-family: inherit;
}
.progress-body {
  padding: 20px;
  display: grid;
  gap: 14px;
}
.progress-row,
.progress-stats,
.progress-actions {
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 10px;
}
.progress-row {
  color: #334155;
  font-weight: 700;
}
.progress-track {
  height: 10px;
  overflow: hidden;
  border-radius: 999px;
  background: #e2e8f0;
}
.progress-bar {
  height: 100%;
  border-radius: inherit;
  background: #2563eb;
  transition: width 0.2s ease;
}
.progress-current {
  min-height: 22px;
  color: #1e293b;
  font-weight: 700;
  line-height: 1.6;
}
.progress-stats {
  justify-content: flex-start;
  flex-wrap: wrap;
  color: #64748b;
  font-size: 13px;
  font-weight: 700;
}
.progress-errors {
  display: grid;
  gap: 6px;
  max-height: 140px;
  overflow-y: auto;
  padding: 10px 12px;
  border-radius: 10px;
  background: #fef2f2;
  color: #b91c1c;
  font-size: 12px;
  line-height: 1.5;
}
.progress-actions {
  justify-content: flex-end;
}
.rule-list { display: grid; gap: 12px; }
.rule-head { display: flex; justify-content: space-between; gap: 12px; align-items: flex-start; margin-bottom: 6px; }
.rule-head strong {
  color: #1e293b;
}
.rule-card p {
  color: #334155;
  line-height: 1.75;
  margin-top: 8px;
  white-space: pre-wrap;
}
.rule-type {
  background: #dbeafe;
  color: #1d4ed8;
  padding: 3px 7px;
  border-radius: 999px;
  font-size: 12px;
}

@media (max-width: 900px) {
  .grid, .form-row { grid-template-columns: 1fr; }
  .radar-toolbar { align-items: flex-start; flex-direction: column; }
  .radar-toolbar-actions { width: 100%; justify-content: space-between; }
  .panel-header-with-controls { flex-direction: column; }
  .section-controls { width: 100%; justify-content: flex-start; }
}
</style>
