---
name: swarmion-commit
description: Use when creating a git commit for Swarmion or Protos work where the commit message should be detailed, project-authored, and non-agentic.
---

# Swarmion Commit

When asked to commit changes:

1. Review the staged and unstaged diff before committing.
2. Stage only the files that belong to the requested change.
3. Write a normal project commit message, as if from a maintainer.

Message style:

- Use an imperative subject line, specific enough to identify the change.
- Include a body when the change is non-trivial.
- Explain what changed, why it changed, and any meaningful verification.
- Do not mention Codex, agents, AI, generated work, prompts, or tooling unless the change itself is about those topics.
- Do not add agentic trailers such as `Generated-by`, `Co-authored-by`, or similar metadata.
