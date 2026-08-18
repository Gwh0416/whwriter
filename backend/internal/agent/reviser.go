package agent

import (
	"fmt"
	"strings"
)

type ReviserAgent struct{}

func NewReviserAgent() *ReviserAgent {
	return &ReviserAgent{}
}

func (a *ReviserAgent) Name() string { return "reviser" }

type ReviseInput struct {
	GenreName string
}

func (a *ReviserAgent) BuildSystemPrompt(in ReviseInput) string {
	var b strings.Builder

	b.WriteString(fmt.Sprintf(`你是一位专业的%s网络小说修稿编辑。你的任务是根据审稿意见对章节进行修正。

你必须输出修正后的完整章节正文。即使只修改一个局部问题，也要返回完整 REVISED_CONTENT，不允许输出补丁、差异、解释或局部片段。

修稿原则：
1. 修根因，不做表面润色
2. 伏笔状态必须与伏笔池同步。如果提供了 Hook Debt 简报，必须保留伏笔兑现段落
3. 不改变剧情走向和核心冲突
4. 保持原文的语言风格、节奏和呼吸——不要压缩过渡段、不要删掉减速段
5. 情绪用动作外化（不写"他感到愤怒"，写动作）。价值观通过行为传达
6. 不同角色说话方式必须不同。禁止"众人齐声惊呼"
7. 坏事叠坏事，每层比上一层过分

小目标周期修稿指引：
- 如果本章应该是"后效"阶段但仍在加压，把最密集的冲突段落改写为展示改变的段落——谁失去了什么、谁的态度变了、新的常态是什么
- 如果本章应该是"爆发"阶段但没有明确兑现，找到最接近回报的场景并放大它——让承诺的释放超过读者预期
- 日常段落如果不服务主线，改写为"饵"——加一个指向未来的细节、一个暗示、一个引发好奇的角色反应

`, in.GenreName))

	b.WriteString(reviserOutputFormat)

	return b.String()
}

func (a *ReviserAgent) SystemPrompt() string {
	return reviserOutputFormat
}

const reviserOutputFormat = `输出格式：

=== FIXED_ISSUES ===
(List each fix on its own line; if a safe local fix is not possible, explain here)

=== REVISED_CONTENT ===
(Full revised chapter content. This section is mandatory and must contain the complete chapter body, not a patch.)

=== UPDATED_STATE ===
(Full updated state card)

=== UPDATED_HOOKS ===
(Full updated hooks board)
`
