package ui

import (
	"github.com/asheshgoplani/agent-deck/internal/session"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// ClaudeOptionsPanel is a UI panel for Claude-specific launch options
// Used in both ForkDialog and NewDialog
type ClaudeOptionsPanel struct {
	// Session mode: 0=new, 1=continue, 2=resume
	sessionMode int
	// Resume session ID input (only for mode=resume)
	resumeIDInput textinput.Model
	// Checkbox states
	skipPermissions      bool
	allowSkipPermissions bool
	autoMode             bool
	useChrome            bool
	useTeammateMode      bool
	// Account picker state (only rendered when len(accounts) >= 2)
	accounts      []string // sorted names from config, plus leading "" for "(default)"
	accountIndex  int      // index into accounts
	// Focus tracking
	focusIndex int
	// Whether this panel is for fork dialog (fewer options)
	isForkMode bool
	// Total number of focusable elements
	focusCount int
}

// Focus indices for NewDialog mode:
// 0: Session mode (radio)
// 1: Resume ID input (only when mode=resume)
// 2: Skip permissions checkbox
// 3: Chrome checkbox

// Focus indices for ForkDialog mode:
// 0: Skip permissions checkbox
// 1: Chrome checkbox

// NewClaudeOptionsPanel creates a new panel for NewDialog
func NewClaudeOptionsPanel() *ClaudeOptionsPanel {
	resumeInput := textinput.New()
	resumeInput.Placeholder = "session_id..."
	resumeInput.CharLimit = 64
	resumeInput.Width = 30

	return &ClaudeOptionsPanel{
		sessionMode:   0, // new
		resumeIDInput: resumeInput,
		isForkMode:    false,
		focusCount:    5, // Will adjust dynamically
	}
}

// NewClaudeOptionsPanelForFork creates a panel for ForkDialog (fewer options)
func NewClaudeOptionsPanelForFork() *ClaudeOptionsPanel {
	return &ClaudeOptionsPanel{
		sessionMode:   0,
		resumeIDInput: textinput.New(), // Not used in fork mode
		isForkMode:    true,
		focusCount:    3, // skip, chrome, teammate
	}
}

// SetDefaults applies default values from config
func (p *ClaudeOptionsPanel) SetDefaults(config *session.UserConfig) {
	if config != nil {
		p.skipPermissions = config.Claude.GetDangerousMode()
		p.allowSkipPermissions = config.Claude.AllowDangerousMode
		p.autoMode = config.Claude.AutoMode
		p.loadAccounts(config)
	}
}

// loadAccounts populates the account picker options from config.
// First slot is always "" (= "use default_account / no override"). The picker
// is only rendered when >=2 configured accounts exist (accountsVisible()).
func (p *ClaudeOptionsPanel) loadAccounts(config *session.UserConfig) {
	names := config.ListClaudeAccounts()
	p.accounts = append([]string{""}, names...)
	// Preselect the configured default_account if present.
	p.accountIndex = 0
	for i, n := range p.accounts {
		if n == config.Claude.DefaultAccount {
			p.accountIndex = i
			break
		}
	}
}

// accountsVisible returns true when the account picker row should be rendered.
// Threshold: at least two real accounts configured (picker earns its pixels).
func (p *ClaudeOptionsPanel) accountsVisible() bool {
	return len(p.accounts) >= 3 // leading "" + >=2 real accounts
}

// SetFromOptions applies persisted ClaudeOptions to the panel fields.
func (p *ClaudeOptionsPanel) SetFromOptions(opts *session.ClaudeOptions) {
	if opts == nil {
		return
	}
	switch opts.SessionMode {
	case "continue":
		p.sessionMode = 1
	case "resume":
		p.sessionMode = 2
		p.resumeIDInput.SetValue(opts.ResumeSessionID)
	default:
		p.sessionMode = 0
	}
	p.skipPermissions = opts.SkipPermissions
	p.allowSkipPermissions = opts.AllowSkipPermissions
	p.autoMode = opts.AutoMode
	p.useChrome = opts.UseChrome
	p.useTeammateMode = opts.UseTeammateMode
	// Select the configured account if it's in the loaded set.
	for i, n := range p.accounts {
		if n == opts.Account {
			p.accountIndex = i
			break
		}
	}
	p.updateInputFocus()
	p.focusCount = p.getFocusCount()
}

// Focus sets focus to this panel
func (p *ClaudeOptionsPanel) Focus() {
	p.focusIndex = 0
	p.updateInputFocus()
}

// Blur removes focus from this panel
func (p *ClaudeOptionsPanel) Blur() {
	p.focusIndex = -1
	p.resumeIDInput.Blur()
}

// IsFocused returns true if any element in the panel has focus
func (p *ClaudeOptionsPanel) IsFocused() bool {
	return p.focusIndex >= 0
}

// AtTop returns true if focus is on the first element
func (p *ClaudeOptionsPanel) AtTop() bool {
	return p.focusIndex <= 0
}

// GetOptions returns current options as ClaudeOptions
func (p *ClaudeOptionsPanel) GetOptions() *session.ClaudeOptions {
	var account string
	if p.accountIndex >= 0 && p.accountIndex < len(p.accounts) {
		account = p.accounts[p.accountIndex]
	}
	opts := &session.ClaudeOptions{
		SkipPermissions:      p.skipPermissions,
		AllowSkipPermissions: p.allowSkipPermissions,
		AutoMode:             p.autoMode,
		UseChrome:            p.useChrome,
		UseTeammateMode:      p.useTeammateMode,
		Account:              account,
	}

	if !p.isForkMode {
		switch p.sessionMode {
		case 0:
			opts.SessionMode = "new"
		case 1:
			opts.SessionMode = "continue"
		case 2:
			opts.SessionMode = "resume"
			opts.ResumeSessionID = p.resumeIDInput.Value()
		}
	}

	return opts
}

// Update handles key events
func (p *ClaudeOptionsPanel) Update(msg tea.Msg) tea.Cmd {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch msg.String() {
		case "up":
			p.focusIndex--
			if p.focusIndex < 0 {
				p.focusIndex = p.getFocusCount() - 1
			}
			p.updateInputFocus()
			return nil

		case "down", "tab":
			p.focusIndex++
			if p.focusIndex >= p.getFocusCount() {
				p.focusIndex = 0
			}
			p.updateInputFocus()
			return nil

		case "shift+tab":
			p.focusIndex--
			if p.focusIndex < 0 {
				p.focusIndex = p.getFocusCount() - 1
			}
			p.updateInputFocus()
			return nil

		case " ":
			// Don't intercept space when focused on a text input
			if p.isResumeInputFocused() {
				break // Let it fall through to text input handling
			}
			// Toggle checkbox or radio at current focus
			p.handleSpaceKey()
			return nil

		case "left", "right":
			// For session mode radio buttons
			if !p.isForkMode && p.focusIndex == 0 {
				if msg.String() == "left" {
					p.sessionMode--
					if p.sessionMode < 0 {
						p.sessionMode = 2
					}
				} else {
					p.sessionMode = (p.sessionMode + 1) % 3
				}
				return nil
			}
			// Account picker: left/right cycles when focused
			if p.getFocusType() == "account" && p.accountsVisible() {
				if msg.String() == "left" {
					p.accountIndex--
					if p.accountIndex < 0 {
						p.accountIndex = len(p.accounts) - 1
					}
				} else {
					p.accountIndex = (p.accountIndex + 1) % len(p.accounts)
				}
				return nil
			}
		}
	}

	// Update text inputs if focused
	if p.isResumeInputFocused() {
		var cmd tea.Cmd
		p.resumeIDInput, cmd = p.resumeIDInput.Update(msg)
		return cmd
	}

	return nil
}

// handleSpaceKey handles space key for toggling checkboxes/radios
func (p *ClaudeOptionsPanel) handleSpaceKey() {
	switch p.getFocusType() {
	case "sessionMode":
		p.sessionMode = (p.sessionMode + 1) % 3
	case "skipPermissions":
		p.skipPermissions = !p.skipPermissions
	case "autoMode":
		p.autoMode = !p.autoMode
	case "chrome":
		p.useChrome = !p.useChrome
	case "teammateMode":
		p.useTeammateMode = !p.useTeammateMode
	case "account":
		if len(p.accounts) > 0 {
			p.accountIndex = (p.accountIndex + 1) % len(p.accounts)
		}
	}
}

// getFocusType returns what type of element is currently focused
func (p *ClaudeOptionsPanel) getFocusType() string {
	if p.isForkMode {
		switch p.focusIndex {
		case 0:
			return "skipPermissions"
		case 1:
			return "autoMode"
		case 2:
			return "chrome"
		case 3:
			return "teammateMode"
		case 4:
			if p.accountsVisible() {
				return "account"
			}
		}
		return ""
	}
	idx := p.focusIndex
	// 0: session mode
	if idx == 0 {
		return "sessionMode"
	}
	// 1: resume input (only if mode == resume)
	if p.sessionMode == 2 {
		if idx == 1 {
			return "resumeInput"
		}
		idx-- // Adjust for missing resume input
	}
	switch idx {
	case 1:
		return "skipPermissions"
	case 2:
		return "autoMode"
	case 3:
		return "chrome"
	case 4:
		return "teammateMode"
	case 5:
		if p.accountsVisible() {
			return "account"
		}
	}
	return ""
}

// getFocusCount returns the number of focusable elements
func (p *ClaudeOptionsPanel) getFocusCount() int {
	if p.isForkMode {
		count := 4 // skip, auto, chrome, teammate
		if p.accountsVisible() {
			count++
		}
		return count
	}

	count := 5 // session mode, skip, auto, chrome, teammate
	if p.sessionMode == 2 {
		count++ // resume input
	}
	if p.accountsVisible() {
		count++
	}
	return count
}

// isResumeInputFocused returns true if resume input is focused
func (p *ClaudeOptionsPanel) isResumeInputFocused() bool {
	return !p.isForkMode && p.sessionMode == 2 && p.focusIndex == 1
}

// updateInputFocus updates which text input has focus
func (p *ClaudeOptionsPanel) updateInputFocus() {
	p.resumeIDInput.Blur()

	if p.isResumeInputFocused() {
		p.resumeIDInput.Focus()
	}
}

// View renders the options panel
func (p *ClaudeOptionsPanel) View() string {
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	dimStyle := lipgloss.NewStyle().Foreground(ColorComment)
	headerStyle := lipgloss.NewStyle().Foreground(ColorComment)

	var content string

	if p.isForkMode {
		content = p.viewForkMode(labelStyle, activeStyle, dimStyle, headerStyle)
	} else {
		content = p.viewNewMode(labelStyle, activeStyle, dimStyle, headerStyle)
	}

	return content
}

// viewForkMode renders options for ForkDialog
func (p *ClaudeOptionsPanel) viewForkMode(labelStyle, activeStyle, dimStyle, headerStyle lipgloss.Style) string {
	var content string
	content += headerStyle.Render("─ Advanced Options ─") + "\n"
	content += renderCheckboxLine("Skip permissions", p.skipPermissions, p.focusIndex == 0)
	content += renderCheckboxLine("Auto mode", p.autoMode, p.focusIndex == 1)
	if p.autoMode && p.skipPermissions {
		content += dimStyle.Render("    ↑ overridden by skip permissions") + "\n"
	}
	content += renderCheckboxLine("Chrome mode", p.useChrome, p.focusIndex == 2)
	content += renderCheckboxLine("Teammate mode", p.useTeammateMode, p.focusIndex == 3)
	if p.accountsVisible() {
		content += p.renderAccountPicker(p.focusIndex == 4)
	}
	return content
}

// viewNewMode renders options for NewDialog
func (p *ClaudeOptionsPanel) viewNewMode(labelStyle, activeStyle, dimStyle, headerStyle lipgloss.Style) string {
	var content string
	content += headerStyle.Render("─ Claude Options ─") + "\n"

	// Session mode radio buttons
	focusIdx := 0
	radioLabel := "  Session: "
	if p.focusIndex == focusIdx {
		radioLabel = activeStyle.Render("▶ Session: ")
	}
	content += radioLabel
	content += p.renderRadio("New", p.sessionMode == 0, p.focusIndex == focusIdx) + "  "
	content += p.renderRadio("Continue", p.sessionMode == 1, p.focusIndex == focusIdx) + "  "
	content += p.renderRadio("Resume", p.sessionMode == 2, p.focusIndex == focusIdx) + "\n"
	focusIdx++

	// Resume ID input (only if resume mode)
	if p.sessionMode == 2 {
		if p.focusIndex == focusIdx {
			content += activeStyle.Render("    ▶ ID: ") + p.resumeIDInput.View() + "\n"
		} else {
			content += "      ID: " + p.resumeIDInput.View() + "\n"
		}
		focusIdx++
	}

	// Skip permissions checkbox
	content += renderCheckboxLine("Skip permissions", p.skipPermissions, p.focusIndex == focusIdx)
	focusIdx++

	// Auto mode checkbox
	content += renderCheckboxLine("Auto mode", p.autoMode, p.focusIndex == focusIdx)
	if p.autoMode && p.skipPermissions {
		content += dimStyle.Render("    ↑ overridden by skip permissions") + "\n"
	}
	focusIdx++

	// Chrome checkbox
	content += renderCheckboxLine("Chrome mode", p.useChrome, p.focusIndex == focusIdx)
	focusIdx++

	// Teammate mode checkbox
	content += renderCheckboxLine("Teammate mode", p.useTeammateMode, p.focusIndex == focusIdx)
	focusIdx++

	// Account picker (only when >=2 accounts configured)
	if p.accountsVisible() {
		content += p.renderAccountPicker(p.focusIndex == focusIdx)
	}

	return content
}

// renderAccountPicker renders the Claude account selector row.
// Format mirrors the session-mode radio ("◀ name ▶") so left/right feels
// consistent with the rest of the dialog. Space also cycles forward.
func (p *ClaudeOptionsPanel) renderAccountPicker(focused bool) string {
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)
	valueStyle := lipgloss.NewStyle().Foreground(ColorCyan)

	name := ""
	if p.accountIndex >= 0 && p.accountIndex < len(p.accounts) {
		name = p.accounts[p.accountIndex]
	}
	display := name
	if display == "" {
		display = "(default)"
	}

	prefix := "  "
	label := "Account: "
	if focused {
		prefix = activeStyle.Render("▶ ")
		label = activeStyle.Render("Account: ")
	} else {
		label = labelStyle.Render(label)
	}
	return prefix + label + valueStyle.Render("◀ "+display+" ▶") + "\n"
}

// renderCheckboxMark renders a checkbox mark [x] or [ ] with consistent styling.
// Shared across all tool option panels for visual consistency.
func renderCheckboxMark(checked, focused bool) string {
	style := lipgloss.NewStyle()
	if focused {
		style = style.Foreground(ColorAccent).Bold(true)
	}
	if checked {
		return style.Render("[x]")
	}
	return style.Render("[ ]")
}

// renderCheckboxLine renders a complete checkbox line with label, matching Claude options panel style.
// Used by Gemini and Codex options in NewDialog for visual consistency with Claude.
func renderCheckboxLine(label string, checked, focused bool) string {
	activeStyle := lipgloss.NewStyle().Foreground(ColorAccent).Bold(true)
	labelStyle := lipgloss.NewStyle().Foreground(ColorText)

	cb := renderCheckboxMark(checked, focused)
	if focused {
		return activeStyle.Render("▶ ") + cb + " " + label + "\n"
	}
	return "  " + cb + " " + labelStyle.Render(label) + "\n"
}

// renderRadio renders a radio button (•) or ( )
func (p *ClaudeOptionsPanel) renderRadio(label string, selected, focused bool) string {
	style := lipgloss.NewStyle()
	if focused && selected {
		style = style.Foreground(ColorAccent).Bold(true)
	} else if selected {
		style = style.Foreground(ColorCyan)
	} else {
		style = style.Foreground(ColorComment)
	}

	if selected {
		return style.Render("(•) " + label)
	}
	return style.Render("( ) " + label)
}
