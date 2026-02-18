# Copilot Instructions

- Product name expansion: use **Anonymization & Exchange Gateway for Imaging Studies**.
- Preserve the acronym **AEGIS**.
- Keep wording consistent across user-facing UI copy and documentation.
- Prefer concise, implementation-focused responses when editing this repository.

## Git Workflow — Full Feature Lifecycle

Never commit directly to `develop` or `main`.

**Start a feature:**
```bash
git checkout -b feature/your-feature-name origin/develop
```

**Closing process (run after every feature):**
```bash
git add <files>
git commit -m "..."
git push -u origin feature/your-feature-name
gh pr create --base develop --head feature/your-feature-name --title "..." --body "..."
gh pr merge <number> --merge --delete-branch
git checkout develop && git pull
# then branch again for the next feature
```