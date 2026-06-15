# Material And Asset Planning

新建演示文稿或大幅改写页面时，planning layer 必须先激活本素材处理层，再写入或更新 `slide_plan.json`。目标是让 agent 先利用用户已经提供的本地素材和上下文，再决定是否需要内置模板、联网搜索或 XML-native 兜底。

本文件覆盖两类对象：

- `material_inventory`：deck 级素材盘点，记录本地素材、链接、缺口、用途和搜索策略。
- `asset_need`：page 级视觉资产需求，记录每页是否需要图片、图表、截图、图标、文案来源或兜底视觉。

## Core Rules

- 本地素材优先。用户给了文件、目录、截图、PDF、PPTX、文案、数据表、链接或已有 slides 时，先判断能否用它满足背景理解、风格参考、图片素材、文案输入或数据输入。
- 没有合适本地素材时，才考虑联网搜索、内置模板、IconPark 或 XML-native 兜底。联网搜索只用于确实需要外部事实、图片、logo、公开截图或背景补充的场景。
- 素材进入 plan 前必须分类并写清用途；不要把“有文件”直接等同于“应出现在页面上”。
- 页面不能依赖不可获得素材才能完成。每个 `asset_need` 都必须有 `fallback_if_missing`。
- 最终 XML 不能留下空白图片框。真实素材不可用时，立即使用 shape、text、line、table、chart、whiteboard 或图标生成可见兜底。
- 真实图片进入 slides 前必须走支持的上传路径：`slides +media-upload` 或 `+create --slides` 的 `@./path` 占位符。禁止把 http(s) 外链直接写进 `<img src>`。

## Material Roles

将每个输入素材归入以下角色之一；一个素材可以有多个角色，但要分别说明用途。

| Role | 用途 | 常见来源 | 默认处理 |
|------|------|----------|----------|
| `background_reference` | 补充主题背景、事实、约束、术语、受众信息 | PDF、文档、网页、PRD、报告、会议纪要 | 读取/摘要后影响叙事和页面重点 |
| `style_reference` | 参考 PPT 风格、配色、版式、字体层级、页面流 | PDF、PPTX、已有 slides、内置模板 | 提取设计语言，不默认复制正文 |
| `visual_asset` | 可直接进入页面的视觉素材 | 图片、截图、logo、图表、论文 figure | 上传后放入计划区域；不合适则重绘或兜底 |
| `copy_source` | 文案、标题、大纲、讲稿、卖点、结论来源 | Markdown、TXT、Docx、用户 prompt、会议纪要 | 改写成低密度 slide 文案 |
| `data_source` | 生成表格、指标卡、图表的数据来源 | CSV、XLSX、表格截图、结构化数据 | 转成 chart/table/数字卡 |
| `brand_asset` | 品牌识别和视觉约束 | logo、VI 色板、品牌手册 | 影响 `theme_style` / `visual_system` |

## Source Priority

1. 用户显式提供的本地素材。
2. 当前工作区内与任务明确相关的素材。
3. 用户给出的在线 slides/wiki/drive/doc 链接。
4. 内置模板库和 IconPark。
5. 联网搜索公开素材或背景信息。
6. XML-native 兜底视觉。

如果多个来源冲突，用户显式提供的素材优先；无法判断时，在 plan 的 `open_issues` 里记录需要用户确认。

## Local Material Handling

- `.pdf` / `.pptx` 作为风格参考或仿写模板：先用 `lark-drive` 的 `drive +import --type slides` 导入为在线 slides，再用 `slides xml_presentations.get` 回读 XML。导入是写操作，执行前必须确认用户意图。默认只提取页面流、配色、背景、字体层级、布局骨架和 motif，不复制原文案。
- `.png` / `.jpg` / `.jpeg` 等图片：判断是可直接展示、需要裁切/缩放、还是只用于理解。进入 XML 前必须上传或使用 `@./path` 占位符。
- `.csv` / `.xlsx` / 表格数据：优先转成 `<chart>`、表格或数字卡；不要截图化数据，除非用户提供的是不可解析的表格图片。
- `.md` / `.txt` / 文档类内容：作为 `copy_source` 或 `background_reference`，提炼为低密度页面文案，不要整段搬进 slide。
- 已有 online slides：直接回读 XML，把它作为 `style_reference` 或改写对象；不要再走导入。

## Search Policy

联网搜索不是默认动作。只有出现以下情况才搜索：

- 用户要求查找公开事实、行业背景、竞品、logo、截图、图片或最新资料。
- plan 中存在关键视觉缺口，本地素材和内置模板都不能满足。
- 用户给出的主题需要真实世界背景才能避免空泛表达。

搜索结果使用规则：

- 图片素材必须先落到本地，再通过 slides 上传流程进入 XML。
- 背景信息必须转化为 `background_reference` 摘要和页面结论，不要把长文本塞进页面。
- 无法确认版权或来源可靠性时，只作为风格/信息参考，不直接作为可展示图片。
- 如果搜索失败或不适合使用，按 `fallback_if_missing` 生成 XML-native 视觉。

## Plan Shape

Deck 级 `material_inventory` 示例：

```json
{
  "material_inventory": {
    "local_first": true,
    "inputs": [
      {
        "source": "./template.pdf",
        "kind": "style_reference",
        "usage": "Import as slides and extract palette, page flow, typography hierarchy, and motif.",
        "status": "available"
      },
      {
        "source": "./notes.md",
        "kind": "copy_source",
        "usage": "Condense into slide headlines and speaker intent.",
        "status": "available"
      }
    ],
    "missing": [
      {
        "need": "Product screenshot for workflow page",
        "search_policy": "Use local screenshot if provided; otherwise search only if user wants real UI imagery.",
        "fallback_if_missing": "Draw a simplified UI wireframe with labeled panels."
      }
    ]
  }
}
```

Page 级 `asset_need` 示例：

```json
{
  "asset_type": "screenshot",
  "source_preference": "local_first",
  "purpose": "Show the target workflow state instead of describing it with bullets.",
  "candidate_sources": ["./assets/workflow.png"],
  "suggested_query": "product workflow screenshot",
  "fallback_if_missing": "Draw a simplified UI wireframe with three panels and callout labels."
}
```

For a page without a meaningful asset need, use:

```json
{
  "asset_type": "none",
  "source_preference": "none",
  "purpose": "No external or simulated asset needed; the page is text-led.",
  "candidate_sources": [],
  "suggested_query": "",
  "fallback_if_missing": "Use typography, spacing, and simple accent shapes only."
}
```

## Supported Asset Types

- `paper_figure`: figure from a paper or technical article.
- `architecture_diagram`: system components, data flow, dependency map, or model structure.
- `icon`: small semantic symbol for a concept, step, role, or status.
- `logo`: brand, product, team, or customer mark.
- `chart`: line, bar, pie, radar, area, or combo data visual. Note: `<chart>` does not support funnel or scatter; map those to `<whiteboard>` at generation time.
- `infographic`: composed visual explanation, usually combining labels, numbers, and simple shapes.
- `screenshot`: product UI, terminal output, workflow state, or page capture.
- `flow_diagram`: process, sequence, decision tree, or mechanism diagram.
- `style_reference`: page-level use of a layout or motif from a template/reference deck.
- `copy_source`: page-level source text to condense into slide copy.
- `none`: explicitly no asset needed.

Do not invent new asset types unless the user asks for a special visual format. If a need is close to these types, choose the closest one and explain the detail in `purpose`.

## Planning Guidance

Match source type to slide role:

- `architecture-diagram` layout usually pairs with `architecture_diagram` or `flow_diagram`.
- `process-flow` layout usually pairs with `flow_diagram`, `icon`, or `infographic`.
- `comparison` layout often works with `icon`, `chart`, `infographic`, or `style_reference`.
- `timeline` layout often works with `icon`, `chart`, or shape-based milestone markers.
- `big-number` layout often works with `chart`, `data_source`, or `infographic`.
- `image-left-text-right` and `image-right-text-left` can use `screenshot`, `paper_figure`, `logo`, `infographic`, or a style reference layout.

`suggested_query` is a future lookup hint. Execute the search only when the search policy says remote material is needed and local sources are insufficient.

`fallback_if_missing` must be concrete enough to turn into XML, for example:

- "Draw a simplified attention matrix with 5 token labels, semi-transparent cells, and arrows to output token."
- "Use three grouped boxes with arrows from client to gateway to service; add small protocol labels."
- "Render a mini bar chart with 4 bars using shapes and value labels."
- "Use a bordered UI wireframe with product area labels, not an empty image."

Weak fallbacks to avoid:

- "Use a placeholder."
- "Find another image."
- "Leave blank if unavailable."
- "Use generic decoration."

## Plan To XML Contract

When generating XML:

1. Apply `material_inventory` first: local style references, copy sources, data sources, and visual assets decide what each page can use.
2. If a real visual asset exists and the workflow supports it, place it in the planned visual region.
3. If no asset exists, immediately render `fallback_if_missing` with XML-native shapes, text, lines, arrows, tables, whiteboard diagrams, or chart-like elements.
4. Size the fallback to satisfy `visual_focus`; it should be a real page element, not a tiny decoration.
5. Keep text-density limits. Do not compensate for missing assets by adding long bullet text.
6. After creation, fetch the presentation and verify asset pages are not blank and that each planned fallback is visible when no real asset was used.
