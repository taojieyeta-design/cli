# Visual Planning

新建演示文稿或大幅改写页面时，在 `slide_plan.json` 完成后、生成 XML 前读取本文件。目标是让 `layout_type`、`visual_focus`、`text_density` 变成实际页面几何，而不是只写在 plan 里。

默认画布按 `960 x 540` 规划。模板 XML 可以覆盖具体坐标，但不能覆盖这些原则：页面要有主视觉区域、文本要受密度约束、不同 `layout_type` 必须产生明显不同的坐标结构。

## Design Ideas

不要生成无设计感的幻灯片。纯白背景 + 标题 + bullets 只能作为极简临时稿，不能作为正式交付。

开始写 XML 前，先在 `slide_plan.json` 里确定 deck 级视觉策略：

- **主题化配色**：配色必须服务本次主题、行业和受众，不要默认蓝色商务风。如果把同一套颜色换到另一个完全不同主题仍然成立，说明配色不够具体。
- **主次比例**：选择 1 个主色承担约 60-70% 视觉权重，1-2 个辅助色承担结构和分区，1 个强调色只用于关键数字、结论或行动点。不要让所有颜色权重相同。
- **背景一致性**：先确定全 deck 的背景策略，默认保持同一明暗基调和底色体系；只有分节、转场或强调页才有意改变背景，并必须通过相同主色、纹理、边栏或 motif 让变化看起来属于同一套设计。无论深浅，都要保证正文、图标和线条对比充足。
- **统一 motif**：选择一个可复用视觉母题贯穿全文，例如粗侧边栏、圆形图标底、半出血图片区、编号节点、卡片左上角色块或大号数字。不要每页换一套装饰语言。

每页至少要有一个视觉元素：图片、图标、图表、表格、流程、对比结构、大号数字、示意图或由 shape 组成的抽象视觉。文本框本身不算主视觉。

可优先考虑这些页面形态：

- **双栏结构**：左文右图或左图右文，视觉区域占 35-45% 宽度。
- **图标行**：图标在色块或圆形底中，右侧是短标题和一句解释。
- **2x2 / 2x3 网格**：适合能力、模块、风险、行动项，每格内容保持同等层级。
- **半出血视觉**：图片或抽象形状占据左/右半屏，文字覆盖或贴边排布。
- **大数字卡片**：关键指标用 60-72pt 数字，下面配 10-14pt 标签。
- **对比列**：before/after、方案 A/B、问题/解法用左右并列，标题和基线严格对齐。
- **时间线/流程图**：步骤用节点和箭头表达，流程方向必须一眼可见。

字体和间距建议：

- 标题 36-44pt，关键结论可更大；正文 14-18pt；注释 10-12pt。
- 正文默认左对齐；只在封面、结尾或大号数字场景中使用居中。
- 页面边距至少 40px；内容块之间保持 24-40px 间距，并在同一 deck 内保持一致。
- 卡片内边距要真实留出空间，不要让文字贴边；对齐 shape 和文字时要考虑文本框 padding。

常见错误必须避免：

- 不要所有页面复用同一种标题 + 三 bullets 版式。
- 不要用低对比文字或低对比图标，例如浅灰字压在浅色背景上。
- 不要让装饰线穿过文字，或让页脚、来源、编号挤压主体内容。
- 不要把素材缺失表现为空白图片框；必须按 `fallback_if_missing` 生成 XML-native 视觉。
- 不要留下模板占位文案、示例公司名、示例日期或与用户主题无关的原模板内容。

## Core Rules

- `layout_type` must change geometry: element positions, region sizes, alignment, and visual rhythm must differ across page types.
- `visual_focus` determines the largest or highest-contrast region. It can be an image, diagram, metric, quote, table, or shape-based placeholder.
- `text_density` caps visible text:
  - `low`: title plus one short statement, or 1-3 labels.
  - `medium`: title plus 2-4 concise bullets or labeled regions.
  - `high`: use a table, columns, grouped labels, or annotations. Do not use one long bullet box.
- Do not create a deck where every content page is title plus bullets. For 4 or more pages, use at least 4 different layout structures when the content allows.
- Keep generous margins. Use `60-80` px outer margins on standard content pages unless a full-bleed image or cover treatment is intentional.
- Reserve vertical space for titles. A typical content title area is `y=36..90`; main content should usually start at `y>=110`.
- Avoid crowding the bottom edge. Keep non-background content above `y=500` unless it is a footer.
- Prefer fewer, larger objects over many small text boxes.
- Keep backgrounds consistent with the deck's `visual_system.background_strategy`. Normal content pages should use the same base background unless there is a clear page-role reason to change.
- Treat text fit as a layout constraint, not a cleanup step. If a text box is too small for the intended line count, shorten the text, split it, or allocate more space before creating XML.

## Background And Motif Consistency

Decks can vary page backgrounds, but variation must be intentional and legible:

- Pick one default background for ordinary content pages and reuse it exactly. Avoid near-identical drift such as several slightly different off-white values unless it encodes a clear section change.
- Cover, section divider, emphasis, and conclusion pages may use a dark, image-led, or high-contrast background. They must still share the deck's primary color, motif, edge treatment, typography, or geometry.
- If a cover uses a split composition, make the split visible in the background or layout. For example, reserve a darker text region and a related but distinct visual region instead of placing all elements on one flat field.
- Reuse a small number of visual devices: side bar, card radius, node style, line weight, icon container, or footer treatment. Do not introduce a new decorative language on each page.
- Insert background and motif shapes before content elements so they do not cover text, images, or diagrams.

## Text Fit Guardrails

Use these as conservative minimums on a 960 x 540 canvas. Increase height when using bold text, Chinese text, mixed Chinese/English, or line spacing above default.

| Text use | Typical font size | Minimum height |
|----------|-------------------|----------------|
| Caption, 1 line | 10-12 | 18 |
| Caption, 2 lines | 10-12 | 30 |
| Body, 1 line | 13-16 | 24 |
| Body, 2 lines | 13-16 | 40 |
| Body, 2 lines, bold | 15-18 | 48 |
| Headline, 1 line | 24-32 | 42 |
| Title, 2 lines | 34-44 | 110 |

Additional rules:

- Do not put long Chinese sentences or long English phrases into `height=18` or `height=22` boxes. Those heights are for short labels only.
- Footer/source text should usually be one short line. If it needs more, make it a real caption block above the footer area.
- Bottom conclusion bars should be at least `40` px tall for one emphasized line and at least `54` px tall for two lines.
- Diagram labels should be short enough to fit the shape. Prefer two short lines over one cramped long line.
- When a text block has more than one `<p>`, size the box for multiple lines explicitly. Do not assume the renderer will auto-expand.
- If a line contains mixed Chinese and English, budget more width than either language alone; mixed text wraps less predictably.

## Layout Types

### `title-cover`

Purpose: introduce the deck's point of view.

Geometry:
- Use one dominant title block, usually `x=70..120`, `y=150..250`, `width=700..820`.
- Add one subtitle or context line, not a bullet list.
- Optional visual focus can be a full-bleed background, large side image, accent band, or abstract shape motif.
- If the cover has a right-side diagram, screenshot, or motif cluster, use a split layout: keep the title/subtitle region within the left or central text region, and reserve a separate visual region so labels and connectors do not cross the title.
- For split covers, make the background reinforce the composition, such as a darker text side and a related visual panel. Avoid one flat field where title and diagram compete for attention.
- Keep source metadata to one short line where possible. If it wraps, shorten author lists or move details to notes.
- The main title should be controlled, normally one or two lines. Do not let it occupy both the text region and the visual region.

Text:
- `low` only unless the user explicitly asks for detail.

### `section-divider`

Purpose: reset rhythm and mark a new chapter.

Geometry:
- Use a large section number, chapter label, or single centered claim.
- Keep the page sparse. A divider is not a content page.
- Visual focus can be one oversized number, a vertical accent bar, or a full-width band.

Text:
- Title plus one phrase. No bullets.

### `two-column`

Purpose: compare two related ideas or pair explanation with evidence.

Geometry:
- Split main region into two balanced columns, for example left `x=60,width=400`, right `x=500,width=400`.
- Each column needs its own heading or visual anchor.
- Do not place one full-width bullet box under a normal title; that is not a two-column layout.

Text:
- `medium`: 2-3 short items per column.
- `high`: use grouped rows or mini table structure inside columns.

### `image-left-text-right`

Purpose: let a visual establish context, with text explaining implication.

Geometry:
- Left visual region should occupy roughly `35-45%` of slide width, often full height or tall crop.
- Right text region starts around `x=420` and should have a strong headline plus short support.
- If no real image is available, create a shape-based placeholder visual that matches `asset_need`.
- For dense screenshots, paper figures, or product captures with small labels, allocate a larger visual region when possible: often `50-65%` of slide width or at least `320` px height.
- Place screenshots in a deliberate frame or panel, and leave enough margin so axes, captions, and edge labels are not cropped by the slide boundary.

Text:
- Keep right-side text short. Avoid more than 4 bullets.
- For screenshot explanation pages, prefer 2-3 interpretation cards or callouts instead of a paragraph block.

### `image-right-text-left`

Purpose: lead with a message, then reinforce it with a visual.

Geometry:
- Left text region starts around `x=60..90`, width `400..460`.
- Right visual region occupies roughly `35-45%` of slide width.
- Align the image or placeholder with the main text block, not only with the title.
- For dense screenshots, paper figures, or product captures with small labels, increase the visual region and reduce text. A readable image is more valuable than a fully populated text column.

Text:
- Use one main claim and 2-3 supporting points.
- Keep callouts parallel and short. If a callout needs more than two lines, split it into a separate note or a new slide.

### `big-number`

Purpose: make one metric or fact memorable.

Geometry:
- Reserve the largest object for the metric: font size often `64-110`, region at least `300 x 120`.
- Pair the number with one explanation and optional 2-3 small supporting labels.
- Do not bury the number in a bullet list or small card.

Text:
- `low` or `medium`. If detail is needed, add small annotations around the metric.
- Supporting labels must not compete with the number. Use compact labels, legends, or mini-cards rather than long explanatory bars.

### `timeline`

Purpose: show sequence, roadmap, history, or phases.

Geometry:
- Create a horizontal or vertical spine with 3-6 milestones.
- Each milestone should have a dot/card/date label connected by a line or arrow.
- Title is separate from the sequence. The sequence is the visual focus.

Text:
- Each milestone gets a short label and optional one-line explanation.
- Do not use paragraph-length milestone descriptions.

### `comparison`

Purpose: make a choice, before/after, old/new, or option tradeoff clear.

Geometry:
- Use two or three distinct panels, columns, or a table-like structure.
- Headings must be visually aligned so differences are easy to scan.
- Use color, border, icon, or label treatment to highlight the preferred option or key difference.

Text:
- Use parallel wording across columns.
- Avoid uneven long bullet lists that destroy comparability.

### `architecture-diagram`

Purpose: explain components, dependencies, or system flow.

Implementation: prefer `<whiteboard>` (see `lark-slides-whiteboard.md`); use `<shape>` + `<line>` only as fallback.

Geometry:
- Main visual area should be a diagram, not prose.
- Use grouped boxes, lanes, arrows or lines, and short labels.
- Keep diagram labels concise. Put explanation in notes or a small side caption if needed.

Text:
- Prefer labels of 1-5 words.
- Use no more than one short explanatory text block.
- If a node label needs two lines, size the node and the text box for two lines. Do not let labels overlap connectors.

### `process-flow`

Purpose: show operational steps, workflow, or cause-effect path.

Implementation: prefer `<whiteboard>` (see `lark-slides-whiteboard.md`); use `<shape>` + `<line>` only as fallback.

Geometry:
- Use numbered steps connected by arrows or lines.
- 3-5 steps is ideal for one slide. If there are more, group them into phases.
- The flow direction must be visually obvious.

Text:
- Each step gets a verb-led label and one short descriptor at most.
- Step labels should be parallel in length and grammar. If one step needs a long explanation, move the explanation to a side note or speaker notes.

### `quote-highlight`

Purpose: emphasize a customer voice, principle, thesis, or decision statement.

Geometry:
- Quote or claim is the dominant text object.
- Use large type, generous whitespace, and optional attribution or context badge.
- Do not combine a quote-highlight page with a normal bullet section.

Text:
- One quote or statement, plus optional attribution. No bullets.

### `conclusion`

Purpose: close with decision, recommendation, or next action.

Geometry:
- Use one dominant closing statement or call to action.
- Add up to 3 next-step cards, checklist items, or owner/date labels.
- Visual focus should be the recommendation or action, not decorative filler.

Text:
- Keep the final page easy to remember. Avoid recap overload.
- Conclusion pages may mirror the cover background, but must clearly reuse the deck's motif or color roles so the ending feels intentional.

## Screenshot And Paper Figure Pages

When a page uses a real screenshot, chart, paper figure, or product capture:

- Choose screenshot placement based on page role, not a fixed slide number. Method overview, evidence, comparison, and failure-analysis pages are common candidates; title, agenda, and conclusion pages usually are not.
- Use the real asset only when it is readable at slide size. If the figure is too dense, crop to the relevant region, create a zoomed detail, or redraw the core message with native shapes.
- A screenshot should normally be the visual focus. Do not shrink it into a decorative thumbnail while surrounding it with dense text.
- Pair the image with a small number of interpretive annotations that tell the audience what to notice.
- Always include a short source caption when using external or paper-derived visuals.
- Verify the final XML contains a supported image token or creation-time local placeholder, not an unsupported external URL.

## Plan To XML Checklist

Before creating XML for each page, answer these checks:

1. Which region is the visual focus, and is it the largest or most prominent object?
2. Does the XML geometry match the `layout_type` description above?
3. Does `text_density` limit the number of paragraphs, bullets, labels, and text boxes?
4. Would this page still be recognizable if the `layout_type` label were removed from the plan?
5. Across the deck, do multiple pages use genuinely different structures?
6. Does the background follow the planned deck strategy, and are any deviations intentional?
7. Are all text boxes large enough for their intended font size and line count?
8. If the page uses a screenshot or paper figure, is it large enough to read and accompanied by concise interpretation?

After fetching the created presentation, verify:

- Use `timeline`, `comparison`, and `architecture-diagram` only when the content calls for them; do not force irrelevant page types.
- Any planned `timeline`, `comparison`, or `architecture-diagram` page uses the matching sequence, side-by-side comparison, or component-and-connection structure.
- Pages are not crowded and do not rely on long bullet boxes.
- Main claim, supporting detail, and visual focus have clear hierarchy.
- Static XML inspection should include text-fit risk: very short text boxes containing long text, multi-paragraph boxes with insufficient height, footer text that may wrap, and labels placed directly over connectors.
- Background and motif consistency should be checked across pages, not only within one slide.
