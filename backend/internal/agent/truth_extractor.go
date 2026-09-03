package agent

import "fmt"

type TruthExtractorAgent struct{}

func NewTruthExtractorAgent() *TruthExtractorAgent {
	return &TruthExtractorAgent{}
}

func (a *TruthExtractorAgent) Name() string { return "truth_extractor" }

func (a *TruthExtractorAgent) SystemPrompt() string {
	return `你是小说真相文件提取 Agent。你只从章节正文中提取可沉淀信息，必须返回严格 JSON，不要输出解释、Markdown 或额外文本。

硬规则：
1. 必须返回 summary，summary.key_events 不能为空。
2. durable_facts 只写未来多章仍成立的长期事实，不写临时状态。
3. 临时状态、当前目标、当前冲突不要写入 durable_facts。
4. hooks 只记录确实形成后续承诺的未解问题、关系冲突、物品线索或人物谜题。
5. 不要补充正文没写到的设定，不要从大纲猜后续剧情。
6. durable_facts.evidence_quote 与 events.evidence_quote 必须逐字复制正文中的连续原文，最多 120 字；没有可靠原文就留空。
7. events 只提取本章实际发生、会影响人物关系或后续剧情的关键事件。
8. 字段缺信息时用空数组或空字符串，不要写“未知”“待定”“同上”。`
}

func (a *TruthExtractorAgent) BuildUserPrompt(bookTitle string, chapterNumber uint, rawOutput string) string {
	return fmt.Sprintf(`请从小说《%s》第 %d 章中提取真相文件增量。

## 输出 JSON 格式
{
  "characters": [{"name": "角色名", "role_type": "protagonist|major|minor", "profile": "一句话简介"}],
  "durable_facts": [{"subject": "主体", "predicate": "关系/属性", "object": "客体/值", "category": "identity|resource|item|rule|relationship", "evidence_quote": "正文原文"}],
  "hooks": [{"hook_id": "H01", "type": "plot|conflict|item|mystery|character", "description": "伏笔描述"}],
  "evidence_notes": [{"title": "线索标题", "kind": "clue|document|observation", "content": "章节中出现的具体细节、证据或文本内容"}],
  "events": [{"title": "事件标题", "event_type": "conflict|discovery|decision|relationship|transition|payoff", "summary": "事件摘要", "participants": ["角色名"], "location": "地点名", "consequence": "事件造成的后果", "evidence_quote": "正文原文"}],
  "summary": {"title": "章节标题", "characters_appeared": "角色1,角色2", "key_events": "关键事件", "state_changes": "状态变化", "hook_activity": "伏笔动态", "mood": "情绪基调", "chapter_type": "过渡|冲突|高潮|收束"}
}

## durable_facts 规则
- 这里只保留中长期有效、未来多章仍应成立的事实。
- 只允许 5 类：identity、resource、item、rule、relationship。
- 当前状态类信息不要写入 durable_facts，这些由状态卡单独维护。
- 章节细节类信息不要写入 durable_facts，应写入 evidence_notes；若形成持续未解问题，应写入 hooks。
- 如果无法确定一条信息能持续至少 3 章，就不要放进 durable_facts。

## 章节输出
%s`, bookTitle, chapterNumber, rawOutput)
}
