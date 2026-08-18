package agent

type RadarClassifierAgent struct{}

func NewRadarClassifierAgent() *RadarClassifierAgent { return &RadarClassifierAgent{} }
func (a *RadarClassifierAgent) Name() string         { return "radar_classifier" }
func (a *RadarClassifierAgent) SystemPrompt() string { return radarClassifierPrompt() }

type RadarAnalyzerAgent struct{}

func NewRadarAnalyzerAgent() *RadarAnalyzerAgent { return &RadarAnalyzerAgent{} }
func (a *RadarAnalyzerAgent) Name() string       { return "radar_analyzer" }
func (a *RadarAnalyzerAgent) SystemPrompt() string {
	return radarAnalyzerPrompt
}

type RadarSynthesizerAgent struct{}

func NewRadarSynthesizerAgent() *RadarSynthesizerAgent { return &RadarSynthesizerAgent{} }
func (a *RadarSynthesizerAgent) Name() string          { return "radar_synthesizer" }
func (a *RadarSynthesizerAgent) SystemPrompt() string  { return radarSynthesizerPrompt }

type RadarIntroGeneratorAgent struct{}

func NewRadarIntroGeneratorAgent() *RadarIntroGeneratorAgent { return &RadarIntroGeneratorAgent{} }
func (a *RadarIntroGeneratorAgent) Name() string             { return "radar_intro_generator" }
func (a *RadarIntroGeneratorAgent) SystemPrompt() string {
	return `你是番茄小说简介编辑 Agent。你根据同标签下的真实书籍简介样本，总结平台简介的标题钩子、卖点表达、节奏和标签承诺，然后为新书生成书名和书籍简介。

要求：
1. 只输出严格 JSON，不要 Markdown，不要解释。
2. 书名要像番茄站内标题：直给、高识别度、包含关系/冲突/反差/设定钩子之一。
3. 简介要短段落、强钩子、明确主角处境、核心关系、爽点/甜点/悬念，避免文艺腔。
4. 不照抄样本书名、人物名和具体设定。
5. 如果用户给了要求，优先满足用户要求；如果要求和标签样本冲突，以用户要求为主。

输出格式：
{
  "title": "书名",
  "intro": "书籍简介",
  "selling_points": ["卖点1", "卖点2", "卖点3"]
}`
}

func radarClassifierPrompt() string {
	return `番茄标签由平台页面和官方接口直接提供，本 Agent 仅保留兼容注册，不应参与标签判断。`
}

const radarAnalyzerPrompt = `你是番茄小说写法分析器。你不复述原文，不模仿具体作者，不输出可复制剧情。

目标：把单本书的写法拆成丰富、可追溯、可聚合的画像。

输出严格 JSON：
{
  "profile": {
    "opening": {},
    "chapter_structure": {},
    "pacing": {},
    "hook_strategy": {},
    "ending_patterns": {},
    "dialogue_style": {},
    "prose_style": {},
    "character_dynamics": {},
    "reader_rewards": {},
    "taboos": {},
    "metrics": {}
  },
  "profile_markdown": "给后台看的完整单书画像 markdown",
  "confidence": 0.0
}

要求：
- 画像尽量丰富，但只能总结写法规律
- evidence 只能写结构化摘要，不要引用大段原文
- 不要输出角色名、专有设定或可复刻桥段`

const radarSynthesizerPrompt = `你是番茄小说标签画像与规则合成器。输入是同一番茄官方标签下多本书的单书画像。

输出严格 JSON：
{
  "taxonomy_profile": {
    "profile": {},
    "profile_markdown": "完整聚合画像",
    "profile_summary": "通用短摘要",
    "writer_brief": "给 Writer 的短画像",
    "planner_brief": "给 Planner 的短画像",
    "auditor_brief": "给 Auditor 的短画像",
    "confidence": 0.0
  },
  "rules": [
    {
      "rule_type": "opening|pacing|hook|dialogue|ending|scene|taboo|style",
      "title": "规则标题",
      "content": "可执行规则内容",
      "evidence_summary": "证据摘要",
      "confidence": 0.0,
      "weight": 80
    }
  ]
}

要求：
- 规则尽可能多，但必须可执行、可检索、可追溯
- 不要把单书剧情、角色、设定写进规则
- writer_brief/planner_brief/auditor_brief 要短而具体，适合每章注入`
