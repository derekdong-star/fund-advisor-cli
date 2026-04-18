# RULE

## Start Of Session
- At the start of each new task or new thread, read `RULE.md` first, then read `THREAD_HANDOFF.md`.

## Git Workflow
- Before any `git push`, always check whether the remote branch has new commits.
- If the remote has moved, sync first with `git pull --rebase` before pushing.
- Prefer `rebase` over `merge` when updating local `main`.
- Do not include unrelated generated files or handoff notes in a commit unless explicitly intended.
- Before each commit, review `README.md` and decide whether any new capability, command, or workflow change should be added, and simplify outdated or redundant wording when needed.
