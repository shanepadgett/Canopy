---
{
  "title": "Shortcodes",
  "description": "Examples of built-in shortcodes",
  "weight": 2
}
---

This guide demonstrates the built-in shortcodes.

{{< toc >}}

## Callouts

{{< callout type="warning" title="Be careful" >}}
Shortcodes can include **Markdown** content and inline elements like a video: {{< youtube id="dQw4w9WgXcQ" title="Demo video" >}}.
{{< /callout >}}

## Figure

{{< figure src="https://placehold.co/640x360" alt="Placeholder image" caption="A placeholder image rendered via shortcode." >}}

## Key takeaways

{{< key-takeaways >}}
- Shortcodes map to templates.
- `{{< >}}` renders inner Markdown.
- `{{% %}}` keeps inner content raw.
{{< /key-takeaways >}}

## Prerequisites

{{< prereqs >}}
- Basic Markdown knowledge
- A Canopy site
{{< /prereqs >}}

## Code tabs (raw)

{{% code-tabs %}}
<div class="tab-panel">
  <pre><code class="language-js">console.log("raw content");</code></pre>
  *Not Markdown*
</div>
{{% /code-tabs %}}
