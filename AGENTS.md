# fluid

## RULE 0 — Scope discipline (read this first)

**Do exactly what was asked. Nothing more.**

- Finish **only** the requested task. Do not "improve" adjacent code, rename things, reformat unrelated files, add tests, or fix unrelated bugs you notice along the way.
- **A question is not an instruction to act.** "which is better?", "how does X work?", "does Y do Z?", "what changed?", "wtf is this?" — all of these mean *answer only*. Do not edit files, create files, or run mutating commands. Reply, then wait for a separate instruction.
- If you notice something else that is wrong, **report it — do not fix it.** Mention it and wait.
- **The moment you hit a blocker, an ambiguity, a design decision, or anything out of scope: STOP and ask a question.** Do not guess. Do not pick "the reasonable default" and keep going.
- **Never declare the task finished without explicit permission.** When the requested change is done, stop, state what was done, and ask before doing anything further.
- If the task turns out to be bigger than described, stop and say so before writing code.

## Project Structure

```
fluid/
  buffer/           cell buffer (2D grid of styled characters), border drawing
  input/            keyboard, mouse, paste events + keybindings + parser
  layout/           flexbox layout engine
  render/           scene graph, base renderable, selection, render context
  style/            color, style (fg/bg/attrs), StyledText, border types & chars
  symbol/           unicode character constants (borders, bars, blocks, scrollbars)
  terminal/         terminal renderer, diff, input loop, hit-testing, backend
  testutil/         test helpers (golden output, mock renderables)
  textbuf/          text buffer, edit buffer, text/edit surface, word wrap, unicode width
  widget/           box, text, textarea, scrollbox, markdown, diff, code, table, tree, etc.
```

## Build & Test

```sh
go build ./...
go test ./...                  # all tests
go test ./widget/ -v           # widget tests
go test ./render/ -run Inline  # inline render tests
go test ./layout/              # layout engine tests
```

## Key Architecture Decisions

### Rendering

fluid is a retained-mode terminal UI framework with a flexbox layout engine:

- **Scene graph**: Widgets implement `render.Renderable`. The tree is rooted at the renderer's root renderable, reached via `Renderer.Root()`.
- **Layout**: CSS-like flexbox (`layout.LayoutNode`) with `FlexGrow`, `FlexShrink`, `AlignItems`, `JustifyContent`, etc. Leaf widgets provide a `MeasureFunc` for intrinsic sizing.
- **Rendering**: Double-buffered cell grid (`buffer.Buffer`). Widgets draw into the buffer via `RenderSelf`. The renderer diffs and flushes to the terminal.
- **Inline mode**: Renders into the terminal's normal scrollback (no alternate screen). Uses `computeContentHeight()` to emit only the rows needed. Trailing clear cells are trimmed per-row.
- **Fullscreen mode**: Uses the alternate screen buffer with a fixed viewport.

### Text Widget Content Models

The `widget.Text` widget has two content paths:

1. **Manual content** (`SetContent(string)` or `SetContent(style.StyledText)`): Sets `t.content` directly. Used for simple or pre-styled text.
2. **TextNode tree** (`AddNode`, `AddText`): A tree of `TextNode` elements with inherited styles. Flattened into `t.content` via `ensureWrapped()` → `rebuildFromTextNodes()`.

**Important**: `ensureWrapped()` is the sync bridge between the TextNode tree and the content model. It must be called before any read of `t.content` (measurement, rendering, line counting). The `MeasureFunc` calls it to ensure layout sees current content.

### Inline Mode Considerations

When using inline mode (`InlineMode: true` in `terminal.Config`):

- Do **not** use `FlexGrow(1)` on content containers — it expands to fill the 10000-row layout ceiling. Use auto-sized `Box` instead of `ScrollBox`.
- Widget backgrounds require non-transparent `BackgroundColor` to be visible, since `isInlineClearCell` trims cells with transparent bg.
- Input areas should not have fixed heights — let them auto-size so content can expand. The `Textarea` widget has a `MeasureFunc` that reports `LineCount` as intrinsic height.

## Remembered rules

- Do not write ugly Go. Write idiomatic, clean Go: small focused functions, clear names, early returns instead of deep nesting, no redundant comments, no copy-paste blocks, proper error wrapping. If it reads badly, refactor it before shipping.
- Bring Rust/TypeScript type discipline to Go: define named types instead of passing bare `string`/`int`, make illegal states unrepresentable, never use `any`/`interface{}` where a concrete type or generic works, handle every error explicitly (no silent `_`), prefer immutable values and constructor functions that return a valid object or an error.
- Do **not** import Rust/TS *syntax* into Go: no `Result[T, E]` / `Option[T]` wrapper types, no monadic chaining, no class-style inheritance hierarchies, no `Promise`-like abstractions over goroutines. Use plain `(T, error)`, channels, and interfaces.

### Complexity budget

Hard limits — if a change breaks one, split it before committing:

- **Function length**: ≤ 60 lines / ≤ 40 statements. Beyond that, extract helpers.
- **Cyclomatic complexity**: ≤ 10 per function (absolute ceiling 15).
- **Nesting depth**: ≤ 3 levels inside a function. Use early returns and guard clauses instead of `else` ladders.
- **Parameters**: ≤ 4. Beyond that, take an options struct. No boolean flag parameters — split into two functions or use a named type.
- **Return values**: ≤ 3 (including `error`). More means the function is doing too much.
- **File size**: split above ~600 lines.
- **Package deps**: no import cycles, no package importing a sibling just for one helper — duplicate the 5 lines or lift it to a shared package.

### Go 1.26+ style — IMPORTANT

This module targets **`go 1.26`** (`go.mod:3`) and the toolchain is Go 1.26.0. **Write modern Go. Do not write Go that looks like it was written in 2018.**

Required:

- **Generics over `interface{}`.** Type parameters, generic type aliases (1.24). `any`, never `interface{}`.
- **Iterators (1.23).** Return `iter.Seq[T]` / `iter.Seq2[K, V]` for sequences; range over functions. Do not hand back `[]T` copies or take visitor callbacks where an iterator fits.
- **`slices` / `maps` (1.21+).** `slices.Sort`, `slices.Contains`, `slices.Clone`, `maps.Keys`, etc. Never hand-roll these loops.
- **Builtins**: `min`, `max`, `clear` (1.21). No manual ternary-style helpers.
- **`for i := range n` (1.22)** instead of `for i := 0; i < n; i++`.
- **`errors.AsType[T](err)` (1.26)** instead of declaring a var and calling `errors.As`. Use `errors.Join` for multi-error, `%w` for wrapping.
- **`sync.WaitGroup.Go(fn)` (1.25)** instead of `wg.Add(1)` + `go func(){ defer wg.Done() }()`.
- **`testing/synctest`** for concurrency tests, `b.Loop()` (1.24) for benchmarks, `t.ArtifactDir()` (1.26) for test output files.
- **`math/rand/v2`** (never `math/rand`), `log/slog` (never `log` for structured output), `os.Root` (1.24) for sandboxed FS access.
- Loop variables are per-iteration since 1.22 — **do not** write `x := x` shadow copies.

Do **not** use `encoding/json/v2` — it is still `GOEXPERIMENT`-gated in 1.26 and will not build.

### Style authority: the Go standard library only

**The only style references are official Go sources.** Match the standard library source in `$GOROOT/src`, Effective Go, Go Code Review Comments, and the Go Doc Comments spec (`go.dev/doc/comment`).

**Do not apply third-party style guides** — not Uber's Go guide, not Google's Go style guide, not any blog's "clean architecture in Go". If a convention cannot be found in the stdlib source, it is not a convention here.

When unsure how to name or shape something, **read the analogous stdlib package and copy its shape.** `buffer` should read like `image`; `textbuf` like `bytes`/`strings`; `style` like `image/color`.

Concrete, as practiced in `$GOROOT/src`:

- **Formatting**: `gofmt` clean, always. `go vet ./...` clean, always. Non-negotiable.
- **Doc comments** start with the identifier: `// Buffer holds ...`, `// Draw renders ...`. Package docs start `// Package buffer ...`. Complete sentences, ending in a period.
- **Error strings**: lowercase, no trailing punctuation — `errors.New("short write")`. (94% of stdlib `errors.New` calls follow this.)
- **Error naming**: sentinel vars are `ErrFoo`; error types are `FooError`.
- **`any`, not `interface{}`** — the stdlib is 9318 `any` vs 286 `interface{}`.
- **No `Get` prefix on getters**: `Name()`, not `GetName()`. Setters are `SetName()`.
- **Receiver names**: 1–2 letters, consistent across every method on the type (`b *Buffer`, `t *Text`). Never `self` or `this`.
- **No stutter**: `buffer.New`, not `buffer.NewBuffer`; `style.Color`, not `style.StyleColor`.
- **Single-method interfaces** take the `-er` name: `Renderer`, `Writer`, `Measurer`.
- **Acronyms keep case**: `URL`, `ID`, `HTTP`, `ANSI`, `UTF8` — `parseURL`, not `parseUrl`.
- **Variable name length scales with scope**: `i`, `b`, `n` in tight loops; descriptive names for package-level identifiers.
- **`ctx context.Context` is always the first parameter.** Never store a context in a struct.
- **Named result parameters** only for documentation value or when needed by `defer`. Never bare `return` in a long function.
- **No `util`, `common`, `helpers`, `base`, `misc` packages.** Name packages after what they provide.
- **No panics across package boundaries.** Panic only for programmer error that cannot be signalled otherwise; return `error` for everything else.
- **Table-driven tests** with `t.Run(name, ...)`, subtests named for the case. Standard `testing` only — no assertion frameworks.

**Precedence:** where the "Rust/TypeScript type discipline" rule above conflicts with stdlib style, **stdlib style wins** for naming, formatting, package layout, and API shape. Type discipline applies to domain modelling (named types, illegal states unrepresentable), not to renaming `b *Buffer` into `buffer *Buffer`.

### Export by default

**Declare structs and methods exported (capitalised) by default.** Write `type Buffer struct` and `func (b *Buffer) Render()`, not `type buffer struct` / `func (b *buffer) render()`.

- Only make something unexported when explicitly asked to, or when it is a genuinely local implementation detail that has been requested to stay private.
- This applies to struct types, struct fields, methods, and constructors.
- Naming still follows stdlib rules above: exported does not mean stuttering (`buffer.Buffer` is fine, `buffer.NewBuffer` is not) and every exported identifier needs a doc comment starting with its name.
