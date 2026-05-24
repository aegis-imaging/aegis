# Credits

## Third-party software

Open-source dependencies are credited in the `go.mod`, `package.json`, and
`requirements.txt` files of the respective services. Their licenses live in
the `vendor/` and `node_modules/` directories at build time.

## UX design inspiration

The XNAT-style **Project → Subject → Study** navigation in the AEGIS admin
dashboard was inspired by the XNAT project's longstanding UI conventions.

- Project: XNAT — Extensible Neuroimaging Archive Toolkit
- Maintainer: Neuroinformatics Research Group, Washington University School of Medicine
- Website: https://www.xnat.org
- Source: https://github.com/NrgXnat/xnat
- License: 2-Clause BSD (Simplified BSD), Copyright (c) 2019, Washington University School of Medicine

**No XNAT source code is incorporated into AEGIS.** The XNAT project served
as a conceptual reference for how working neuroimaging researchers expect to
navigate imaging data — what they see post-login, how projects expand into
subject tables, how subject pages reveal imaging sessions over time, and how
top-bar search spans projects/subjects/sessions. All AEGIS code (React
components, Go HTTP handlers, SQL queries, CSS) was written fresh against
the AEGIS codebase and its conventions.

We are grateful to the XNAT maintainers and the wider neuroimaging community
for establishing the patterns that make our analyst-facing UI feel familiar.
