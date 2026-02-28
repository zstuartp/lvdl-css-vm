// Copyright 2026 Zackary Parsons. Licensed under Apache-2.0.

package emit

import "io"

// themeDarkCSS returns CSS blocks that activate dark-mode variables
// when #theme-dark is checked, or when the system prefers dark and
// #theme-light is not checked.
func themeDarkCSS(darkVars string) string {
	return ":root:has(#theme-dark:checked){\n" + darkVars + "\n}\n" +
		"@media(prefers-color-scheme:dark){\n  :root:not(:has(#theme-light:checked)){\n" + darkVars + "\n  }\n}\n"
}

// themeRadios writes three hidden radio inputs for the 3-way theme
// toggle (Auto/Dark/Light). Place outside any <form> so that Reset
// does not clobber the user's theme choice.
func themeRadios(w io.Writer) {
	io.WriteString(w, "<input type=\"radio\" name=\"theme\" id=\"theme-auto\" checked>\n")
	io.WriteString(w, "<input type=\"radio\" name=\"theme\" id=\"theme-dark\">\n")
	io.WriteString(w, "<input type=\"radio\" name=\"theme\" id=\"theme-light\">\n")
}

// themeToggleHTML writes the visible pill-shaped toggle labels.
func themeToggleHTML(w io.Writer) {
	io.WriteString(w, "<div class=\"theme-toggle\">")
	io.WriteString(w, "<label for=\"theme-auto\" class=\"tt auto\">Auto</label>")
	io.WriteString(w, "<label for=\"theme-dark\" class=\"tt dark\">Dark</label>")
	io.WriteString(w, "<label for=\"theme-light\" class=\"tt light\">Light</label>")
	io.WriteString(w, "</div>\n")
}

// themeToggleCSS returns CSS for the .theme-toggle pill and active
// state highlighting. Includes hiding rules for the radio inputs.
func themeToggleCSS() string {
	return `
#theme-auto,#theme-dark,#theme-light{position:absolute;opacity:0;pointer-events:none;width:0;height:0;overflow:hidden}
.theme-toggle{display:inline-flex;border:1px solid var(--border);border-radius:999px;overflow:hidden}
.tt{padding:4px 12px;cursor:pointer;font-size:11px;font-weight:700;color:var(--text-muted);user-select:none}
.tt:not(:first-child){border-left:1px solid var(--border)}
.tt:hover{background:var(--bg-active)}
:root:has(#theme-auto:checked) .tt.auto,
:root:has(#theme-dark:checked) .tt.dark,
:root:has(#theme-light:checked) .tt.light{color:var(--text);background:var(--bg-active)}
`
}
