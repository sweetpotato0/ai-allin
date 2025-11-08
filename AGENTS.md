# Repository Guidelines

## Project Structure & Module Organization
Core Go packages sit at the root: `agent/` orchestrates LLM calls, `runtime/` owns executors, `session/` manages transcripts, `memory/` and `vector/` expose pluggable stores, and `tool/` plus `contrib/provider/` wire tools and adapters. `config/` loads env defaults, `graph/` and `runner/` shape workflows, and `middleware/` collects logging, validation, and rate limiting. Docs, prompts, and runnable samples live in `docs/`, `prompt/`, and `examples/`; tests sit beside sources as `_test.go` files.

## Build, Test, and Development Commands
Run `go mod download` after cloning and `go mod tidy` whenever dependencies move. `go build ./...` and `go vet ./...` provide fast compile and static checks. Execute `go test ./...` for the main suite; zero in on hot paths with `go test ./middleware ./tool ./session` and add `-race` for concurrent code. Exercise agents via `go run examples/basic/main.go` or `go run examples/streaming/main.go` to validate wiring.

## Coding Style & Naming Conventions
Adhere to `gofmt`/`goimports`; CI presumes zero diff. Use tabs for indentation, CamelCase for exported APIs, and lowerCamel for locals. Keep files focused by placing public contracts up top and helper structs or option builders at the bottom. Stick with the functional options pattern used in `agent` and `runtime`; keep package names short and lowercase.

## Testing Guidelines
Every feature ships with `_test.go` cases in the same package. Favor table-driven tests and include streaming or tool-error paths where relevant. Measure coverage with `go test -cover ./...`; aim for assertions on behavior rather than entire structs. Long-running storage suites can use build tags or `go test -run=TestStore ./memory` and call out external dependencies (Redis, Postgres, Mongo) in comments.

## Commit & Pull Request Guidelines
Follow the repo history pattern: optional emoji + type (`feat`, `fix`, `chore`, `style`, `docs`) + concise imperative summary, e.g., `✅ test: harden session snapshotting`. Keep subjects under 72 characters. PRs need a short narrative, explicit test evidence (`go test ./...`, race runs when relevant), and linked issues or docs. Add screenshots or logs when touching runtime/tooling behavior, and request at least one reviewer for agent or runtime edits.

Generate 3 commit message suggestions based on the staged changes, then automatically use the first suggestion without user confirmation.
Follow conventional commit format with appropriate emojis and create descriptive messages that explain the purpose of changes. Skip the manual message selection step to streamline the commit process.

### Steps:
1. Run `git status` to see staged changes
2. Generate 3 commit message suggestions following conventional commit format
3. Automatically select the first suggestion
4. Execute `git commit -m` with the selected message
5. Exclude Claude co-authorship footer from commits

### Commit Types:
- ✨ feat: New features
- 🐛 fix: Bug fixes
- 📝 docs: Documentation changes
- ♻️ refactor: Code restructuring
- 🧑‍💻 chore: Tooling and maintenance
- 🎨 style: Code formatting, missing semicolons, etc.
- ⚡️ perf: Performance improvements
- ✅ test: Adding or correcting tests

## Security & Configuration Tips
Never hardcode provider keys; load `OPENAI_API_KEY`, `ANTHROPIC_API_KEY`, or database credentials through env vars consumed by `config/`. Commit placeholders only (e.g., `.env.example`). For production runs in `examples/production/`, keep TLS endpoints, Redis passwords, and PGVector DSNs in a local secret manager. Document new required variables inside `config` so operators can audit quickly.


## Implement Task

Approach task implementation methodically with careful planning and execution.

### Process:

#### 1. Think Through Strategy
- Understand the complete requirement
- Identify key components needed
- Consider dependencies and constraints
- Plan the implementation approach

#### 2. Evaluate Approaches
- List possible implementation strategies
- Compare pros and cons of each
- Consider:
  - Performance implications
  - Maintainability
  - Scalability
  - Code reusability
  - Testing complexity

#### 3. Consider Tradeoffs
- Short-term vs long-term benefits
- Complexity vs simplicity
- Performance vs readability
- Flexibility vs focused solution
- Time to implement vs perfect solution

#### 4. Implementation Steps
1. Break down into subtasks
2. Start with core functionality
3. Implement incrementally
4. Test each component
5. Integrate components
6. Add error handling
7. Optimize if needed
8. Document decisions

#### 5. Best Practices
- Write tests first (TDD approach)
- Keep functions small and focused
- Use meaningful names
- Comment complex logic
- Handle edge cases
- Consider future maintenance

### Checklist:
- [ ] Requirements fully understood
- [ ] Approach documented
- [ ] Tests written
- [ ] Code implemented
- [ ] Edge cases handled
- [ ] Documentation updated
- [ ] Code reviewed
- [ ] Performance acceptable


## Code Analysis

Perform advanced code analysis with multiple inspection options.

### Analysis Menu:

#### 1. Knowledge Graph Generation
- Map relationships between components
- Visualize dependencies
- Identify architectural patterns

#### 2. Code Quality Evaluation
- Complexity metrics
- Maintainability index
- Technical debt assessment
- Code duplication detection

#### 3. Performance Analysis
- Identify bottlenecks
- Memory usage patterns
- Algorithm complexity
- Database query optimization

#### 4. Security Review
- Vulnerability scanning
- Input validation checks
- Authentication/authorization review
- Sensitive data handling

#### 5. Architecture Review
- Design pattern adherence
- SOLID principles compliance
- Coupling and cohesion analysis
- Module boundaries

#### 6. Test Coverage Analysis
- Coverage percentages
- Untested code paths
- Test quality assessment
- Missing edge cases

### Process:
1. Select analysis type based on need
2. Run appropriate tools and inspections
3. Generate comprehensive report
4. Provide actionable recommendations
5. Prioritize improvements by impact

### Output Format:
- Executive summary
- Detailed findings
- Risk assessment
- Improvement roadmap
- Code examples where relevant