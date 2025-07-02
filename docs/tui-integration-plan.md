# TUI Integration Plan for CodeForge

## Overview
This document outlines the plan to integrate OpenCode's TUI approach into CodeForge while preserving existing functionality.

## Architecture Design

### 1. Module Structure
```
internal/tui/
├── app.go                  # Main TUI application model
├── components/
│   ├── chat/
│   │   ├── chat.go         # Main chat view
│   │   ├── editor.go       # Message input editor
│   │   ├── messages.go     # Message list display
│   │   └── sidebar.go      # Session/file sidebar
│   ├── dialogs/
│   │   ├── model.go        # Enhanced model selector (adapt existing)
│   │   ├── file.go         # File picker dialog
│   │   ├── session.go      # Session management dialog
│   │   └── help.go         # Help/keybindings dialog
│   └── common/
│       ├── status.go       # Status bar component
│       └── spinner.go      # Loading indicators
├── layout/
│   ├── split.go            # Split pane layouts
│   ├── overlay.go          # Dialog overlay system
│   └── container.go        # Container with borders/padding
├── themes/
│   ├── theme.go            # Theme interface
│   └── default.go          # Default CodeForge theme
└── styles/
    ├── colors.go           # Color definitions
    └── markdown.go         # Markdown rendering styles
```

### 2. Integration Points

#### A. Preserve Existing Components
- **Model Discovery**: Keep `internal/llm/models/discovery.go`
- **Provider Management**: Maintain all provider SDKs
- **Agent System**: Keep `internal/app/agent.go` integration
- **Command Routing**: Preserve natural language command handling

#### B. Enhance Current Components
- **Model Selector**: Enhance existing BubbleTea selector with OpenCode features
- **Chat Interface**: Replace readline with full TUI while keeping logic
- **Session Management**: Visual session switching with existing storage

#### C. New Components
- **File Browser**: Interactive file selection and context
- **Split Views**: Code viewer alongside chat
- **Status Bar**: Show current model, session, tokens used
- **Theme System**: User-customizable themes

### 3. Implementation Phases

#### Phase 1: Base TUI Structure (High Priority)
1. Create `internal/tui/app.go` with main application model
2. Set up basic layout system (containers, splits)
3. Integrate with existing `cmd/codeforge/main.go`
4. Add command-line flag for TUI mode: `codeforge --tui`

#### Phase 2: Core Chat Interface
1. Implement message display component
2. Create input editor with multi-line support
3. Connect to existing chat logic in `internal/chat/`
4. Add streaming response support

#### Phase 3: Enhanced Model Selection
1. Adapt existing model selector to match OpenCode style
2. Add provider grouping and navigation
3. Implement favorites persistence
4. Add visual indicators for availability

#### Phase 4: Session Management
1. Create session list sidebar
2. Implement session switching
3. Add session creation/deletion dialogs
4. Persist session state

#### Phase 5: Advanced Features
1. File picker for context
2. Markdown rendering with syntax highlighting
3. Theme switching
4. Export/import conversations

## Technical Considerations

### 1. State Management
- Central app model following BubbleTea patterns
- Immutable updates with message passing
- Clean separation between UI and business logic

### 2. Performance
- Lazy loading for large message histories
- Efficient rendering with viewport components
- Background model discovery

### 3. Compatibility
- TUI mode as opt-in feature (`--tui` flag)
- Existing CLI mode remains default
- API server continues to work independently

### 4. Error Handling
- Graceful degradation for terminal limitations
- Clear error messages in UI
- Fallback to CLI mode if TUI fails

## Migration Strategy

1. **Parallel Development**: TUI developed alongside existing interface
2. **Feature Parity**: Ensure all CLI features work in TUI
3. **User Testing**: Beta flag for early adopters
4. **Documentation**: Update README and add TUI guide
5. **Gradual Rollout**: Make TUI default after stability

## Success Criteria

- ✅ Full feature parity with current CLI interface
- ✅ Smooth, responsive UI with <100ms interactions
- ✅ Proper error handling and recovery
- ✅ Works on common terminals (iTerm, Terminal.app, WSL)
- ✅ Maintains all existing integrations
- ✅ Clean, maintainable code following Go best practices