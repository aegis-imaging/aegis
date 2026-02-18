# Copilot Instructions

- Product name expansion: use **Anonymization & Exchange Gateway for Imaging Studies**.
- Preserve the acronym **AEGIS**.
- Keep wording consistent across user-facing UI copy and documentation.
- Prefer concise, implementation-focused responses when editing this repository.

## Git Branching Strategy

- **Never commit directly to `develop` or `main`.**
- Always create a feature branch from `origin/develop`:
  ```bash
  git checkout -b feature/your-feature-name origin/develop
  ```
- Build and commit on the feature branch, then push it and open a PR to `develop`.
- This avoids merge conflicts when multiple agents work in parallel.