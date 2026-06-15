# Planning Layer

新建演示文稿或大幅改写页面时，必须先写 `.lark-slides/plan/<deck-or-task-id>/slide_plan.json`，再生成 XML。这个文件是 deck 的设计中间层，用来把叙事、页面角色、布局、视觉重点和文字密度固定下来，避免从用户提示直接跳到 XML。

小型已有页编辑可豁免，例如只替换一个标题、改一个数字、插入一个块、上传并插入一张图。只要任务会重排多页、生成新 deck、替换整页结构，仍然需要规划层。

## Required Flow

1. 理解用户需求，必要时澄清主题、受众、页数、风格。
2. 激活素材处理层：按 `asset-planning.md` 盘点用户提供的本地素材、可引用链接和缺口素材；本地素材优先，没有合适本地素材时再使用内置模板或联网搜索。
3. 选择唯一 plan 目录：`.lark-slides/plan/<deck-or-task-id>/`。
4. 先创建目录：`mkdir -p .lark-slides/plan/<deck-or-task-id>`。
5. 写入 `.lark-slides/plan/<deck-or-task-id>/slide_plan.json`。
6. 读取 `xml-schema-quick-ref.md`、`visual-planning.md` 和 `asset-planning.md`。
7. 按 plan、visual planning 和 asset planning 规则逐页生成 XML，把 `layout_type`、`visual_focus`、`text_density` 转成具体页面几何和文本量约束，并把缺失素材转成可执行兜底视觉。
8. 创建或大幅改写后，按 `validation-checklist.md` 做显式验证；本文件只要求验证记录能说明 plan 到 XML 的对应关系。

素材和模板不能代替 plan。素材处理层只能影响背景理解、`theme_style`、`visual_system`、页面流、文案输入、布局选择和局部视觉骨架；最终仍必须有 `.lark-slides/plan/<deck-or-task-id>/slide_plan.json`。

如果用户提供 `.pptx` 或 `.pdf`，并称其为 PPT 模板、幻灯片模板、版式/风格参考，先按 `lark-drive` 的 `drive +import --type slides` 导入为在线 slides，在这个slides上进行内容二次创作。

## Plan Path

Use a separate plan directory per deck or task so multiple presentations in the same workspace cannot overwrite each other.

Recommended IDs:

- New deck before creation: title slug plus date/time, such as `q3-review-20260507-1805`.
- Existing PPT rewrite: the `xml_presentation_id`.
- Ambiguous or untitled task: short task slug plus date/time.

Rules:

- Do not reuse `.lark-slides/plan/slide_plan.json` as a shared path.
- Create the directory before writing the file.
- Reuse the same plan path for XML generation and post-create verification for that deck.

## Artifact Lifecycle

`.lark-slides/` is local agent state. It supports recovery, iteration, and later edits, but it should not be treated as source code or committed by default.

Keep:

- `.lark-slides/plan/<deck-or-task-id>/slide_plan.json` after successful creation or major rewrite. The plan is the editable design state for the deck.
- A small manifest when useful for follow-up work, such as `xml_presentation_id`, slide IDs, `revision_id`, plan path, and verification status.

Clean or avoid keeping:

- Transient XML payloads after successful creation and verification. Prefer `/tmp` for throwaway XML, or delete generated XML files after success.
- Stale XML drafts that no longer match the current presentation state.

Exception:

- If creation fails or partially succeeds, keep the relevant XML/debug payloads until recovery is complete. Record `xml_presentation_id` first, then fetch current state before retrying.

## JSON Shape

```json
{
  "presentation_goal": "Explain the proposal and secure approval for the next phase.",
  "audience": "Product and engineering leaders who know the domain but need a concise decision narrative.",
  "theme_style": "Clean business style, light background, restrained blue accent, strong visual hierarchy.",
  "material_inventory": {
    "local_first": true,
    "inputs": [
      {
        "source": "./reference.pdf",
        "kind": "style_reference",
        "usage": "Import as slides, then extract page flow, palette, typography hierarchy, and reusable motif.",
        "status": "available"
      }
    ],
    "missing": [
      {
        "need": "Product UI screenshot",
        "search_policy": "Search only if no local screenshot is available; otherwise use XML-native fallback."
      }
    ]
  },
  "visual_system": {
    "background_strategy": "Content pages use one light base; cover and closing may use a related dark treatment with the same accent system.",
    "motif": "A reusable left accent bar and consistent card/header treatments.",
    "color_roles": {
      "primary": "Used for the dominant structural motif and about 60-70% of visual weight.",
      "secondary": "Used for grouped regions, comparison panels, or supporting categories.",
      "accent": "Used only for key numbers, conclusions, or focus markers."
    }
  },
  "typography_constraints": {
    "title_max_lines": 2,
    "body_max_lines_per_box": 2,
    "footer_max_lines": 1,
    "long_text_handling": "Shorten, split into multiple boxes, or move detail to speaker notes instead of shrinking into a tight box."
  },
  "verification_plan": {
    "check_background_consistency": true,
    "check_text_fit": true,
    "check_visual_focus": true,
    "check_asset_rendering": true
  },
  "slides": [
    {
      "page": 1,
      "title": "Proposal Title",
      "key_message": "The initiative is ready for a focused pilot.",
      "layout_type": "title-cover",
      "visual_focus": "Large title area with one concise supporting statement.",
      "asset_need": {
        "asset_type": "logo",
        "purpose": "Signal product or team identity on the opening page.",
        "suggested_query": "product logo",
        "fallback_if_missing": "Use a small text badge and abstract shape motif instead of a real logo."
      },
      "text_density": "low",
      "speaker_intent": "Frame the decision and establish the deck's point of view."
    }
  ]
}
```

## Required Fields

Top-level fields:

- `presentation_goal`: what the whole deck is trying to achieve.
- `audience`: target readers or listeners and their assumed background.
- `theme_style`: visual tone, palette direction, and professional style.
- `material_inventory`: planning-layer record of local inputs, linked references, chosen usage, missing materials, search policy, and fallbacks. Follow `asset-planning.md`.
- `visual_system`: deck-level visual rules that must stay stable across pages, including background strategy, recurring motif, and color roles.
- `typography_constraints`: deck-level limits for line count, text box density, and how to handle long text before XML generation.
- `verification_plan`: plan-specific checks that must be reflected in the final verification record. Detailed post-create validation lives in `validation-checklist.md`.
- `slides`: ordered page plans.

Each slide must include:

- `page`: 1-based page number.
- `title`: slide title.
- `key_message`: the one idea this page must land.
- `layout_type`: planned page structure.
- `visual_focus`: dominant visual object or region.
- `asset_need`: planning-only structured asset metadata; no search, download, or upload required. Follow `asset-planning.md`.
- `text_density`: `low`, `medium`, or `high`.
- `speaker_intent`: why the speaker needs this page and how it advances the story.

## Layout Vocabulary

Use one of these `layout_type` values unless the user explicitly needs a custom structure:

- `title-cover`
- `section-divider`
- `two-column`
- `image-left-text-right`
- `image-right-text-left`
- `big-number`
- `timeline`
- `comparison`
- `architecture-diagram`
- `process-flow`
- `quote-highlight`
- `conclusion`

The value must affect XML geometry, not just appear as a label. For example, `timeline` should create a horizontal or vertical sequence, `comparison` should create distinct side-by-side regions, and `big-number` should reserve dominant space for a large metric.

## Text Density Rules

- `low`: title plus 1 short statement, or 1-3 very short labels.
- `medium`: title plus 2-4 concise bullets or labeled regions.
- `high`: allowed only when the user needs detail; use tables, columns, or grouped regions instead of a long bullet list.

Do not let all pages become title + bullet slides. For decks of 4 or more pages, aim for at least 4 different `layout_type` values when the content allows it.

Text density must be realistic for the planned geometry. If a page needs long titles, bilingual labels, paper figure captions, legal disclaimers, or dense technical wording, record how the text will be shortened, split, or moved to speaker notes. Do not rely on small font sizes or tight boxes to make text fit.

## Visual System Planning

Before generating XML, define a visual system that can survive the whole deck:

- `background_strategy`: specify the default background for normal content pages, and which page roles may intentionally differ. Do not let pages drift through near-identical but inconsistent background colors.
- `motif`: choose one or two reusable structural devices, such as a side bar, header rail, numbered node, card treatment, diagram lane, or section band. The motif should appear consistently enough that pages feel related.
- `color_roles`: assign primary, secondary, and accent roles. The same color must not mean unrelated things across pages.
- `cover_content_relationship`: if the cover uses a different dark or image-led treatment, state how it connects to content pages through shared colors, motifs, or geometry.
- `closing_relationship`: if the closing page mirrors the cover, state that explicitly so it looks intentional rather than like a new theme.

These are planning constraints, not decoration notes. They must affect coordinates, background fills, shape styles, and text placement in generated XML.

## Iterative Deck State

When continuing an existing deck, update the same plan path rather than creating a new disconnected plan. Keep the plan aligned with what has actually been created.

Recommended optional fields for long-running work:

- `deck_status`: current slide count, target slide count if known, and last verified revision or timestamp.
- `created_slides`: page number, slide id when known, and the page role.
- `assets_used`: source, local path when applicable, uploaded token when known, and which page uses it.
- `open_issues`: known layout, text fit, asset, or consistency risks that still need correction.

Do not hard-code a page number just because a previous deck used that pattern. Plan by page role and evidence need, such as "method overview pages should use a figure when the source has a readable figure" instead of binding screenshots, charts, or diagrams to a fixed page index. The plan should describe decision rules, not a rigid template sequence.

## Asset Planning

`material_inventory` records deck-level source handling before XML generation. `asset_need` records page-level visual needs. Follow `asset-planning.md` for both.

Local materials are preferred over remote search. Common material roles:

- `background_reference`: files or links used to understand the topic, facts, audience, or constraints.
- `style_reference`: PDF/PPTX/slides/templates used for palette, typography, layout, motif, and page flow.
- `visual_asset`: images, screenshots, logos, icons, charts, diagrams, or figures that may appear on slides.
- `copy_source`: text, outline, notes, PRD, report, or transcript used as content input.
- `data_source`: tables, CSV/XLSX, metrics, or structured values used for charts or tables.

If a local `.pdf` / `.pptx` is a style reference, use `lark-drive` to import it as online slides and then use `xml_presentations.get` to read the XML before planning. This is a write operation and requires user intent. Extract design language only by default; do not copy source text unless the user asks.

Use an object for one planned asset, an array for multiple real needs, or `asset_type: "none"` when no asset is useful. Each planned asset must include:

- `asset_type`: one of `paper_figure`, `architecture_diagram`, `icon`, `logo`, `chart`, `infographic`, `screenshot`, `flow_diagram`, or `none`.
- `purpose`: why this asset helps the page's key message.
- `suggested_query`: short future lookup hint only; do not execute it unless separately requested.
- `fallback_if_missing`: concrete XML-native visual plan using shapes, labels, tables, whiteboard diagrams, or placeholder panels.

For detailed rules and examples, read `asset-planning.md`.

Good examples:

- `{"asset_type":"architecture_diagram","purpose":"Explain component relationships.","suggested_query":"service architecture diagram","fallback_if_missing":"Draw a component diagram with grouped boxes, connector arrows, and short labels."}`
- `{"asset_type":"logo","purpose":"Identify the customer context.","suggested_query":"customer logo","fallback_if_missing":"Use a text label in a small badge."}`
- `{"asset_type":"chart","purpose":"Show adoption trend.","suggested_query":"monthly adoption trend chart","fallback_if_missing":"Draw a simple trend line chart with axis labels and data points."}`

## XML Generation Contract

Before writing each slide XML, map the plan fields to concrete decisions:

- `key_message` determines the headline, dominant claim, or main takeaway.
- `material_inventory` determines which local materials, imported style references, remote search results, or fallbacks are allowed to influence the page.
- `layout_type` determines the coordinate structure and element types. Use `visual-planning.md` for concrete layout rules.
- `visual_focus` determines the largest visual region or emphasized object.
- `text_density` caps visible text volume.
- `asset_need` informs placeholder diagrams, icons, charts, screenshots, or shape-based fallback visuals only. Missing real assets must use `fallback_if_missing`, not blank regions.

After creating or rewriting the PPT, use `validation-checklist.md` as the source of truth for verification. The verification record should also mention the plan-specific mapping:

- Page count matches the plan.
- Planned `key_message`, `layout_type`, `visual_focus`, `text_density`, and `asset_need` are represented in XML.
- `visual_system` and `typography_constraints` visibly affected backgrounds, structure, hierarchy, and text placement.
- Any maintained `deck_status`, `created_slides`, `assets_used`, or `open_issues` fields are updated to match the current deck state.
